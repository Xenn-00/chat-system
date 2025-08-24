package app_error

import (
	"encoding/json"
	"net/http"
)

type AppError struct {
	Code    int    `json:"-"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

func (e AppError) Error() string {
	return e.Message
}

func (e AppError) JSON(w http.ResponseWriter) error {
	return json.NewEncoder(w).Encode(e)
}

func NewAppError(code int, msg, field string) *AppError {
	return &AppError{
		Code:    code,
		Message: msg,
		Field:   field,
	}
}
