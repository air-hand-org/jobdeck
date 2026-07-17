package auth

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"jobdeck/services/job-api/internal/app"
	"jobdeck/services/job-api/internal/domain"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type ActorClaims struct {
	jwt.RegisteredClaims

	ActorType domain.ActorType `json:"actor_type"`
}

var _ app.TokenVerifier = (*JWTVerifier)(nil)

type JWTVerifier struct {
	publicKey ed25519.PublicKey
	issuer    string
	audience  string
}

func (v JWTVerifier) Verify(ctx context.Context, rawToken string) (domain.AuthenticatedIdentity, error) {
	claims := ActorClaims{}
	token, err := jwt.ParseWithClaims(
		rawToken,
		&claims,
		func(_ *jwt.Token) (any, error) {
			return v.publicKey, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}),
		jwt.WithIssuer(v.issuer),
		jwt.WithAudience(v.audience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(30*time.Second),
	)

	if err != nil || !token.Valid {
		return domain.AuthenticatedIdentity{}, domain.ErrTokenInvalid
	}

	actor_id, err := uuid.Parse(claims.Subject)
	if err != nil {
		return domain.AuthenticatedIdentity{}, domain.ErrTokenInvalid
	}

	return domain.AuthenticatedIdentity{
		ID:   actor_id,
		Type: claims.ActorType,
	}, nil
}

func decodePublicKey(encoded string) (ed25519.PublicKey, error) {
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return nil, err
	}
	if len(decoded) != ed25519.PublicKeySize {
		return nil, errors.New("invalid Ed25519 public key")
	}
	return ed25519.PublicKey(decoded), nil
}

func NewJWTVerifier(pubkey string) (*JWTVerifier, error) {
	decoded, err := decodePublicKey(pubkey)
	if err != nil {
		return nil, err
	}
	return &JWTVerifier{
		publicKey: decoded,
		issuer:    "jobdeck-auth",
		audience:  "jobdeck-job-api",
	}, nil
}
