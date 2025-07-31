package state

import (
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
)

func InitSecret() (*JwtSecret, error) {
	privKeyBytes, err := os.ReadFile("private.pem")
	if err != nil {
		return nil, err
	}

	pubKeyBytes, err := os.ReadFile("public.pem")
	if err != nil {
		return nil, err
	}

	// decode & parse
	privKey, err := jwt.ParseRSAPrivateKeyFromPEM(privKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(pubKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("invalid public key: %w", err)
	}

	log.Info().Msg("JWT secret initialized successfully")
	return &JwtSecret{
		Private: privKey,
		Public: pubKey,
	}, nil
}