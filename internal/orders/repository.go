package orders

import (
	"errors"
	"sync"
)

var ErrNotFound = errors.New("order not found")

type Repo struct {
	mu            sync.RWMutex
	orders        map[string]Order
	notifications map[string][]Notification
}

func NewRepo() *Repo {
	return &Repo{
		orders:        make(map[string]Order),
		notifications: make(map[string][]Notification),
	}
}

func (r *Repo) Create(o Order) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[o.ID] = o
}

func (r *Repo) Get(id string) (Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o, ok := r.orders[id]
	if !ok {
		return Order{}, ErrNotFound
	}
	return o, nil
}

func (r *Repo) Update(o Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.orders[o.ID]; !ok {
		return ErrNotFound
	}
	r.orders[o.ID] = o
	return nil
}

func (r *Repo) AddNotification(n Notification) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.notifications[n.OrderID] = append(r.notifications[n.OrderID], n)
}

func (r *Repo) ListNotifications(orderID string) []Notification {
	r.mu.RLock()
	defer r.mu.RUnlock()
	src := r.notifications[orderID]
	out := make([]Notification, len(src))
	copy(out, src)
	return out
}
