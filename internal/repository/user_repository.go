package repository

import (
	"context"
	"errors"
	"fmt"

	"btsid/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepository defines the database operation contract for users.
type UserRepository interface {
	CreateUser(ctx context.Context, username, passwordHash string) (*domain.User, error)
	GetUserByUsername(ctx context.Context, username string) (*domain.User, error)
	GetUserByID(ctx context.Context, id int64) (*domain.User, error)
}

type PgUserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *PgUserRepository {
	return &PgUserRepository{pool: pool}
}

// CreateUser persists a new user record in PostgreSQL.
// Returns domain.ErrConflict if username violates UNIQUE constraint (code 23505).
func (r *PgUserRepository) CreateUser(ctx context.Context, username, passwordHash string) (*domain.User, error) {
	query := `
		INSERT INTO users (username, password_hash, created_at, updated_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id, username, password_hash, created_at, updated_at
	`

	var u domain.User
	err := r.pool.QueryRow(ctx, query, username, passwordHash).Scan(
		&u.ID,
		&u.Username,
		&u.PasswordHash,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, domain.NewConflictError("username already exists")
		}
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	return &u, nil
}

// GetUserByUsername retrieves a user by their exact username.
// Returns domain.ErrNotFound if user does not exist.
func (r *PgUserRepository) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := `
		SELECT id, username, password_hash, created_at, updated_at
		FROM users
		WHERE username = $1
	`

	var u domain.User
	err := r.pool.QueryRow(ctx, query, username).Scan(
		&u.ID,
		&u.Username,
		&u.PasswordHash,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.NewNotFoundError("user not found")
		}
		return nil, fmt.Errorf("failed to query user by username: %w", err)
	}

	return &u, nil
}

// GetUserByID retrieves a user by their primary key ID.
// Returns domain.ErrNotFound if user does not exist.
func (r *PgUserRepository) GetUserByID(ctx context.Context, id int64) (*domain.User, error) {
	query := `
		SELECT id, username, password_hash, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var u domain.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.Username,
		&u.PasswordHash,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.NewNotFoundError("user not found")
		}
		return nil, fmt.Errorf("failed to query user by ID: %w", err)
	}

	return &u, nil
}
