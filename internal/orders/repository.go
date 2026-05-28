package orders

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("order not found")

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func (r *Repo) Create(ctx context.Context, o Order) error {
	const q = `
		INSERT INTO orders (id, customer_name, customer_email, delivery_method, pickup_store_id, pickup_code, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.pool.Exec(ctx, q,
		o.ID, o.CustomerName, o.CustomerEmail, o.DeliveryMethod,
		o.PickupStoreID, o.PickupCode, o.Status, o.CreatedAt, o.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}
	return nil
}

func (r *Repo) Get(ctx context.Context, id string) (Order, error) {
	const q = `
		SELECT id, customer_name, customer_email, delivery_method, pickup_store_id, pickup_code, status, created_at, updated_at
		FROM orders WHERE id = $1`
	var o Order
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&o.ID, &o.CustomerName, &o.CustomerEmail, &o.DeliveryMethod,
		&o.PickupStoreID, &o.PickupCode, &o.Status, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Order{}, ErrNotFound
		}
		return Order{}, fmt.Errorf("get order: %w", err)
	}
	return o, nil
}

func (r *Repo) Update(ctx context.Context, o Order) error {
	const q = `
		UPDATE orders
		SET customer_name = $2, customer_email = $3, delivery_method = $4,
		    pickup_store_id = $5, pickup_code = $6, status = $7, updated_at = $8
		WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q,
		o.ID, o.CustomerName, o.CustomerEmail, o.DeliveryMethod,
		o.PickupStoreID, o.PickupCode, o.Status, o.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update order: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
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
