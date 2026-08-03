package secret

import (
	"testing"

	"jobdeck/services/auth/internal/app"
)

func TestArgon2IDPHCBuilderRoundTrip(t *testing.T) {
	t.Parallel()

	password := []byte("correct horse battery staple")
	salt := []byte("0123456789abcdef")
	factory := NewPHCBuilderFactory()

	builder := factory.NewBuilder(salt)
	phc := builder.PHC(password)

	parsedBuilder, err := factory.FromPHC(phc)
	if err != nil {
		t.Fatalf("FromPHC() error = %v", err)
	}
	if !parsedBuilder.Compare(password, phc.Hash) {
		t.Fatal("Compare() = false, want true")
	}
	if parsedBuilder.Compare([]byte("incorrect password"), phc.Hash) {
		t.Fatal("Compare() = true for incorrect password, want false")
	}
	if got := parsedBuilder.PHC(password).String(); got != phc.String() {
		t.Errorf("PHC() = %q, want %q", got, phc.String())
	}
}

func TestArgon2IDPHCBuilderGeneratesSalt(t *testing.T) {
	t.Parallel()

	phc := NewPHCBuilderFactory().NewBuilder(nil).PHC([]byte("password"))
	if got := len(phc.Salt); got != 16 {
		t.Errorf("salt length = %d, want 16", got)
	}
	if got := len(phc.Hash); got != 32 {
		t.Errorf("hash length = %d, want 32", got)
	}
}

func TestArgon2IDPHCBuilderFromPHCRejectsInvalidPHC(t *testing.T) {
	t.Parallel()

	valid := app.PHC{
		Algorithm: "argon2id",
		Version:   19,
		Params:    "m=19456,t=2,p=1",
		Salt:      []byte("0123456789abcdef"),
		Hash:      make([]byte, 32),
	}

	tests := map[string]app.PHC{
		"unsupported algorithm": {
			Algorithm: "argon2i",
			Version:   valid.Version,
			Params:    valid.Params,
			Salt:      valid.Salt,
			Hash:      valid.Hash,
		},
		"unsupported version": {
			Algorithm: valid.Algorithm,
			Version:   16,
			Params:    valid.Params,
			Salt:      valid.Salt,
			Hash:      valid.Hash,
		},
		"missing parameter": {
			Algorithm: valid.Algorithm,
			Version:   valid.Version,
			Params:    "m=19456,t=2",
			Salt:      valid.Salt,
			Hash:      valid.Hash,
		},
		"duplicate parameter": {
			Algorithm: valid.Algorithm,
			Version:   valid.Version,
			Params:    "m=19456,m=19457,t=2,p=1",
			Salt:      valid.Salt,
			Hash:      valid.Hash,
		},
	}

	for name, phc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := newArgon2IDPHCBuilderFromPHC(phc); err == nil {
				t.Fatal("newArgon2IDPHCBuilderFromPHC() error = nil, want error")
			}
		})
	}
}
