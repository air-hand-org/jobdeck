package secret

import "jobdeck/services/auth/internal/app"

type phcBuilderFactory struct{}

var _ app.PHCBuilderFactory = (*phcBuilderFactory)(nil)

func (f phcBuilderFactory) NewBuilder(salt []byte) app.PHCBuilder {
	return newArgon2IDPHCBuilder(salt)
}

func (f phcBuilderFactory) FromPHC(phc app.PHC) (app.PHCBuilder, error) {
	return newArgon2IDPHCBuilderFromPHC(phc)
}

func NewPHCBuilderFactory() *phcBuilderFactory {
	return &phcBuilderFactory{}
}
