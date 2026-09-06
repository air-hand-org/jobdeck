package repository

import (
	"context"
	"jobdeck/services/auth/internal/app"
	authdb "jobdeck/services/auth/internal/repository/gen"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store provides authentication persistence.
type Store struct {
	pool *pgxpool.Pool
	q    *authdb.Queries
}

var _ app.AuthStore = (*Store)(nil)

// Auths returns the authentication repository.
func (s *Store) Auths() app.AuthRepository {
	return newAuthRepository(s.q)
}

// WithinAuthTx executes fn inside an authentication transaction.
func (s *Store) WithinAuthTx(ctx context.Context, fn func(ctx context.Context, auths app.AuthRepository) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	auths := newAuthRepository(s.q.WithTx(tx))
	if err := fn(ctx, auths); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// NewStore creates a Store backed by a PostgreSQL pool.
func NewStore(p *pgxpool.Pool) *Store {
	return &Store{pool: p, q: authdb.New(p)}
}
