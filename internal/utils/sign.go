package utils

import (
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	Sub      string  `json:"sub"`
	Username string  `json:"username"`
	Jti      *string `json:"jti,omitempty"`
	Iat      int64   `json:"iat"`
	Exp      int64   `json:"exp"`
	jwt.RegisteredClaims
}

func IssueNewTokens(userId, username string, privateKey *rsa.PrivateKey) (string, string, string, error) {
	issueAt := time.Now().Unix()
	expAccess := issueAt + 3600
	expRefresh := issueAt + 7*24*3600
	jti := uuid.New().String()

	// access token
	accessClaims := &Claims{
		Sub:      userId,
		Username: username,
		Iat:      issueAt,
		Exp:      expAccess,
	}

	access, err := GenerateSign(accessClaims, privateKey)
	if err != nil {
		return "", "", "", err
	}

	// refresh token
	refreshClaims := &Claims{
		Sub:      userId,
		Username: username,
		Jti:      &jti,
		Iat:      issueAt,
		Exp:      expRefresh,
	}
	refresh, err := GenerateSign(refreshClaims, privateKey)
	if err != nil {
		return "", "", "", err
	}

	return access, refresh, jti, nil

}

func GenerateSign(claims *Claims, privateKey *rsa.PrivateKey) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"sub":      claims.Sub,
		"username": claims.Username,
		"jti":      &claims.Jti,
		"iat":      claims.Iat,
		"exp":      claims.Exp,
	})

	return token.SignedString(privateKey)
}

func ParseAndVerifySign(token string, pubKey *rsa.PublicKey) (*Claims, error) {
	parsedToken, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return pubKey, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := parsedToken.Claims.(*Claims)
	if !ok || !parsedToken.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	if time.Unix(claims.Exp, 0).Before(time.Now()) {
		return nil, fmt.Errorf("token expired")
	}

	return claims, nil
}
