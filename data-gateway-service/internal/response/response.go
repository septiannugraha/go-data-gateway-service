package response

import (
	"encoding/json"
	"net/http"
)

// StandardResponse represents the standard API response format
type StandardResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// ErrorInfo contains error details
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Meta contains pagination and other metadata
type Meta struct {
	Page       int    `json:"page,omitempty"`
	PerPage    int    `json:"per_page,omitempty"`
	Total      int    `json:"total,omitempty"`
	TotalPages int    `json:"total_pages,omitempty"`
	RequestID  string `json:"request_id,omitempty"`
}

// Success sends a successful response
func Success(w http.ResponseWriter, data interface{}, meta *Meta) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := StandardResponse{
		Success: true,
		Data:    data,
		Meta:    meta,
	}

	json.NewEncoder(w).Encode(response)
}

// Error sends an error response
func Error(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := StandardResponse{
		Success: false,
		Error: &ErrorInfo{
			Code:    http.StatusText(statusCode),
			Message: message,
		},
	}

	json.NewEncoder(w).Encode(response)
}

// ErrorWithDetails sends an error response with additional details
func ErrorWithDetails(w http.ResponseWriter, message string, details string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := StandardResponse{
		Success: false,
		Error: &ErrorInfo{
			Code:    http.StatusText(statusCode),
			Message: message,
			Details: details,
		},
	}

	json.NewEncoder(w).Encode(response)
}
