package domain

import "time"

type User struct {
	ID           string    `db:"id"`
	Avatar       *string   `db:"avatar"`
	FirstName    string    `db:"first_name"`
	LastName     string    `db:"last_name"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`
	Role         Role      `db:"role"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

type UserInfo struct {
	Avatar    string `db:"avatar"`
	Email     string `db:"email"`
	FirstName string `db:"first_name"`
	LastName  string `db:"last_name"`
	Password  string `db:"password"`
	Role      Role   `db:"role"`
}
