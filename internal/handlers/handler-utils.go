package handlers

import (
	"fmt"
	"net/http"

	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog/log"
	"github.com/xenn00/chat-system/internal/dtos"
	app_error "github.com/xenn00/chat-system/internal/errors"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

type HandlerFunc func(w http.ResponseWriter, r *http.Request) *app_error.AppError

func WrapHandler(fn HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			log.Error().Err(err).Msg(fmt.Sprintf("error occur, request id: %s", r.Header.Get("X-Request-ID")))
			writeJSON(w, err.Code, map[string]any{
				"message": "Error occur",
				"errors": map[string]any{
					"code":    err.Code,
					"field":   err.Field,
					"message": err.Message,
				},
				"data":       nil,
				"request_id": r.Header.Get("X-Request-ID"),
			})
		}
	}
}

func CreateResponse[T any](message string, data T, requestId string) dtos.Response[T] {
	return dtos.Response[T]{
		Message:   message,
		Data:      data,
		RequestID: requestId,
	}
}
