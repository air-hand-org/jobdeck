package migrations

import (
	"context"
	"database/sql"
	"os"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upSeedTestUserCredential, downSeedTestUserCredential)
}

var testUserActorID = uuid.MustParse("ffffffff-ffff-ffff-ffff-fffffffffffd")

func upSeedTestUserCredential(ctx context.Context, tx *sql.Tx) error {
	secretHash := os.Getenv("AUTH_TEST_USER_SECRET_HASH")
	if secretHash == "" {
		return nil
	}
	_, err := tx.ExecContext(
		ctx,
		`
		INSERT INTO auth.local_auth_client_credential(
			actor_id, client_id, secret_hash
		)
		VALUES ($1, $2, $3)
		`,
		testUserActorID,
		"test-user",
		secretHash,
	)
	if err != nil {
		return err
	}
	return nil
}

func downSeedTestUserCredential(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(
		ctx,
		`DELETE FROM auth.local_auth_client_credential WHERE actor_id = $1`,
		testUserActorID,
	)
	if err != nil {
		return err
	}
	return nil
}
