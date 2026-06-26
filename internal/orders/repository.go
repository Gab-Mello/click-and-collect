package orders

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("order not found")

// dbtx is the subset of pgx behavior we use for SQL. Both *pgxpool.Pool and
// pgx.Tx satisfy it, so the same query helpers run inside or outside a
// transaction.
type dbtx interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func (r *Repo) Create(ctx context.Context, o Order) error {
	return createOrder(ctx, r.pool, o)
}

// CreateTx inserts an order using the supplied transaction or pool.
func (r *Repo) CreateTx(ctx context.Context, q dbtx, o Order) error {
	return createOrder(ctx, q, o)
}

func createOrder(ctx context.Context, q dbtx, o Order) error {
	const sql = `
		INSERT INTO orders (id, customer_name, customer_email, delivery_method, pickup_store_id, pickup_code, status, cart_id, total_amount_cents, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, err := q.Exec(ctx, sql,
		o.ID, o.CustomerName, o.CustomerEmail, o.DeliveryMethod,
		o.PickupStoreID, o.PickupCode, o.Status, o.CartID, o.TotalAmountCents,
		o.CreatedAt, o.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}
	return nil
}

// AddItemsTx inserts a batch of order items. The caller is expected to have
// populated each item's ID and OrderID.
func (r *Repo) AddItemsTx(ctx context.Context, q dbtx, items []OrderItem) error {
	const sql = `
		INSERT INTO order_items (id, order_id, product_id, product_name, unit_price_cents, quantity, total_price_cents)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	for _, it := range items {
		if _, err := q.Exec(ctx, sql,
			it.ID, it.OrderID, it.ProductID, it.ProductName,
			it.UnitPriceCents, it.Quantity, it.TotalPriceCents,
		); err != nil {
			return fmt.Errorf("insert order item: %w", err)
		}
	}
	return nil
}

func (r *Repo) Get(ctx context.Context, id string) (Order, error) {
	const q = `
		SELECT id, customer_name, customer_email, delivery_method, pickup_store_id, pickup_code, status, cart_id, total_amount_cents, created_at, updated_at
		FROM orders WHERE id = $1`
	var o Order
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&o.ID, &o.CustomerName, &o.CustomerEmail, &o.DeliveryMethod,
		&o.PickupStoreID, &o.PickupCode, &o.Status, &o.CartID, &o.TotalAmountCents,
		&o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Order{}, ErrNotFound
		}
		return Order{}, fmt.Errorf("get order: %w", err)
	}
	items, err := r.GetItems(ctx, o.ID)
	if err != nil {
		return Order{}, err
	}
	o.Items = items
	return o, nil
}

func (r *Repo) Update(ctx context.Context, o Order) error {
	const q = `
		UPDATE orders
		SET customer_name = $2, customer_email = $3, delivery_method = $4,
		    pickup_store_id = $5, pickup_code = $6, status = $7,
		    cart_id = $8, total_amount_cents = $9, updated_at = $10
		WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q,
		o.ID, o.CustomerName, o.CustomerEmail, o.DeliveryMethod,
		o.PickupStoreID, o.PickupCode, o.Status,
		o.CartID, o.TotalAmountCents, o.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update order: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) GetItems(ctx context.Context, orderID string) ([]OrderItem, error) {
	const q = `
		SELECT id, order_id, product_id, product_name, unit_price_cents, quantity, total_price_cents
		FROM order_items
		WHERE order_id = $1
		ORDER BY id`
	rows, err := r.pool.Query(ctx, q, orderID)
	if err != nil {
		return nil, fmt.Errorf("query order items: %w", err)
	}
	defer rows.Close()

	out := []OrderItem{}
	for rows.Next() {
		var it OrderItem
		if err := rows.Scan(
			&it.ID, &it.OrderID, &it.ProductID, &it.ProductName,
			&it.UnitPriceCents, &it.Quantity, &it.TotalPriceCents,
		); err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate order items: %w", err)
	}
	return out, nil
}

func (r *Repo) AddNotification(ctx context.Context, n Notification) error {
	const q = `INSERT INTO notifications (id, order_id, message, created_at) VALUES ($1, $2, $3, $4)`
	if _, err := r.pool.Exec(ctx, q, n.ID, n.OrderID, n.Message, n.CreatedAt); err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}
	return nil
}

func (r *Repo) ListNotifications(ctx context.Context, orderID string) ([]Notification, error) {
	const q = `SELECT id, order_id, message, created_at FROM notifications WHERE order_id = $1 ORDER BY created_at`
	rows, err := r.pool.Query(ctx, q, orderID)
	if err != nil {
		return nil, fmt.Errorf("query notifications: %w", err)
	}
	defer rows.Close()

	var out []Notification
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.OrderID, &n.Message, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notifications: %w", err)
	}
	return out, nil
}
