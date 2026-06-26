package carts

import "time"

type Status string

const (
	StatusActive     Status = "ACTIVE"
	StatusCheckedOut Status = "CHECKED_OUT"
)

func (s Status) Valid() bool {
	switch s {
	case StatusActive, StatusCheckedOut:
		return true
	}
	return false
}

type Cart struct {
	ID        string     `json:"id"`
	Status    Status     `json:"status"`
	Items     []CartItem `json:"items"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type CartItem struct {
	ID        string    `json:"id"`
	CartID    string    `json:"cart_id"`
	ProductID string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
