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
		sanitizedPath, err := clean(r.URL.Path)
		if err != nil {
			http.Error(w, "invalid path", http.StatusInternalServerError)
			return
		}

		// sanitize quary params-------------------------------------------------------------
		params := r.URL.Query()

		sanitizedQuery := make(map[string][]string)

		for k, values := range params {
			sanitizedKey, err := clean(k)
			if err != nil {
				http.Error(w, "query key is invalid", http.StatusInternalServerError)
				return
			}

			var sanatizedValues []string
			for _, v := range values {
				cleanValue, err := clean(v)
				if err != nil {
					http.Error(w, "query value is invalid", http.StatusInternalServerError)
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
					http.Error(w, utils.ErrorHandler(err, "error reading request body").Error(), http.StatusUnsupportedMediaType)
					return
				}

				bodyString := strings.TrimSpace(string(bodyBytes))

				if len(bodyString) > 0 {

					// this will unmartial any kind of json data
					var inputData any
					err := json.NewDecoder(bytes.NewReader([]byte(bodyString))).Decode(&inputData)
					if err != nil {
						http.Error(w, utils.ErrorHandler(err, "invalid json body in xss").Error(), http.StatusUnsupportedMediaType)
						return
					}

					sanitizedData, err := clean(inputData)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}

					// martial to json body

					sanitizedBody, err := json.Marshal(sanitizedData)
					if err != nil {
						http.Error(w, "error martialing sanitized data", http.StatusInternalServerError)
						return
					}

					r.Body = io.NopCloser(bytes.NewReader(sanitizedBody))

				}
			}

		} else if r.Header.Get("Content-Type") != "" {
			myErr := utils.ErrorHandler(fmt.Errorf("non application/json body"), "unsupported json body")
			http.Error(w, myErr.Error(), http.StatusUnsupportedMediaType)
			return
		}

		next.ServeHTTP(w, r)
	})

}

//clean sanitizes input data

func clean(data any) (any, error) {

	switch d := data.(type) {
	case map[string]any:
		for k, v := range d {
			d[k] = sanitizeValue(v)
		}

		return d, nil

	case []any:
		for i, v := range d {
			d[i] = sanitizeValue(v)
		}

		return d, nil

	case string:
		return sanitizeString(d), nil

	default:
		return nil, utils.ErrorHandler(fmt.Errorf("unsupported type: %T", data), fmt.Sprintf("unsupported type: %T", data))
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
