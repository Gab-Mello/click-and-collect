package carts

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrCartNotFound     = errors.New("cart not found")
	ErrCartItemNotFound = errors.New("cart item not found")
)

// dbtx is satisfied by both *pgxpool.Pool and pgx.Tx, so the same SQL helpers
// run inside or outside a transaction.
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

func (r *Repo) CreateCart(ctx context.Context, c Cart) error {
	const q = `
		INSERT INTO carts (id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4)`
	if _, err := r.pool.Exec(ctx, q, c.ID, c.Status, c.CreatedAt, c.UpdatedAt); err != nil {
		return fmt.Errorf("insert cart: %w", err)
	}
	return nil
}

func (r *Repo) GetCart(ctx context.Context, id string) (Cart, error) {
	const q = `
		SELECT id, status, created_at, updated_at
		FROM carts WHERE id = $1`
	var c Cart
	err := r.pool.QueryRow(ctx, q, id).Scan(&c.ID, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Cart{}, ErrCartNotFound
		}
		return Cart{}, fmt.Errorf("get cart: %w", err)
	}
	return c, nil
}

func (r *Repo) ListItems(ctx context.Context, cartID string) ([]CartItem, error) {
	const q = `
		SELECT id, cart_id, product_id, quantity, created_at, updated_at
		FROM cart_items
		WHERE cart_id = $1
		ORDER BY created_at`
	rows, err := r.pool.Query(ctx, q, cartID)
	if err != nil {
		return nil, fmt.Errorf("query cart items: %w", err)
	}
	defer rows.Close()

	out := []CartItem{}
	for rows.Next() {
		var it CartItem
		if err := rows.Scan(&it.ID, &it.CartID, &it.ProductID, &it.Quantity, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan cart item: %w", err)
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cart items: %w", err)
	}
	return out, nil
}

// UpsertItem inserts a new cart_items row, or increases the quantity of the
// existing (cart_id, product_id) row by `addQty`. Returns the resulting item.
func (r *Repo) UpsertItem(ctx context.Context, it CartItem) (CartItem, error) {
	const q = `
		INSERT INTO cart_items (id, cart_id, product_id, quantity, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (cart_id, product_id) DO UPDATE
		  SET quantity   = cart_items.quantity + EXCLUDED.quantity,
		      updated_at = EXCLUDED.updated_at
		RETURNING id, cart_id, product_id, quantity, created_at, updated_at`
	var out CartItem
	err := r.pool.QueryRow(ctx, q,
		it.ID, it.CartID, it.ProductID, it.Quantity, it.CreatedAt, it.UpdatedAt,
	).Scan(&out.ID, &out.CartID, &out.ProductID, &out.Quantity, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		return CartItem{}, fmt.Errorf("upsert cart item: %w", err)
	}
	return out, nil
}

func (r *Repo) SetItemQuantity(ctx context.Context, cartID, productID string, quantity int, updatedAt time.Time) (CartItem, error) {
	const q = `
		UPDATE cart_items
		SET quantity = $3, updated_at = $4
		WHERE cart_id = $1 AND product_id = $2
		RETURNING id, cart_id, product_id, quantity, created_at, updated_at`
	var out CartItem
	err := r.pool.QueryRow(ctx, q, cartID, productID, quantity, updatedAt).Scan(
		&out.ID, &out.CartID, &out.ProductID, &out.Quantity, &out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CartItem{}, ErrCartItemNotFound
		}
		return CartItem{}, fmt.Errorf("update cart item: %w", err)
	}
	return out, nil
}

func (r *Repo) RemoveItem(ctx context.Context, cartID, productID string) error {
	const q = `DELETE FROM cart_items WHERE cart_id = $1 AND product_id = $2`
	tag, err := r.pool.Exec(ctx, q, cartID, productID)
	if err != nil {
		return fmt.Errorf("delete cart item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCartItemNotFound
	}
	return nil
}

// LockCartTx selects the cart row FOR UPDATE inside the supplied transaction,
// preventing concurrent checkouts of the same cart.
func (r *Repo) LockCartTx(ctx context.Context, q dbtx, id string) (Cart, error) {
	const sql = `
		SELECT id, status, created_at, updated_at
		FROM carts
		WHERE id = $1
		FOR UPDATE`
	var c Cart
	err := q.QueryRow(ctx, sql, id).Scan(&c.ID, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Cart{}, ErrCartNotFound
		}
		return Cart{}, fmt.Errorf("lock cart: %w", err)
	}
	return c, nil
}

func (r *Repo) ListItemsTx(ctx context.Context, q dbtx, cartID string) ([]CartItem, error) {
	const sql = `
		SELECT id, cart_id, product_id, quantity, created_at, updated_at
		FROM cart_items
		WHERE cart_id = $1
		ORDER BY created_at`
	rows, err := q.Query(ctx, sql, cartID)
	if err != nil {
		return nil, fmt.Errorf("query cart items: %w", err)
	}
	defer rows.Close()

	out := []CartItem{}
	for rows.Next() {
		var it CartItem
		if err := rows.Scan(&it.ID, &it.CartID, &it.ProductID, &it.Quantity, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan cart item: %w", err)
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cart items: %w", err)
	}
	return out, nil
}

func (r *Repo) MarkCheckedOutTx(ctx context.Context, q dbtx, cartID string, updatedAt time.Time) error {
	const sql = `UPDATE carts SET status = $2, updated_at = $3 WHERE id = $1`
	tag, err := q.Exec(ctx, sql, cartID, StatusCheckedOut, updatedAt)
	if err != nil {
		return fmt.Errorf("mark cart checked out: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCartNotFound
	}
	return nil
}
