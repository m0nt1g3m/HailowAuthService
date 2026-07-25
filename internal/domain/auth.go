package domain

import "time"

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

type Session struct {
	ID        string
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
}
