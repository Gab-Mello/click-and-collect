package carts

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/gab-mello/click-and-collect/internal/orders"
	"github.com/gab-mello/click-and-collect/internal/products"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrCartNotActive   = errors.New("cart is not active")
	ErrCartEmpty       = errors.New("cart has no items")
	ErrInvalidQuantity = errors.New("quantity must be greater than zero")
	ErrProductInactive = errors.New("product is not available")
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

// Create starts a new anonymous ACTIVE cart. Customer identity is collected
// later, at checkout time, and stored on the resulting order — not the cart.
func (s *Service) Create(ctx context.Context) (Cart, error) {
	now := s.clock()
	c := Cart{
		ID:        newID(),
		Status:    StatusActive,
		Items:     []CartItem{},
		CreatedAt: now,
		UpdatedAt: now,
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

// CheckoutInput is the request payload for converting a cart into an order.
type CheckoutInput struct {
	CustomerName   string
	CustomerEmail  string
	DeliveryMethod orders.DeliveryMethod
	PickupStoreID  *string
}

// Checkout converts an ACTIVE non-empty cart into an order plus order_items
// inside a single transaction. On any failure, the transaction rolls back and
// the cart remains ACTIVE. Returns the created order (with items + total).
func (s *Service) Checkout(ctx context.Context, cartID string, in CheckoutInput) (orders.Order, *orders.Notification, error) {
	var (
		out  orders.Order
		note *orders.Notification
	)
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		cart, err := s.repo.LockCartTx(ctx, tx, cartID)
		if err != nil {
			return err
		}
		if cart.Status != StatusActive {
			return ErrCartNotActive
		}

		items, err := s.repo.ListItemsTx(ctx, tx, cartID)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return ErrCartEmpty
		}

		orderItems, total, err := s.snapshotItems(ctx, items)
		if err != nil {
			return err
		}

		cartIDCopy := cart.ID
		o, n, err := s.orders.CheckoutTx(ctx, tx, orders.CheckoutInput{
			CustomerName:     in.CustomerName,
			CustomerEmail:    in.CustomerEmail,
			DeliveryMethod:   in.DeliveryMethod,
			PickupStoreID:    in.PickupStoreID,
			CartID:           &cartIDCopy,
			Items:            orderItems,
			TotalAmountCents: total,
		})
		if err != nil {
			return err
		}
		if err := s.repo.MarkCheckedOutTx(ctx, tx, cart.ID, s.clock()); err != nil {
			return err
		}

		out, note = o, n
		return nil
	})
	return out, note, err
}

// snapshotItems looks up every product currently in the cart, rejects inactive
// ones, and produces order items with the product name and unit price
// captured at this moment. The returned total is the sum of total_price_cents.
func (s *Service) snapshotItems(ctx context.Context, items []CartItem) ([]orders.OrderItem, int64, error) {
	out := make([]orders.OrderItem, 0, len(items))
	var total int64
	for _, it := range items {
		p, err := s.products.Get(ctx, it.ProductID)
		if err != nil {
			return nil, 0, err
		}
		if !p.Active {
			return nil, 0, ErrProductInactive
		}
		line := orders.OrderItem{
			ProductID:       p.ID,
			ProductName:     p.Name,
			UnitPriceCents:  p.PriceCents,
			Quantity:        it.Quantity,
			TotalPriceCents: p.PriceCents * int64(it.Quantity),
		}
		out = append(out, line)
		total += line.TotalPriceCents
	}
	return out, total, nil
}

func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Errorf("rand.Read: %w", err))
	}
	return hex.EncodeToString(b[:])
}
