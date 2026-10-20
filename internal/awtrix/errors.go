package awtrix

import (
	"encoding/json"
	"fmt"
	"strings"
)

// HTTPError preserves status information for callers such as atomic deployment.
type HTTPError struct {
	StatusCode int
	Status     string
	Method     string
	Path       string
	Code       string
	Message    string
	Field      string
}

func (e *HTTPError) Error() string {
	message := fmt.Sprintf("%s %s: AWTRIX returned %s", e.Method, e.Path, e.Status)
	if e.Code != "" {
		message += " (" + e.Code + ")"
	}
	if e.Message != "" {
		message += ": " + e.Message
	}
	if e.Field != "" {
		message += " [field=" + e.Field + "]"
	}
	return message
}

func responseError(method, path string, statusCode int, status string, body []byte) *HTTPError {
	err := &HTTPError{StatusCode: statusCode, Status: status, Method: method, Path: path}
	var payload struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Field   string `json:"field"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &payload) == nil && (payload.Error.Code != "" || payload.Error.Message != "") {
		err.Code = errorDetail(payload.Error.Code)
		err.Message = errorDetail(payload.Error.Message)
		err.Field = errorDetail(payload.Error.Field)
	} else {
		err.Message = errorDetail(string(body))
	}
	return err
}

func errorDetail(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 512 {
		s = s[:512] + "..."
	}
	return s
}
