package domain

import "time"

type User struct {
	ID           string
	AvatarURL    *string
	FirstName    string
	LastName     string
	Email        string
	PhoneNumber  string
	City         string
	Street       string
	Building     string
	Flat         *int32
	Porch        *int32
	Floor        *int32
	PasswordHash string
	Role         Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserInfo struct {
	ID          string
	AvatarURL   *string
	Email       string
	FirstName   string
	LastName    string
	PhoneNumber string
	City        string
	Street      string
	Building    string
	Flat        *int32
	Porch       *int32
	Floor       *int32
	Password    string
	Role        Role
}

type DeliveryInfo struct {
	City     string
	Street   string
	Building string
	Flat     *int32
	Porch    *int32
	Floor    *int32
}
