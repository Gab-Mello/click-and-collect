package stores

import (
	"errors"
	"sync"
)

var ErrNotFound = errors.New("store not found")

type Repo struct {
	mu     sync.RWMutex
	byID   map[string]Store
	sorted []string
}

func NewRepo() *Repo {
	r := &Repo{byID: make(map[string]Store)}
	for _, s := range seed() {
		r.byID[s.ID] = s
		r.sorted = append(r.sorted, s.ID)
	}
	return r
}

func (r *Repo) List() []Store {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Store, 0, len(r.sorted))
	for _, id := range r.sorted {
		out = append(out, r.byID[id])
	}
	return out
}

func (r *Repo) Get(id string) (Store, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.byID[id]
	if !ok {
		return Store{}, ErrNotFound
	}
	return s, nil
}

func seed() []Store {
	return []Store{
		{
			ID: "store-sp-paulista", Name: "MegaLoja Paulista",
			Address: "Av. Paulista, 1000", City: "São Paulo", State: "SP", ZIP: "01310-100",
			Capacity: 50, Active: true,
		},
		{
			ID: "store-rj-copacabana", Name: "MegaLoja Copacabana",
			Address: "Av. Atlântica, 500", City: "Rio de Janeiro", State: "RJ", ZIP: "22010-000",
			Capacity: 30, Active: true,
		},
		{
			ID: "store-mg-savassi", Name: "MegaLoja Savassi",
			Address: "R. Pernambuco, 200", City: "Belo Horizonte", State: "MG", ZIP: "30130-150",
			Capacity: 20, Active: false,
		},
	}
}
