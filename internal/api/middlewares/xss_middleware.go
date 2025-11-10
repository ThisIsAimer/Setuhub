package middlewares

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"hackathon/pkg/utils"

	"github.com/microcosm-cc/bluemonday"
)

func XSSMiddleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// sanitize path-----------------------------------------------------------------------------------------
		sanitizedPath, myErr := clean(r.URL.Path)
		if myErr.MyError != nil {
			utils.WriteJSONError(w, "Invalid path", http.StatusInternalServerError)
			return
		}

		// sanitize quary params-------------------------------------------------------------
		params := r.URL.Query()

		sanitizedQuery := make(map[string][]string)

		for k, values := range params {
			sanitizedKey, myErr := clean(k)
			if myErr.MyError != nil {
				utils.WriteJSONError(w, "Query key is invalid", http.StatusInternalServerError)
				return
			}

			var sanatizedValues []string
			for _, v := range values {
				cleanValue, myErr := clean(v)
				if myErr.MyError != nil {
					utils.WriteJSONError(w, "Query value is invalid", http.StatusInternalServerError)
					return
				}
				sanatizedValues = append(sanatizedValues, cleanValue.(string))
			}
			sanitizedQuery[sanitizedKey.(string)] = sanatizedValues
		}

		r.URL.Path = sanitizedPath.(string)

		r.URL.RawQuery = url.Values(sanitizedQuery).Encode()

		//sanitize body-------------------------------------------------------------------------------

		if r.Header.Get("Content-Type") == "application/json" {

			if r.Body != nil {
				bodyBytes, err := io.ReadAll(r.Body)
				if err != nil {
					myErr := utils.ErrorHandler(err, "Error reading request body", http.StatusUnsupportedMediaType)
					utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
					return
				}

				bodyString := strings.TrimSpace(string(bodyBytes))

				if len(bodyString) > 0 {

					// this will unmartial any kind of json data
					var inputData any
					err := json.NewDecoder(bytes.NewReader([]byte(bodyString))).Decode(&inputData)
					if err != nil {
						myErr := utils.ErrorHandler(err, "Invalid json body in xss", http.StatusUnsupportedMediaType)
						utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
						return
					}

					sanitizedData, myErr := clean(inputData)
					if myErr.MyError != nil {
						utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
						return
					}

					// martial to json body

					sanitizedBody, err := json.Marshal(sanitizedData)
					if err != nil {
						utils.WriteJSONError(w, "Error martialing sanitized data", http.StatusInternalServerError)
						return
					}

					r.Body = io.NopCloser(bytes.NewReader(sanitizedBody))

				}
			}

		} else if r.Header.Get("Content-Type") != "" {
			myErr := utils.ErrorHandler(fmt.Errorf("non application/json body"), "Unsupported json body", http.StatusUnsupportedMediaType)
			utils.WriteJSONError(w, myErr.MyError.Error(), myErr.Status)
			return
		}

		next.ServeHTTP(w, r)
	})

}

//clean sanitizes input data

func clean(data any) (any, utils.Errorhandler) {

	switch d := data.(type) {
	case map[string]any:
		for k, v := range d {
			d[k] = sanitizeValue(v)
		}

		return d, utils.Errorhandler{}

	case []any:
		for i, v := range d {
			d[i] = sanitizeValue(v)
		}

		return d, utils.Errorhandler{}

	case string:
		return sanitizeString(d), utils.Errorhandler{}

	default:
		return nil, utils.ErrorHandler(fmt.Errorf("unsupported type: %T", data), fmt.Sprintf("Unsupported type: %T", data), http.StatusUnsupportedMediaType)
	}

}

func sanitizeValue(value any) any {

	switch d := value.(type) {

	case map[string]any:
		for k, v := range d {
			d[k] = sanitizeValue(v)
		}
		return d
	case []any:
		for i, v := range d {
			d[i] = sanitizeValue(v)
		}
		return d

	case float64, int, bool, nil:
		return d

	default:
		return d
	}
}

func sanitizeString(value string) string {

	return bluemonday.UGCPolicy().Sanitize(value)
}
