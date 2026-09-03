// Package token issues and validates auth tokens.
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

// ActorClaims contains JWT claims for an actor.
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

// JWTIssuer issues signed JWTs.
type JWTIssuer struct {
	privateKey ed25519.PrivateKey
}

var _ app.TokenIssuer = (*JWTIssuer)(nil)

// Token issues a signed access token.
func (i JWTIssuer) Token(actorID domain.ActorId, actorType string, audience string, now time.Time) (app.AccessToken, error) {
	claims := ActorClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "jobdeck-auth",
			Subject:   actorID.String(),
			Audience:  []string{audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(i.expiresDuration(actorType))),
		},
		ActorType: actorType,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	signedJWT, err := token.SignedString(i.privateKey)
	if err != nil {
		return app.AccessToken{}, err
	}
	return app.AccessToken{
		Token:        signedJWT,
		TokenType:    "Bearer",
		ExpiresInSec: int64(i.expiresDuration(actorType).Seconds()),
	}, nil
}

func (i JWTIssuer) expiresDuration(actorType string) time.Duration {
	switch actorType {
	case "USER":
		return 60 * time.Minute
	default:
		return 10 * time.Minute
	}
}

// NewJWTIssuer creates a JWT issuer from an encoded private key.
func NewJWTIssuer(encodedPrivateKey string) (*JWTIssuer, error) {
	privateKey, err := decodePrivateKey(encodedPrivateKey)
	if err != nil {
		return nil, err
	}

	return &JWTIssuer{
		privateKey: privateKey,
	}, nil
}
