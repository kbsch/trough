package api

import (
	"encoding/json"
	"errors"
	"net/http"
)

// APIError represents an API error with HTTP status code
type APIError struct {
	Code       int    `json:"-"`
	Message    string `json:"error"`
	Detail     string `json:"detail,omitempty"`
	RequestID  string `json:"request_id,omitempty"`
}

func (e *APIError) Error() string {
	return e.Message
}

// Common API errors
var (
	ErrNotFound          = &APIError{Code: http.StatusNotFound, Message: "Resource not found"}
	ErrBadRequest        = &APIError{Code: http.StatusBadRequest, Message: "Bad request"}
	ErrInternalServer    = &APIError{Code: http.StatusInternalServerError, Message: "Internal server error"}
	ErrUnauthorized      = &APIError{Code: http.StatusUnauthorized, Message: "Unauthorized"}
	ErrForbidden         = &APIError{Code: http.StatusForbidden, Message: "Forbidden"}
	ErrTooManyRequests   = &APIError{Code: http.StatusTooManyRequests, Message: "Too many requests"}
	ErrServiceUnavailable = &APIError{Code: http.StatusServiceUnavailable, Message: "Service unavailable"}
)

// NewAPIError creates a new API error
func NewAPIError(code int, message string) *APIError {
	return &APIError{Code: code, Message: message}
}

// NewNotFoundError creates a not found error with detail
func NewNotFoundError(detail string) *APIError {
	return &APIError{
		Code:    http.StatusNotFound,
		Message: "Resource not found",
		Detail:  detail,
	}
}

// NewBadRequestError creates a bad request error with detail
func NewBadRequestError(detail string) *APIError {
	return &APIError{
		Code:    http.StatusBadRequest,
		Message: "Bad request",
		Detail:  detail,
	}
}

// NewValidationError creates a validation error
func NewValidationError(field, message string) *APIError {
	return &APIError{
		Code:    http.StatusBadRequest,
		Message: "Validation error",
		Detail:  field + ": " + message,
	}
}

// NewInternalError creates an internal server error with detail
func NewInternalError(detail string) *APIError {
	return &APIError{
		Code:    http.StatusInternalServerError,
		Message: "Internal server error",
		Detail:  detail,
	}
}

// WriteError writes an API error to the response
func WriteError(w http.ResponseWriter, err error, requestID string) {
	var apiErr *APIError

	if errors.As(err, &apiErr) {
		// It's already an API error
		apiErr.RequestID = requestID
	} else {
		// Wrap it in an internal error
		apiErr = &APIError{
			Code:      http.StatusInternalServerError,
			Message:   "Internal server error",
			RequestID: requestID,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(apiErr.Code)
	json.NewEncoder(w).Encode(apiErr)
}

// ErrorResponse is a helper to write error responses
type ErrorResponse struct {
	Error     string `json:"error"`
	Detail    string `json:"detail,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// WriteJSON writes a JSON response
func WriteJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

// WriteSuccess writes a success response
func WriteSuccess(w http.ResponseWriter, data interface{}) {
	WriteJSON(w, http.StatusOK, data)
}
