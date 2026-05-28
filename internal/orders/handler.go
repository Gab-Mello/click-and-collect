package orders

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gab-mello/click-and-collect/internal/httpx"
	"github.com/gab-mello/click-and-collect/internal/stores"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Mount(r chi.Router) {
	r.Route("/orders", func(r chi.Router) {
		r.Post("/", h.create)
		r.Get("/{id}", h.get)
		r.Patch("/{id}/status", h.updateStatus)
		r.Get("/{id}/notifications", h.notifications)
	})
}

type createRequest struct {
	CustomerName   string  `json:"customer_name"`
	CustomerEmail  string  `json:"customer_email"`
	DeliveryMethod string  `json:"delivery_method"`
	PickupStoreID  *string `json:"pickup_store_id,omitempty"`
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

type updateStatusResponse struct {
	Order        Order         `json:"order"`
	Notification *Notification `json:"notification,omitempty"`
}

type notificationsResponse struct {
	Notifications []Notification `json:"notifications"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "invalid JSON body")
		return
	}

	o, err := h.svc.Create(CreateInput{
		CustomerName:   req.CustomerName,
		CustomerEmail:  req.CustomerEmail,
		DeliveryMethod: DeliveryMethod(req.DeliveryMethod),
		PickupStoreID:  req.PickupStoreID,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, o)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	o, err := h.svc.Get(id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, o)
}

func (h *Handler) updateStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req updateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "invalid JSON body")
		return
	}
	o, n, err := h.svc.UpdateStatus(id, Status(req.Status))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, updateStatusResponse{Order: o, Notification: n})
}

func (h *Handler) notifications(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ns, err := h.svc.ListNotifications(id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, notificationsResponse{Notifications: ns})
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "order_not_found", err.Error())
	case errors.Is(err, stores.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "store_not_found", err.Error())
	case errors.Is(err, ErrStoreInactive),
		errors.Is(err, ErrInvalidTransition):
		httpx.WriteError(w, http.StatusConflict, errCode(err), err.Error())
	case errors.Is(err, ErrInvalidDeliveryMethod),
		errors.Is(err, ErrCustomerNameRequired),
		errors.Is(err, ErrCustomerEmailRequired),
		errors.Is(err, ErrPickupStoreRequired),
		errors.Is(err, ErrPickupStoreNotAllowed),
		errors.Is(err, ErrInvalidStatus):
		httpx.WriteError(w, http.StatusBadRequest, errCode(err), err.Error())
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
	}
}

func errCode(err error) string {
	switch {
	case errors.Is(err, ErrInvalidDeliveryMethod):
		return "invalid_delivery_method"
	case errors.Is(err, ErrCustomerNameRequired):
		return "customer_name_required"
	case errors.Is(err, ErrCustomerEmailRequired):
		return "customer_email_required"
	case errors.Is(err, ErrPickupStoreRequired):
		return "pickup_store_required"
	case errors.Is(err, ErrPickupStoreNotAllowed):
		return "pickup_store_not_allowed"
	case errors.Is(err, ErrStoreInactive):
		return "store_inactive"
	case errors.Is(err, ErrInvalidStatus):
		return "invalid_status"
	case errors.Is(err, ErrInvalidTransition):
		return "invalid_transition"
	}
	return "validation_error"
}
