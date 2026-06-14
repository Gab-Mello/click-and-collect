package server

import (
	"encoding/json"
	"net/http"

	"github.com/gab-mello/click-and-collect/internal/docs"
	"github.com/gab-mello/click-and-collect/internal/orders"
	"github.com/gab-mello/click-and-collect/internal/stores"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(storesH *stores.Handler, ordersH *orders.Handler) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(CORS)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", healthz)

	// API documentation. Mounted outside /api/v1 so the URLs stay stable
	// across API versions and are obviously not part of the contract.
	r.Method(http.MethodGet, "/openapi.yaml", docs.SpecHandler())
	r.Method(http.MethodGet, "/docs", docs.UIHandler())

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", healthz)
		storesH.Mount(r)
		ordersH.Mount(r)
	})

	return r
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
