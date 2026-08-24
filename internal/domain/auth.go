package domain

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleUnspecified Role = "ROLE_UNSPECIFIED"
	RoleCustomer    Role = "ROLE_CUSTOMER"
	RoleAdmin       Role = "ROLE_ADMIN"
)

func (r *Role) Scan(value any) error {
	if value == nil {
		*r = RoleUnspecified
		return nil
	}
	switch v := value.(type) {
	case string:
		*r = Role(v)
	case []byte:
		*r = Role(v)
	default:
		return fmt.Errorf("Cannot scan %T into domain.Role", value)
	}
	return nil
}

func (r Role) Value() (driver.Value, error) {
	return string(r), nil
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

type RefreshSession struct {
	ID           string
	UserID       uuid.UUID
	DeviceID     string
	RefreshToken string
	UserAgent    string
	IP           string
	CreatedAt    time.Time
	ExpiresAt    time.Time
}

func (s *RefreshSession) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}
