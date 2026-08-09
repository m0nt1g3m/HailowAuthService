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
			INSERT INTO users_schema.users (first_name, last_name, email, phone_number, city, street, building, password_hash, role)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING *`

	rows, err := r.db.Query(ctx, query, input.FirstName, input.LastName, input.Email, input.PhoneNumber, input.City, input.Street, input.Building, input.Password, input.Role)
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

func (r *UserRepository) UpdateProfile(ctx context.Context, input *domain.User) (*domain.User, error) {
	query := `
			UPDATE users_schema.users
			SET email = $1, first_name = $2, last_name = $3, phone_number = $4, updated_at = CURRENT_TIMESTAMP
			WHERE id = $5
			RETURNING *
	`

	rows, err := r.db.Query(ctx, query, input.Email, input.FirstName, input.LastName, input.PhoneNumber, input.ID)
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

func (r *UserRepository) UpdateAvatar(ctx context.Context, userID uuid.UUID, avatarURL string) (*domain.User, string, error) {
	query := `
        WITH old_user AS (
            SELECT avatar_url FROM users_schema.users WHERE id = $2
        )
        UPDATE users_schema.users
        SET avatar_url = $1, updated_at = CURRENT_TIMESTAMP
        WHERE id = $2
        RETURNING *, (SELECT avatar_url FROM old_user) AS old_avatar_url
    `

	rows, err := r.db.Query(ctx, query, avatarURL, userID)
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

	// Temporary result shape used to map the updated user row and the previous avatar URL.
	type result struct {
		// Embed the user fields directly so pgx can map them from the returned row.
		domain.User `pgx:",inline"`
		// Capture the previous avatar URL from the SQL result.
		OldAvatarURL *string `db:"old_avatar_url"`
	}

	res, err := pgx.RowToStructByName[result](rows)
	if err != nil {
		return nil, "", err
	}

	var oldAvatarStr string
	if res.OldAvatarURL != nil {
		oldAvatarStr = *res.OldAvatarURL
	}

	return &res.User, oldAvatarStr, nil
}

func (r *UserRepository) UpdateDeliveryInfo(ctx context.Context, input *domain.User) (*domain.User, error) {
	query := `
			UPDATE users_schema.users
			SET city = $1, street = $2, building = $3, porch = $4, floor = $5, flat = $6, updated_at = CURRENT_TIMESTAMP
			WHERE id = $7
			RETURNING *
	`

	rows, err := r.db.Query(ctx, query, input.City, input.Street, input.Building, input.Porch, input.Floor, input.Flat, input.ID)
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

func (r *UserRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, newPasswordHash string) (*domain.User, error) {
	query := `
		UPDATE users_schema.users
		SET password_hash = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
		RETURNING *
	`

	rows, err := r.db.Query(ctx, query, newPasswordHash, userID)
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

func (r *UserRepository) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	query := `
		DELETE FROM users_schema.users
		WHERE id = $1
	`

	cmdTag, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}
