package repository

import (
	"jobdeck/services/auth/internal/domain"
	authdb "jobdeck/services/auth/internal/repository/gen"
)

func localAuthClientFromDB(input authdb.AuthLocalAuthClientDetail) domain.LocalAuthClient {
	return domain.LocalAuthClient{
		ActorId:    input.ActorID,
		ClientId:   input.ClientID,
		SecretHash: input.SecretHash,
		Name:       input.Name,
		Type:       input.Type,
		Active:     input.Active,
	}
}
