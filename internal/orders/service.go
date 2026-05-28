package orders

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/gab-mello/click-and-collect/internal/stores"
)

var (
	ErrInvalidDeliveryMethod = errors.New("invalid delivery method")
	ErrCustomerNameRequired  = errors.New("customer name is required")
	ErrCustomerEmailRequired = errors.New("customer email is required")
	ErrPickupStoreRequired   = errors.New("pickup store is required for pickup_in_store orders")
	ErrPickupStoreNotAllowed = errors.New("pickup store must not be set for standard orders")
	ErrStoreInactive         = errors.New("selected store is not accepting pickups")
	ErrInvalidStatus         = errors.New("invalid status")
	ErrInvalidTransition     = errors.New("invalid status transition")
)

type Service struct {
	repo   *Repo
	stores *stores.Service
	clock  func() time.Time
}

func NewService(repo *Repo, stores *stores.Service) *Service {
	return &Service{repo: repo, stores: stores, clock: time.Now}
}

type CreateInput struct {
	CustomerName   string
	CustomerEmail  string
	DeliveryMethod DeliveryMethod
	PickupStoreID  *string
}

func (s *Service) Create(in CreateInput) (Order, error) {
	if strings.TrimSpace(in.CustomerName) == "" {
		return Order{}, ErrCustomerNameRequired
	}
	if strings.TrimSpace(in.CustomerEmail) == "" {
		return Order{}, ErrCustomerEmailRequired
	}
	if !in.DeliveryMethod.Valid() {
		return Order{}, ErrInvalidDeliveryMethod
	}

	now := s.clock()
	o := Order{
		ID:             newID(),
		CustomerName:   strings.TrimSpace(in.CustomerName),
		CustomerEmail:  strings.TrimSpace(in.CustomerEmail),
		DeliveryMethod: in.DeliveryMethod,
		Status:         StatusAwaitingPreparation,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	switch in.DeliveryMethod {
	case DeliveryStandard:
		if in.PickupStoreID != nil {
			return Order{}, ErrPickupStoreNotAllowed
		}
	case DeliveryPickupInStore:
		if in.PickupStoreID == nil || strings.TrimSpace(*in.PickupStoreID) == "" {
			return Order{}, ErrPickupStoreRequired
		}
		store, err := s.stores.Get(*in.PickupStoreID)
		if err != nil {
			return Order{}, err
		}
		if !store.Active {
			return Order{}, ErrStoreInactive
		}
		storeID := store.ID
		code := newPickupCode()
		o.PickupStoreID = &storeID
		o.PickupCode = &code
	}

	s.repo.Create(o)
	return o, nil
}

func (s *Service) Get(id string) (Order, error) {
	return s.repo.Get(id)
}

func (s *Service) ListNotifications(orderID string) ([]Notification, error) {
	if _, err := s.repo.Get(orderID); err != nil {
		return nil, err
	}
	return s.repo.ListNotifications(orderID), nil
}

// UpdateStatus transitions the order and, when the new status is READY_FOR_PICKUP
// for a pickup-in-store order, records a simulated notification (RF04). The
// returned notification pointer is non-nil only when one was created.
func (s *Service) UpdateStatus(id string, next Status) (Order, *Notification, error) {
	if !next.Valid() {
		return Order{}, nil, ErrInvalidStatus
	}
	o, err := s.repo.Get(id)
	if err != nil {
		return Order{}, nil, err
	}
	if !canTransition(o.Status, next) {
		return Order{}, nil, ErrInvalidTransition
	}

	o.Status = next
	o.UpdatedAt = s.clock()
	if err := s.repo.Update(o); err != nil {
		return Order{}, nil, err
	}

	if next == StatusReadyForPickup && o.DeliveryMethod == DeliveryPickupInStore {
		n := Notification{
			ID:        newID(),
			OrderID:   o.ID,
			Message:   fmt.Sprintf("Order %s is ready for pickup. Use code %s.", o.ID, derefCode(o.PickupCode)),
			CreatedAt: s.clock(),
		}
		s.repo.AddNotification(n)
		return o, &n, nil
	}
	return o, nil, nil
}

func canTransition(from, to Status) bool {
	switch from {
	case StatusAwaitingPreparation:
		return to == StatusReadyForPickup || to == StatusCancelled
	case StatusReadyForPickup:
		return to == StatusCollected || to == StatusCancelled
	}
	return false
}

func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Errorf("rand.Read: %w", err))
	}
	return hex.EncodeToString(b[:])
}

func newPickupCode() string {
	max := big.NewInt(1_000_000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		panic(fmt.Errorf("rand.Int: %w", err))
	}
	return fmt.Sprintf("%06d", n.Int64())
}

func derefCode(c *string) string {
	if c == nil {
		return ""
	}
	return *c
}
