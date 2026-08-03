package token

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"jobdeck/services/auth/internal/app"
	"jobdeck/services/auth/internal/domain"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type ActorClaims struct {
	jwt.RegisteredClaims
	ActorType string `json:"actor_type"`
}

func decodePrivateKey(encoded string) (ed25519.PrivateKey, error) {
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return nil, err
	}
	if len(decoded) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid Ed25519 private key")
	}
	return ed25519.PrivateKey(decoded), nil
}

type JWTIssuer struct {
	privateKey ed25519.PrivateKey
}

var _ app.TokenIssuer = (*JWTIssuer)(nil)

func (i JWTIssuer) Token(actor_id domain.ActorId, actor_type string, audience string, now time.Time) (app.AccessToken, error) {
	claims := ActorClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "jobdeck-auth",
			Subject:   actor_id.String(),
			Audience:  []string{audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(i.expiresDuration(actor_type))),
		},
		ActorType: actor_type,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	signed_jwt, err := token.SignedString(i.privateKey)
	if err != nil {
		return app.AccessToken{}, err
	}
	return app.AccessToken{
		Token:        signed_jwt,
		TokenType:    "Bearer",
		ExpiresInSec: int64(i.expiresDuration(actor_type).Seconds()),
	}, nil
}

func (i JWTIssuer) expiresDuration(actor_type string) time.Duration {
	switch actor_type {
	case "USER":
		return 60 * time.Minute
	default:
		return 10 * time.Minute
	}
}

func NewJWTIssuer(encoded_privatekey string) (*JWTIssuer, error) {
	privatekey, err := decodePrivateKey(encoded_privatekey)
	if err != nil {
		return nil, err
	}

	return &JWTIssuer{
		privateKey: privatekey,
	}, nil
}
