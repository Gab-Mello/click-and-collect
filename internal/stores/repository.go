package stores

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("store not found")

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func (r *Repo) List(ctx context.Context) ([]Store, error) {
	const q = `SELECT id, name, address, city, state, zip, capacity, active FROM stores ORDER BY id`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query stores: %w", err)
	}
	defer rows.Close()

	var out []Store
	for rows.Next() {
		var s Store
		if err := rows.Scan(&s.ID, &s.Name, &s.Address, &s.City, &s.State, &s.ZIP, &s.Capacity, &s.Active); err != nil {
			return nil, fmt.Errorf("scan store: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stores: %w", err)
	}
	return out, nil
}

func (r *Repo) Get(ctx context.Context, id string) (Store, error) {
	const q = `SELECT id, name, address, city, state, zip, capacity, active FROM stores WHERE id = $1`
	var s Store
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&s.ID, &s.Name, &s.Address, &s.City, &s.State, &s.ZIP, &s.Capacity, &s.Active,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Store{}, ErrNotFound
		}
		return Store{}, fmt.Errorf("get store: %w", err)
	}
	return s, nil
}
