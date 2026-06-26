package orders

import "time"

type DeliveryMethod string

const (
	DeliveryStandard      DeliveryMethod = "standard"
	DeliveryPickupInStore DeliveryMethod = "pickup_in_store"
)

func (d DeliveryMethod) Valid() bool {
	switch d {
	case DeliveryStandard, DeliveryPickupInStore:
		return true
	}
	return false
}

type Status string

const (
	StatusAwaitingPreparation Status = "AWAITING_PREPARATION"
	StatusReadyForPickup      Status = "READY_FOR_PICKUP"
	StatusCollected           Status = "COLLECTED"
	StatusCancelled           Status = "CANCELLED"
)

func (s Status) Valid() bool {
	switch s {
	case StatusAwaitingPreparation, StatusReadyForPickup, StatusCollected, StatusCancelled:
		return true
	}
	return false
}

type Order struct {
	ID               string         `json:"id"`
	CustomerName     string         `json:"customer_name"`
	CustomerEmail    string         `json:"customer_email"`
	DeliveryMethod   DeliveryMethod `json:"delivery_method"`
	PickupStoreID    *string        `json:"pickup_store_id,omitempty"`
	PickupCode       *string        `json:"pickup_code,omitempty"`
	Status           Status         `json:"status"`
	CartID           *string        `json:"cart_id,omitempty"`
	TotalAmountCents int64          `json:"total_amount_cents"`
	Items            []OrderItem    `json:"items"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

type OrderItem struct {
	ID              string `json:"id"`
	OrderID         string `json:"order_id"`
	ProductID       string `json:"product_id"`
	ProductName     string `json:"product_name"`
	UnitPriceCents  int64  `json:"unit_price_cents"`
	Quantity        int    `json:"quantity"`
	TotalPriceCents int64  `json:"total_price_cents"`
}

type Notification struct {
	ID        string    `json:"id"`
	OrderID   string    `json:"order_id"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}
