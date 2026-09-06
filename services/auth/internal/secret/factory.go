package secret

import "jobdeck/services/auth/internal/app"

// PHCBuilderFactory creates password hashing builders.
type PHCBuilderFactory struct{}

var _ app.PHCBuilderFactory = (*PHCBuilderFactory)(nil)

// NewBuilder creates a password hashing builder using the provided salt.
func (f PHCBuilderFactory) NewBuilder(salt []byte) app.PHCBuilder {
	return newArgon2IDPHCBuilder(salt)
}

// FromPHC creates a password hashing builder from parsed PHC parameters.
func (f PHCBuilderFactory) FromPHC(phc app.PHC) (app.PHCBuilder, error) {
	return newArgon2IDPHCBuilderFromPHC(phc)
}

// NewPHCBuilderFactory creates a password hashing builder factory.
func NewPHCBuilderFactory() *PHCBuilderFactory {
	return &PHCBuilderFactory{}
}
