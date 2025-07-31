package app_error

type AppError struct {
	Code    int    `json:"-"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

func (e AppError) Error() string {
	return e.Message
}

func NewAppError(code int, msg, field string) *AppError {
	return &AppError{
		Code:    code,
		Message: msg,
		Field:   field,
	}
}
