package domain

import "time"

type Role int

const (
	RoleUnspecified Role = iota
	RoleCustomer
	RoleSeller
	RoleAdmin
)

func (r Role) String() string {
	switch r {
	case RoleCustomer:
		return "customer"
	case RoleSeller:
		return "seller"
	case RoleAdmin:
		return "admin"
	default:
		return "unspecified"
	}
}

func ParseRole(value string) Role {
	switch value {
	case "customer":
		return RoleCustomer
	case "seller":
		return RoleSeller
	case "admin":
		return RoleAdmin
	default:
		return RoleUnspecified
	}
}

type User struct {
	ID           string
	AvatarURL    string
	FirstName    string
	LastName     string
	Email        string
	PasswordHash string
	Role         Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
