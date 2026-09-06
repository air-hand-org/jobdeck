// Package secret provides password hashing implementations.
package secret

import (
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"jobdeck/services/auth/internal/app"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

type argon2IDPHCBuilder struct {
	Salt        []byte
	Time        uint32
	Memory      uint32
	KeyLen      uint32
	Parallelism uint8
}

func (builder argon2IDPHCBuilder) PHC(password []byte) app.PHC {
	passwordHash := builder.hashPassword(password)
	return app.PHC{
		Algorithm: "argon2id",
		Version:   19,
		Params:    fmt.Sprintf("m=%d,t=%d,p=%d", builder.Memory, builder.Time, builder.Parallelism),
		Salt:      builder.Salt,
		Hash:      passwordHash,
	}
}

func (builder argon2IDPHCBuilder) Compare(password []byte, hash []byte) bool {
	derivedHash := builder.hashPassword(password)
	return subtle.ConstantTimeCompare(derivedHash, hash) == 1
}

func (builder argon2IDPHCBuilder) hashPassword(password []byte) []byte {
	return argon2.IDKey(password, builder.Salt, builder.Time, builder.Memory, builder.Parallelism, builder.KeyLen)
}

func newArgon2IDPHCBuilder(salt []byte) *argon2IDPHCBuilder {
	// https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html#introduction
	if salt == nil {
		salt = make([]byte, 16)
		rand.Read(salt)
	}
	return &argon2IDPHCBuilder{
		Salt:        salt,
		Time:        2,
		Memory:      19 * 1024,
		KeyLen:      32,
		Parallelism: 1,
	}
}

func newArgon2IDPHCBuilderFromPHC(phc app.PHC) (*argon2IDPHCBuilder, error) {
	if phc.Algorithm != "argon2id" {
		return nil, errors.New("invalid argon2id phc")
	}
	if phc.Version != 19 {
		return nil, errors.New("invalid argon2id phc")
	}
	var builder argon2IDPHCBuilder
	builder.Salt = phc.Salt
	builder.KeyLen = uint32(len(phc.Hash))

	// Params
	params := strings.Split(phc.Params, ",")
	seen := map[string]bool{}
	for _, p := range params {
		key, value, ok := strings.Cut(p, "=")
		if !ok || seen[key] {
			return nil, errors.New("invalid argon2id phc")
		}
		seen[key] = true

		switch key {
		case "m":
			n, err := strconv.ParseUint(value, 10, 32)
			if err != nil || n == 0 {
				return nil, errors.New("invalid argon2id phc")
			}
			builder.Memory = uint32(n)
		case "t":
			n, err := strconv.ParseUint(value, 10, 32)
			if err != nil || n == 0 {
				return nil, errors.New("invalid argon2id phc")
			}
			builder.Time = uint32(n)
		case "p":
			n, err := strconv.ParseUint(value, 10, 8)
			if err != nil || n == 0 {
				return nil, errors.New("invalid argon2id phc")
			}
			builder.Parallelism = uint8(n)
		default:
			return nil, errors.New("invalid argon2id phc")
		}
	}
	if !seen["m"] || !seen["t"] || !seen["p"] {
		return nil, errors.New("invalid argon2id phc")
	}
	return &builder, nil
}
