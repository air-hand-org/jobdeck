package token

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestJWTIssuerTokenContainsExpectedClaims(t *testing.T) {
	t.Parallel()

	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	issuer, err := NewJWTIssuer(base64.StdEncoding.EncodeToString(privateKey))
	require.NoError(t, err)
	now := time.Date(2026, time.July, 30, 0, 0, 0, 0, time.UTC)
	actorID := uuid.New()

	tests := []struct {
		name            string
		actor_type      string
		expiresDuration time.Duration
	}{
		{
			name:            "user",
			actor_type:      "USER",
			expiresDuration: 60 * time.Minute,
		},
		{
			name:            "service",
			actor_type:      "SERVICE",
			expiresDuration: 10 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accessToken, err := issuer.Token(actorID, tt.actor_type, "jobdeck-job-api", now)
			require.NoError(t, err)
			require.Equal(t, "Bearer", accessToken.TokenType)
			require.Equal(t, int64(tt.expiresDuration.Seconds()), accessToken.ExpiresInSec)

			claims := ActorClaims{}
			parsed, err := jwt.ParseWithClaims(accessToken.Token, &claims, func(token *jwt.Token) (any, error) {
				require.IsType(t, jwt.SigningMethodEdDSA, token.Method)
				return publicKey, nil
			}, jwt.WithTimeFunc(func() time.Time { return now }))
			require.NoError(t, err)
			require.True(t, parsed.Valid)
			require.Equal(t, actorID.String(), claims.Subject)
			require.Equal(t, tt.actor_type, claims.ActorType)
			require.Equal(t, []string{"jobdeck-job-api"}, []string(claims.Audience))
			require.True(t, claims.IssuedAt.Equal(now))
			require.True(t, claims.ExpiresAt.Equal(now.Add(tt.expiresDuration)))
		})
	}
}
