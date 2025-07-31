package dtos

type Response[T any] struct {
	Message   string         `json:"message"`
	Data      T              `json:"data"`
	RequestID string         `json:"request_id,omitempty"`
	Errors    *ErrorResponse `json:"errors,omitempty"`
}

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}
