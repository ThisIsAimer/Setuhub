package databasehandler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hackathon/internal/models"
	"hackathon/internal/repositories/sqlconnect"
	"hackathon/pkg/utils"
	"log"
	"net/http"
	"strings"
	"time"
)

const expoSendURL = "https://exp.host/--/api/v2/push/send"
const expoReceiptsURL = "https://exp.host/--/api/v2/push/getReceipts"
const expoChunkSize = 100

// --- payload types used for Expo API ---
type expoMessage struct {
	To    string            `json:"to"`
	Title string            `json:"title,omitempty"`
	Body  string            `json:"body,omitempty"`
	Data  map[string]string `json:"data,omitempty"`
	Sound string            `json:"sound,omitempty"`
}

type expoTicket struct {
	ID     string `json:"id,omitempty"`
	Status string `json:"status,omitempty"`
}

type expoSendResponse struct {
	Data   []expoTicket  `json:"data"`
	Errors []interface{} `json:"errors,omitempty"`
}

// receipts
type receiptDetails struct {
	Error string `json:"error,omitempty"`
}

type receipt struct {
	Status  string          `json:"status"`
	Message string          `json:"message,omitempty"`
	Details *receiptDetails `json:"details,omitempty"`
}

type receiptsResponse struct {
	Data map[string]receipt `json:"data"`
}

// --- helper: send a single Expo message (wraps /push/send with single message) ---
func sendToOneExpo(ctx context.Context, token string, title, body string, data map[string]string) error {
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("empty Expo token")
	}

	msg := expoMessage{
		To:    token,
		Title: title,
		Body:  body,
		Data:  data,
	}

	payload, _ := json.Marshal([]expoMessage{msg}) // Expo accepts array of messages
	req, _ := http.NewRequestWithContext(ctx, "POST", expoSendURL, bytes.NewBuffer(payload))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var sendResp expoSendResponse
	if err := json.NewDecoder(resp.Body).Decode(&sendResp); err != nil {
		return fmt.Errorf("decode expo send response: %w", err)
	}

	// Optionally: fetch receipts for the ticket id and remove invalid token if needed
	if len(sendResp.Data) > 0 && sendResp.Data[0].ID != "" {
		// small delay before receipts
		time.Sleep(2 * time.Second)
		ids := []string{sendResp.Data[0].ID}
		if receipts, rerr := getExpoReceipts(ctx, ids); rerr == nil {
			for _, rc := range receipts.Data {
				if rc.Status == "error" && rc.Details != nil && rc.Details.Error == "DeviceNotRegistered" {
					// remove token from DB
					fmt.Printf("badtokens found :%v", token)
				}
			}
		}
	}

	return nil
}

// --- helper: send many messages (chunked) ---
func sendToManyExpo(ctx context.Context, tokens []string, title, body string, data map[string]string) (string, error) {
	if len(tokens) == 0 {
		return "no tokens provided", nil
	}

	// build messages (one per token)
	messages := make([]expoMessage, 0, len(tokens))
	for _, t := range tokens {
		if strings.TrimSpace(t) == "" {
			continue
		}
		messages = append(messages, expoMessage{
			To:    t,
			Title: title,
			Body:  body,
			Data:  data,
		})
	}

	client := &http.Client{Timeout: 15 * time.Second}

	// collect ticketIDs -> token mapping so we can check receipts later
	tokenToTicket := make(map[string]string)

	// send in chunks of 100
	for i := 0; i < len(messages); i += expoChunkSize {
		end := i + expoChunkSize
		if end > len(messages) {
			end = len(messages)
		}
		chunk := messages[i:end]

		payload, _ := json.Marshal(chunk)
		req, _ := http.NewRequestWithContext(ctx, "POST", expoSendURL, bytes.NewBuffer(payload))
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return "", fmt.Errorf("expo send failed: %w", err)
		}
		var sendResp expoSendResponse
		if err := json.NewDecoder(resp.Body).Decode(&sendResp); err != nil {
			resp.Body.Close()
			return "", fmt.Errorf("decode send response: %w", err)
		}
		resp.Body.Close()

		// map tickets back to tokens (responses are index aligned)
		for idx, t := range sendResp.Data {
			token := chunk[idx].To
			tokenToTicket[token] = t.ID
		}
	}

	// gather ticket ids and check receipts
	var ticketIDs []string
	for _, tid := range tokenToTicket {
		if tid != "" {
			ticketIDs = append(ticketIDs, tid)
		}
	}

	// Wait a bit before checking receipts (Expo recommends a short wait)
	time.Sleep(2 * time.Second)

	// call receipts in chunks as well (Expo may limit request sizes)
	allInvalidTokens := make([]string, 0)
	for i := 0; i < len(ticketIDs); i += expoChunkSize {
		end := i + expoChunkSize
		if end > len(ticketIDs) {
			end = len(ticketIDs)
		}
		idsChunk := ticketIDs[i:end]
		receipts, err := getExpoReceipts(ctx, idsChunk)
		if err != nil {
			// non-fatal: log and continue
			log.Printf("getExpoReceipts error: %v", err)
			continue
		}

		// receipts.Data is map[ticketID]receipt
		for tid, rc := range receipts.Data {
			if rc.Status == "error" && rc.Details != nil {
				// find token for this ticket id
				var badToken string
				for tok, ticket := range tokenToTicket {
					if ticket == tid {
						badToken = tok
						break
					}
				}
				if badToken != "" {
					// handle specific error codes; DeviceNotRegistered -> remove token
					if rc.Details.Error == "DeviceNotRegistered" || strings.Contains(rc.Details.Error, "unregistered") {
						allInvalidTokens = append(allInvalidTokens, badToken)
					} else {
						// other errors can be logged for retry/backoff
						log.Printf("expo delivery error for token %s: %v", badToken, rc.Details.Error)
					}
				}
			}
		}
	}

	fmt.Printf("no.of badtokens found : %d, %v", len(allInvalidTokens), allInvalidTokens)

	return fmt.Sprintf("requested notifications for %d tokens; removed %d invalid tokens", len(tokens), len(allInvalidTokens)), nil
}

// --- receipts helper ---
func getExpoReceipts(ctx context.Context, ticketIDs []string) (*receiptsResponse, error) {
	if len(ticketIDs) == 0 {
		return &receiptsResponse{Data: map[string]receipt{}}, nil
	}
	body, _ := json.Marshal(map[string][]string{"ids": ticketIDs})
	req, _ := http.NewRequestWithContext(ctx, "POST", expoReceiptsURL, bytes.NewBuffer(body))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rr receiptsResponse
	if err := json.NewDecoder(resp.Body).Decode(&rr); err != nil {
		return nil, fmt.Errorf("decode receipts: %w", err)
	}
	return &rr, nil
}

// --- adapted sendNotification (single user) ---
func sendNotification(uuid string) {
	db, err := sqlconnect.ConnectDB()
	if err != nil {
		utils.ErrorHandler(err, "unable to notify")
		return
	}
	defer db.Close()

	var expoToken string
	query := `SELECT expo_token FROM users WHERE uuid = $1;`
	err = db.QueryRow(query, uuid).Scan(&expoToken)
	if err != nil {
		utils.ErrorHandler(err, "unable to notify")
		return
	}
	if strings.TrimSpace(expoToken) == "" {
		// nothing to do
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	err = sendToOneExpo(ctx, expoToken,
		"someone has shown interest in your request!",
		"A user on the way",
		map[string]string{"link": "https://yourapp.com/", "type": "information"},
	)
	if err != nil {
		utils.ErrorHandler(err, "unable to notify")
		return
	}
}

// --- adapted sendNotifications (many users in area) ---
func sendNotifications(post models.Post, noti chan string) {
	db, err := sqlconnect.ConnectDB()
	if err != nil {
		myErr := utils.ErrorHandler(err, "unable to notify")
		noti <- myErr.Error()
		return
	}
	defer db.Close()

	// select expo_token from users within radius
	query := `SELECT expo_token FROM users WHERE ST_DWithin(coordinates, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $3);`

	rows, err := db.Query(query, post.Longitude, post.Latitude, post.Radius)
	if err != nil {
		myErr := utils.ErrorHandler(err, "unable to notify")
		noti <- myErr.Error()
		return
	}
	defer rows.Close()

	tokens := make([]string, 0)
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			myErr := utils.ErrorHandler(err, "unable to notify")
			noti <- myErr.Error()
			return
		}
		if token != "" {
			tokens = append(tokens, token)
		}
	}
	if rows.Err() != nil {
		myErr := utils.ErrorHandler(rows.Err(), "unable to notify")
		noti <- myErr.Error()
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	notified, err := sendToManyExpo(ctx, tokens,
		fmt.Sprintf("New %s request: %s", post.Type, post.Title),
		post.Description,
		map[string]string{"link": fmt.Sprintf("https://yourapp.com/%s", post.PostUUID), "type": post.Type},
	)
	if err != nil {
		myErr := utils.ErrorHandler(err, "unable to notify")
		noti <- myErr.Error()
		return
	}

	noti <- notified
}
