// Package models provides shared data types for the API.
package models

// PaginatedResponse wraps a slice of results with pagination metadata.
type PaginatedResponse[T any] struct {
	Data       []T  `json:"data"`
	TotalCount int  `json:"total_count"`
	Offset     int  `json:"offset"`
	Limit      int  `json:"limit"`
	HasMore    bool `json:"has_more"`
}

// ErrorDetail contains error code and message.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse is the standard API error envelope.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}
