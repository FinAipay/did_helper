package models

import (
	"time"
)

type APIKey struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Key         string    `json:"key"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
}
