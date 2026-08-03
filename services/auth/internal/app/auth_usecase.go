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

//go:generate moq -out gen_auth_repository_moq_test.go . AuthRepository
type AuthRepository interface {
	GetLocalAuthClient(ctx context.Context, client_id string) (domain.LocalAuthClient, error)
	UpdateLocalAuthClientSecretHash(ctx context.Context, actor_id domain.ActorId, secret_hash string) error
}

//go:generate moq -out gen_auth_store_moq_test.go . AuthStore
type AuthStore interface {
	Auths() AuthRepository
	WithinAuthTx(ctx context.Context, fn func(ctx context.Context, auths AuthRepository) error) error
}

type AccessToken struct {
	Token        string
	TokenType    string // "Bearer"
	ExpiresInSec int64
}

//go:generate moq -out gen_token_issuer_moq_test.go . TokenIssuer
type TokenIssuer interface {
	Token(actor_id domain.ActorId, actor_type string, audience string, now time.Time) (AccessToken, error)
}

type AuthUsecase struct {
	store   AuthStore
	issuer  TokenIssuer
	factory PHCBuilderFactory
}

func NewAuthUsecase(store AuthStore, issuer TokenIssuer, factory PHCBuilderFactory) *AuthUsecase {
	return &AuthUsecase{
		store:   store,
		issuer:  issuer,
		factory: factory,
	}
}

func (u *AuthUsecase) Authenticate(ctx context.Context, client_id string, client_secret string, audience string) (AccessToken, error) {
	client, err := u.store.Auths().GetLocalAuthClient(ctx, client_id)
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
	if !builder.Compare([]byte(client_secret), phc.Hash) {
		return AccessToken{}, errors.New("failed to authenticate")
	}

	new_builder := u.factory.NewBuilder(phc.Salt)
	// update phc string with new builder
	current_secret_hash := builder.PHC([]byte(client_secret)).String()
	new_secret_hash := new_builder.PHC([]byte(client_secret)).String()
	if current_secret_hash != new_secret_hash {
		err := u.store.WithinAuthTx(ctx, func(ctx context.Context, auths AuthRepository) error {
			return auths.UpdateLocalAuthClientSecretHash(ctx, client.ActorId, new_secret_hash)
		})
		if err != nil {
			return AccessToken{}, err
		}
	}

	return u.issuer.Token(client.ActorId, client.Type, audience, time.Now())
}

type PHCBuilderFactory interface {
	NewBuilder(salt []byte) PHCBuilder
	FromPHC(phc PHC) (PHCBuilder, error)
}

type PHCBuilder interface {
	PHC(password []byte) PHC
	Compare(password []byte, current_hash []byte) bool
}

type PHC struct {
	Algorithm string
	Version   uint32
	Params    string
	Salt      []byte
	Hash      []byte
}

func (p PHC) String() string {
	encoded_salt := base64.RawStdEncoding.EncodeToString(p.Salt)
	encoded_hash := base64.RawStdEncoding.EncodeToString(p.Hash)
	return fmt.Sprintf("$%s$v=%d$%s$%s$%s", p.Algorithm, p.Version, p.Params, encoded_salt, encoded_hash)
}

func parsePHCString(phc string) (PHC, error) {
	fields := strings.Split(phc, "$")
	if len(fields) != 6 {
		return PHC{}, errors.New("invalid phc string")
	}
	version_str := strings.Split(fields[2], "=")
	if len(version_str) != 2 {
		return PHC{}, errors.New("invalid phc string")
	}
	version, err := strconv.ParseUint(version_str[1], 10, 32)
	if err != nil {
		return PHC{}, err
	}
	decoded_salt, err := base64.RawStdEncoding.DecodeString(fields[4])
	if err != nil {
		return PHC{}, err
	}
	decoded_hash, err := base64.RawStdEncoding.DecodeString(fields[5])
	if err != nil {
		return PHC{}, err
	}
	return PHC{
		Algorithm: fields[1],
		Version:   uint32(version),
		Params:    fields[3],
		Salt:      decoded_salt,
		Hash:      decoded_hash,
	}, nil
}
