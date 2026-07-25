package repository

import (
	"context"
	"errors"
	"sync"

	"HailowAuthService/internal/domain"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
}

type InMemoryUserRepository struct {
	mu sync.RWMutex

	usersByID    map[string]*domain.User
	usersByEmail map[string]*domain.User
}

func NewInMemoryUserRepository() UserRepository {
	return &InMemoryUserRepository{
		usersByID:    make(map[string]*domain.User),
		usersByEmail: make(map[string]*domain.User),
	}
}

func (r *InMemoryUserRepository) Create(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.usersByEmail[user.Email]; exists {
		return domain.ErrUserAlreadyExists
	}

	r.usersByID[user.ID] = user
	r.usersByEmail[user.Email] = user
	return nil
}

func (r *InMemoryUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.usersByID[id]
	if !exists {
		return nil, nil
	}

	return user, nil
}

func (r *InMemoryUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.usersByEmail[email]
	if !exists {
		return nil, nil
	}

	return user, nil
}

type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresUserRepository(pool *pgxpool.Pool) UserRepository {
	return &PostgresUserRepository{pool: pool}
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `INSERT INTO users_schema.users
		(id, avatar_url, first_name, last_name, email, password_hash, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.pool.Exec(ctx, query,
		user.ID,
		user.AvatarURL,
		user.FirstName,
		user.LastName,
		user.Email,
		user.PasswordHash,
		user.Role.String(),
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrUserAlreadyExists
		}
		return err
	}

	return nil
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `SELECT id, avatar_url, first_name, last_name, email, password_hash, role, created_at, updated_at
		FROM users_schema.users
		WHERE id = $1`

	row := r.pool.QueryRow(ctx, query, id)
	user := &domain.User{}
	var role string
	if err := row.Scan(
		&user.ID,
		&user.AvatarURL,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.PasswordHash,
		&role,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	user.Role = domain.ParseRole(role)
	return user, nil
}

func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, avatar_url, first_name, last_name, email, password_hash, role, created_at, updated_at
		FROM users_schema.users
		WHERE email = $1`

	row := r.pool.QueryRow(ctx, query, email)
	user := &domain.User{}
	var role string
	if err := row.Scan(
		&user.ID,
		&user.AvatarURL,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.PasswordHash,
		&role,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	user.Role = domain.ParseRole(role)
	return user, nil
}
