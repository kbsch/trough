package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

// APIError represents an error response
type APIError struct {
	Error     string `json:"error"`
	Code      string `json:"code,omitempty"`
	Details   any    `json:"details,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// APIResponse is a generic success response wrapper
type APIResponse struct {
	Data any    `json:"data,omitempty"`
	Meta *Meta  `json:"meta,omitempty"`
}

// Meta contains pagination and other metadata
type Meta struct {
	Total      int `json:"total,omitempty"`
	Page       int `json:"page,omitempty"`
	PerPage    int `json:"per_page,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
}

// JSON writes a JSON response
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// Success writes a success response
func Success(w http.ResponseWriter, data any) {
	JSON(w, http.StatusOK, data)
}

// SuccessWithMeta writes a success response with metadata
func SuccessWithMeta(w http.ResponseWriter, data any, meta *Meta) {
	JSON(w, http.StatusOK, APIResponse{Data: data, Meta: meta})
}

// Created writes a 201 response
func Created(w http.ResponseWriter, data any) {
	JSON(w, http.StatusCreated, data)
}

// Accepted writes a 202 response
func Accepted(w http.ResponseWriter, data any) {
	JSON(w, http.StatusAccepted, data)
}

// NoContent writes a 204 response
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Error writes an error response
func Error(w http.ResponseWriter, r *http.Request, status int, message string) {
	requestID := middleware.GetReqID(r.Context())
	JSON(w, status, APIError{Error: message, RequestID: requestID})
}

// ErrorSimple writes an error response without request (for backwards compatibility)
func ErrorSimple(w http.ResponseWriter, status int, message string) {
	JSON(w, status, APIError{Error: message})
}

// ErrorWithCode writes an error response with an error code
func ErrorWithCode(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, APIError{Error: message, Code: code})
}

// BadRequest writes a 400 response
func BadRequest(w http.ResponseWriter, r *http.Request, message string) {
	Error(w, r, http.StatusBadRequest, message)
}

// NotFound writes a 404 response
func NotFound(w http.ResponseWriter, r *http.Request, message string) {
	if message == "" {
		message = "Resource not found"
	}
	Error(w, r, http.StatusNotFound, message)
}

// InternalError writes a 500 response
func InternalError(w http.ResponseWriter, r *http.Request, message string) {
	if message == "" {
		message = "Internal server error"
	}
	Error(w, r, http.StatusInternalServerError, message)
}

// TooManyRequests writes a 429 response
func TooManyRequests(w http.ResponseWriter, r *http.Request, message string) {
	if message == "" {
		message = "Too many requests"
	}
	Error(w, r, http.StatusTooManyRequests, message)
}
