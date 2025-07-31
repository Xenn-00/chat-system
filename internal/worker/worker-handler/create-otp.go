package worker_handler

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	worker_service "github.com/xenn00/chat-system/internal/worker/worker-service"
)

type createUserOTPPayload struct {
	UserId    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

func HandlerCreateUserOTP(ctx context.Context, redis *redis.Client, raw json.RawMessage) error {
	var payload createUserOTPPayload

	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("invalid otp payload: %w", err)
	}

	otp := generateOTP()
	log.Info().Msgf("generated otp %s for user %s", otp, payload.UserId)

	key := fmt.Sprintf("otp:%s", payload.UserId)
	if err := redis.Set(ctx, key, otp, time.Minute*5).Err(); err != nil {
		return fmt.Errorf("failed to save otp: %w", err)
	}

	return worker_service.SendMailTrapOTP(payload.UserId, otp)
}

func generateOTP() string {
	b := make([]byte, 3)
	_, err := rand.Read(b)
	if err != nil {
		return "000000"
	}

	num := int(b[0])<<16 | int(b[1])<<8 | int(b[2])
	otp := num % 1000000
	return fmt.Sprintf("%06d", otp)
}
