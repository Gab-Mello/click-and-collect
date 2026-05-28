package stores

import (
	"errors"
	"net/http"

	"github.com/gab-mello/click-and-collect/internal/httpx"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Mount(r chi.Router) {
	r.Route("/stores", func(r chi.Router) {
		r.Get("/", h.list)
		r.Get("/{id}", h.get)
	})
}

type listResponse struct {
	Stores []Store `json:"stores"`
}

func (h *Handler) list(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, listResponse{Stores: h.svc.List()})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	s, err := h.svc.Get(id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "store_not_found", err.Error())
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s)
}
