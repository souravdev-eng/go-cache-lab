package bookstore

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("book not found")

type Book struct {
	ID         int64     `json:"id"`
	Title      string    `json:"title"`
	Author     string    `json:"author"`
	PriceCents int64     `json:"price_cents"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type BookUpdate struct {
	Title      string `json:"title"`
	Author     string `json:"author"`
	PriceCents int64  `json:"price_cents"`
}
