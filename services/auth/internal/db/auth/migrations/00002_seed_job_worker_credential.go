package migrations

import (
	"context"
	"database/sql"
	"os"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upSeedJobWorkerCredential, downSeedJobWorkerCredential)
}

var workerActorID = uuid.MustParse("ffffffff-ffff-ffff-ffff-fffffffffffe")

func upSeedJobWorkerCredential(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	client_id := os.Getenv("AUTH_JOB_WORKER_CLIENT_ID")
	secret_hash := os.Getenv("AUTH_JOB_WORKER_SECRET_HASH")
	if client_id == "" || secret_hash == "" {
		return nil
	}
	_, err := tx.ExecContext(
		ctx,
		`
		INSERT INTO auth.local_auth_client_credential (
			actor_id, client_id, secret_hash
		)
		VALUES ($1, $2, $3)
		`,
		workerActorID,
		client_id,
		secret_hash,
	)
	if err != nil {
		return err
	}
	return nil
}

func downSeedJobWorkerCredential(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.ExecContext(
		ctx,
		`DELETE FROM auth.local_auth_client_credential WHERE actor_id = $1`,
		workerActorID,
	)
	if err != nil {
		return err
	}
	return nil
}
