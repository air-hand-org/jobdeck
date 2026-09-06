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

// ActorClaims contains the JWT claims issued for an actor.
type ActorClaims struct {
	jwt.RegisteredClaims

	ActorType domain.ActorType `json:"actor_type"`
}

var _ app.TokenVerifier = (*JWTVerifier)(nil)

// JWTVerifier validates actor JWTs.
type JWTVerifier struct {
	publicKey ed25519.PublicKey
	issuer    string
	audience  string
}

// Verify validates a token and returns its identity.
func (v JWTVerifier) Verify(_ context.Context, rawToken string) (domain.AuthenticatedIdentity, error) {
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

	actorID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return domain.AuthenticatedIdentity{}, domain.ErrTokenInvalid
	}

	return domain.AuthenticatedIdentity{
		ID:   actorID,
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

// NewJWTVerifier creates a verifier from a base64-encoded Ed25519 public key.
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
