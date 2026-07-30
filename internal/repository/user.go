package repository

import (
	"HailowAuthService/internal/domain"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT *
		FROM users_schema.users
		WHERE id = $1
		LIMIT 1
	`

	rows, err := r.db.Query(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	user, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[domain.User])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT *
		FROM users_schema.users
		WHERE email = $1
		LIMIT 1
	`

	rows, err := r.db.Query(ctx, query, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	user, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[domain.User])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) CreateUser(ctx context.Context, input *domain.UserInfo) (*domain.User, error) {
	query := `
			INSERT INTO users_schema.users (first_name, last_name, email, password_hash, role)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING *`

	rows, err := r.db.Query(ctx, query, input.FirstName, input.LastName, input.Email, input.Password, input.Role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	user, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[domain.User])
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) UpdateUserInfo(ctx context.Context, input *domain.User) (*domain.User, error) {
	query := `
			UPDATE users_schema.users
			SET email = $1, first_name = $2, last_name = $3
			WHERE id = $4
			RETURNING *
	`

	rows, err := r.db.Query(ctx, query, input.Email, input.FirstName, input.LastName, input.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	user, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[domain.User])
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := `
		SELECT *
		FROM users_schema.users
		WHERE email = $1 OR first_name = $1 OR last_name = $1
		LIMIT 1
	`

	rows, err := r.db.Query(ctx, query, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	user, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[domain.User])
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) UpdateAvatar(ctx context.Context, userID uuid.UUID, avatar string) (*domain.User, string, error) {
	query := `
        WITH old_user AS (
            SELECT avatar FROM users_schema.users WHERE id = $2
        )
        UPDATE users_schema.users
        SET avatar = $1
        WHERE id = $2
        RETURNING *, (SELECT avatar FROM old_user) AS old_avatar
    `

	rows, err := r.db.Query(ctx, query, avatar, userID)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, "", err
		}
		return nil, "", domain.ErrUserNotFound
	}

	type result struct {
		domain.User `pgx:",inline"`
		OldAvatar   *string `db:"old_avatar"`
	}

	res, err := pgx.RowToStructByName[result](rows)
	if err != nil {
		return nil, "", err
	}

	var oldAvatarStr string
	if res.OldAvatar != nil {
		oldAvatarStr = *res.OldAvatar
	}

	return &res.User, oldAvatarStr, nil
}
