// Package app contains auth application services.
package app

import (
	"context"
	"encoding/base64"
	"fmt"
	"jobdeck/services/auth/internal/domain"
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
)

// AuthRepository persists authentication client data.
//
//go:generate moq -out gen_auth_repository_moq_test.go . AuthRepository
type AuthRepository interface {
	GetLocalAuthClient(ctx context.Context, clientID string) (domain.LocalAuthClient, error)
	UpdateLocalAuthClientSecretHash(ctx context.Context, actorID domain.ActorId, secretHash string) error
}

// AuthStore provides authentication repositories and transaction boundaries.
//
//go:generate moq -out gen_auth_store_moq_test.go . AuthStore
type AuthStore interface {
	Auths() AuthRepository
	WithinAuthTx(ctx context.Context, fn func(ctx context.Context, auths AuthRepository) error) error
}

// AccessToken is an issued bearer token and its lifetime.
type AccessToken struct {
	Token        string
	TokenType    string // "Bearer"
	ExpiresInSec int64
}

// TokenIssuer issues access tokens for actors.
//
//go:generate moq -out gen_token_issuer_moq_test.go . TokenIssuer
type TokenIssuer interface {
	Token(actorID domain.ActorId, actorType string, audience string, now time.Time) (AccessToken, error)
}

// AuthUsecase authenticates clients and issues tokens.
type AuthUsecase struct {
	store   AuthStore
	issuer  TokenIssuer
	factory PHCBuilderFactory
}

// NewAuthUsecase creates an authentication use case.
func NewAuthUsecase(store AuthStore, issuer TokenIssuer, factory PHCBuilderFactory) *AuthUsecase {
	return &AuthUsecase{
		store:   store,
		issuer:  issuer,
		factory: factory,
	}
}

// Authenticate verifies a client secret and issues an access token.
func (u *AuthUsecase) Authenticate(ctx context.Context, clientID string, clientSecret string, audience string) (AccessToken, error) {
	client, err := u.store.Auths().GetLocalAuthClient(ctx, clientID)
	if err != nil {
		return AccessToken{}, err
	}
	if !client.Active {
		return AccessToken{}, errors.New("client is deactivated")
	}
	phc, err := parsePHCString(client.SecretHash)
	if err != nil {
		return AccessToken{}, err
	}
	builder, err := u.factory.FromPHC(phc)
	if err != nil {
		return AccessToken{}, err
	}
	if !builder.Compare([]byte(clientSecret), phc.Hash) {
		return AccessToken{}, errors.New("failed to authenticate")
	}

	newBuilder := u.factory.NewBuilder(phc.Salt)
	// update phc string with new builder
	currentSecretHash := builder.PHC([]byte(clientSecret)).String()
	newSecretHash := newBuilder.PHC([]byte(clientSecret)).String()
	if currentSecretHash != newSecretHash {
		err := u.store.WithinAuthTx(ctx, func(ctx context.Context, auths AuthRepository) error {
			return auths.UpdateLocalAuthClientSecretHash(ctx, client.ActorId, newSecretHash)
		})
		if err != nil {
			return AccessToken{}, err
		}
	}

	return u.issuer.Token(client.ActorId, client.Type, audience, time.Now())
}

// PHCBuilderFactory creates password hash builders.
type PHCBuilderFactory interface {
	NewBuilder(salt []byte) PHCBuilder
	FromPHC(phc PHC) (PHCBuilder, error)
}

// PHCBuilder hashes and compares passwords.
type PHCBuilder interface {
	PHC(password []byte) PHC
	Compare(password []byte, currentHash []byte) bool
}

// PHC is a parsed password hashing container.
type PHC struct {
	Algorithm string
	Version   uint32
	Params    string
	Salt      []byte
	Hash      []byte
}

func (p PHC) String() string {
	encodedSalt := base64.RawStdEncoding.EncodeToString(p.Salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(p.Hash)
	return fmt.Sprintf("$%s$v=%d$%s$%s$%s", p.Algorithm, p.Version, p.Params, encodedSalt, encodedHash)
}

func parsePHCString(phc string) (PHC, error) {
	fields := strings.Split(phc, "$")
	if len(fields) != 6 {
		return PHC{}, errors.New("invalid phc string")
	}
	versionStr := strings.Split(fields[2], "=")
	if len(versionStr) != 2 {
		return PHC{}, errors.New("invalid phc string")
	}
	version, err := strconv.ParseUint(versionStr[1], 10, 32)
	if err != nil {
		return PHC{}, err
	}
	decodedSalt, err := base64.RawStdEncoding.DecodeString(fields[4])
	if err != nil {
		return PHC{}, err
	}
	decodedHash, err := base64.RawStdEncoding.DecodeString(fields[5])
	if err != nil {
		return PHC{}, err
	}
	return PHC{
		Algorithm: fields[1],
		Version:   uint32(version),
		Params:    fields[3],
		Salt:      decodedSalt,
		Hash:      decodedHash,
	}, nil
}
