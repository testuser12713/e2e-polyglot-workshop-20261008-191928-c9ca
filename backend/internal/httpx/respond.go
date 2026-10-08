// Package httpx holds the shared HTTP response helpers.
//
// It lives below internal/httpapi and the feature handler packages so both can
// import it without an import cycle (httpapi imports the handler packages, so
// the handler packages cannot import httpapi).
package httpx

import (
	"encoding/json"
	"log"
	"net/http"
)

// ErrorBody is the single error envelope every error response uses (AC-21).
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail is the code/message pair inside ErrorBody.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// JSON writes v as a JSON body with the given status code.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write JSON response: %v", err)
	}
}

// Error writes the uniform error body with the given status and code.
func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, ErrorBody{Error: ErrorDetail{Code: code, Message: message}})
}

// NotFound writes the uniform 404 body for an unknown path.
func NotFound(w http.ResponseWriter) {
	Error(w, http.StatusNotFound, "not_found", "resource not found")
}

// NotImplemented writes the uniform 501 body for a route another ticket owns.
// A stub answers 501, never 500.
func NotImplemented(w http.ResponseWriter, feature string) {
	Error(w, http.StatusNotImplemented, "not_implemented", feature+" is not implemented yet")
}
