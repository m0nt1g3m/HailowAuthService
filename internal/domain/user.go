package domain

import "time"

type User struct {
	ID           string    `db:"id"`
	AvatarURL    *string   `db:"avatar_url"`
	FirstName    string    `db:"first_name"`
	LastName     string    `db:"last_name"`
	Email        string    `db:"email"`
	PhoneNumber  string    `db:"phone_number"`
	City         string    `db:"city"`
	Street       string    `db:"street"`
	Building     string    `db:"building"`
	Flat         *int32    `db:"flat"`
	Porch        *int32    `db:"porch"`
	Floor        *int32    `db:"floor"`
	PasswordHash string    `db:"password_hash"`
	Role         Role      `db:"role"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

type UserInfo struct {
	AvatarURL   *string `db:"avatar_url"`
	Email       string  `db:"email"`
	FirstName   string  `db:"first_name"`
	LastName    string  `db:"last_name"`
	PhoneNumber string  `db:"phone_number"`
	City        string  `db:"city"`
	Street      string  `db:"street"`
	Building    string  `db:"building"`
	Flat        *int32  `db:"flat"`
	Porch       *int32  `db:"porch"`
	Floor       *int32  `db:"floor"`
	Password    string  `db:"password"`
	Role        Role    `db:"role"`
}
