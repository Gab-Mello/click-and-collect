package carts

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gab-mello/click-and-collect/internal/orders"
	"github.com/gab-mello/click-and-collect/internal/products"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrCartNotActive       = errors.New("cart is not active")
	ErrCartEmpty           = errors.New("cart has no items")
	ErrInvalidQuantity     = errors.New("quantity must be greater than zero")
	ErrProductInactive     = errors.New("product is not available")
	ErrCustomerEmailReq    = errors.New("customer email is required")
)

type Service struct {
	repo     *Repo
	products *products.Service
	orders   *orders.Service
	pool     *pgxpool.Pool
	clock    func() time.Time
}

func NewService(repo *Repo, ps *products.Service, os *orders.Service, pool *pgxpool.Pool) *Service {
	return &Service{
		repo:     repo,
		products: ps,
		orders:   os,
		pool:     pool,
		clock:    time.Now,
	}
}

// Create starts a new ACTIVE cart for the given customer email.
func (s *Service) Create(ctx context.Context, customerEmail string) (Cart, error) {
	email := strings.TrimSpace(customerEmail)
	if email == "" {
		return Cart{}, ErrCustomerEmailReq
	}
	now := s.clock()
	c := Cart{
		ID:            newID(),
		CustomerEmail: email,
		Status:        StatusActive,
		Items:         []CartItem{},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.repo.CreateCart(ctx, c); err != nil {
		return Cart{}, err
	}
	return c, nil
}

// Get returns the cart with its items hydrated.
func (s *Service) Get(ctx context.Context, id string) (Cart, error) {
	c, err := s.repo.GetCart(ctx, id)
	if err != nil {
		return Cart{}, err
	}
	items, err := s.repo.ListItems(ctx, id)
	if err != nil {
		return Cart{}, err
	}
	c.Items = items
	return c, nil
}

// AddItem upserts a (product_id) into the cart: if the row exists, quantity is
// increased; otherwise a new row is created. Quantity must be > 0 and the
// product must exist and be active. Returns the fully hydrated cart.
func (s *Service) AddItem(ctx context.Context, cartID, productID string, quantity int) (Cart, error) {
	if quantity <= 0 {
		return Cart{}, ErrInvalidQuantity
	}
	c, err := s.repo.GetCart(ctx, cartID)
	if err != nil {
		return Cart{}, err
	}
	if c.Status != StatusActive {
		return Cart{}, ErrCartNotActive
	}
	p, err := s.products.Get(ctx, productID)
	if err != nil {
		return Cart{}, err
	}
	if !p.Active {
		return Cart{}, ErrProductInactive
	}

	now := s.clock()
	if _, err := s.repo.UpsertItem(ctx, CartItem{
		ID:        newID(),
		CartID:    cartID,
		ProductID: p.ID,
		Quantity:  quantity,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return Cart{}, err
	}
	return s.Get(ctx, cartID)
}

// SetItemQuantity overwrites the quantity of an existing cart item. The cart
// must be ACTIVE; quantity must be > 0.
func (s *Service) SetItemQuantity(ctx context.Context, cartID, productID string, quantity int) (Cart, error) {
	if quantity <= 0 {
		return Cart{}, ErrInvalidQuantity
	}
	c, err := s.repo.GetCart(ctx, cartID)
	if err != nil {
		return Cart{}, err
	}
	if c.Status != StatusActive {
		return Cart{}, ErrCartNotActive
	}
	if _, err := s.repo.SetItemQuantity(ctx, cartID, productID, quantity, s.clock()); err != nil {
		return Cart{}, err
	}
	return s.Get(ctx, cartID)
}

// RemoveItem deletes the cart item identified by (cart_id, product_id).
func (s *Service) RemoveItem(ctx context.Context, cartID, productID string) (Cart, error) {
	c, err := s.repo.GetCart(ctx, cartID)
	if err != nil {
		return Cart{}, err
	}
	if c.Status != StatusActive {
		return Cart{}, ErrCartNotActive
	}
	if err := s.repo.RemoveItem(ctx, cartID, productID); err != nil {
		return Cart{}, err
	}
	return s.Get(ctx, cartID)
}

func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Errorf("rand.Read: %w", err))
	}
	return hex.EncodeToString(b[:])
}
