package carts

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gab-mello/click-and-collect/internal/httpx"
	"github.com/gab-mello/click-and-collect/internal/orders"
	"github.com/gab-mello/click-and-collect/internal/products"
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
	r.Route("/carts", func(r chi.Router) {
		r.Post("/", h.create)
		r.Get("/{id}", h.get)
		r.Post("/{id}/items", h.addItem)
		r.Patch("/{id}/items/{product_id}", h.patchItem)
		r.Delete("/{id}/items/{product_id}", h.deleteItem)
		r.Post("/{id}/checkout", h.checkout)
	})
}

type createRequest struct {
	CustomerEmail string `json:"customer_email"`
}

type addItemRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type patchItemRequest struct {
	Quantity int `json:"quantity"`
}

type checkoutRequest struct {
	CustomerName   string  `json:"customer_name"`
	CustomerEmail  string  `json:"customer_email"`
	DeliveryMethod string  `json:"delivery_method"`
	PickupStoreID  *string `json:"pickup_store_id,omitempty"`
}

type checkoutResponse struct {
	Order        orders.Order         `json:"order"`
	Notification *orders.Notification `json:"notification,omitempty"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "invalid JSON body")
		return
	}
	c, err := h.svc.Create(r.Context(), req.CustomerEmail)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, c)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	c, err := h.svc.Get(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}

func (h *Handler) addItem(w http.ResponseWriter, r *http.Request) {
	cartID := chi.URLParam(r, "id")
	var req addItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "invalid JSON body")
		return
	}
	c, err := h.svc.AddItem(r.Context(), cartID, req.ProductID, req.Quantity)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}

func (h *Handler) patchItem(w http.ResponseWriter, r *http.Request) {
	cartID := chi.URLParam(r, "id")
	productID := chi.URLParam(r, "product_id")
	var req patchItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "invalid JSON body")
		return
	}
	c, err := h.svc.SetItemQuantity(r.Context(), cartID, productID, req.Quantity)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}

func (h *Handler) deleteItem(w http.ResponseWriter, r *http.Request) {
	cartID := chi.URLParam(r, "id")
	productID := chi.URLParam(r, "product_id")
	c, err := h.svc.RemoveItem(r.Context(), cartID, productID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}

func (h *Handler) checkout(w http.ResponseWriter, r *http.Request) {
	cartID := chi.URLParam(r, "id")
	var req checkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "invalid JSON body")
		return
	}
	o, n, err := h.svc.Checkout(r.Context(), cartID, CheckoutInput{
		CustomerName:   req.CustomerName,
		CustomerEmail:  req.CustomerEmail,
		DeliveryMethod: orders.DeliveryMethod(req.DeliveryMethod),
		PickupStoreID:  req.PickupStoreID,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, checkoutResponse{Order: o, Notification: n})
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrCartNotFound):
		httpx.WriteError(w, http.StatusNotFound, "cart_not_found", err.Error())
	case errors.Is(err, ErrCartItemNotFound):
		httpx.WriteError(w, http.StatusNotFound, "cart_item_not_found", err.Error())
	case errors.Is(err, products.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "product_not_found", err.Error())
	case errors.Is(err, stores.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "store_not_found", err.Error())

	case errors.Is(err, ErrCartNotActive),
		errors.Is(err, ErrCartEmpty),
		errors.Is(err, ErrProductInactive),
		errors.Is(err, orders.ErrStoreInactive):
		httpx.WriteError(w, http.StatusConflict, errCode(err), err.Error())

	case errors.Is(err, ErrInvalidQuantity),
		errors.Is(err, ErrCustomerEmailReq),
		errors.Is(err, orders.ErrCustomerNameRequired),
		errors.Is(err, orders.ErrCustomerEmailRequired),
		errors.Is(err, orders.ErrInvalidDeliveryMethod),
		errors.Is(err, orders.ErrPickupStoreRequired),
		errors.Is(err, orders.ErrPickupStoreNotAllowed):
		httpx.WriteError(w, http.StatusBadRequest, errCode(err), err.Error())

	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
	}
}

func errCode(err error) string {
	switch {
	case errors.Is(err, ErrCartNotActive):
		return "cart_not_active"
	case errors.Is(err, ErrCartEmpty):
		return "cart_empty"
	case errors.Is(err, ErrProductInactive):
		return "product_inactive"
	case errors.Is(err, ErrInvalidQuantity):
		return "invalid_quantity"
	case errors.Is(err, ErrCustomerEmailReq):
		return "customer_email_required"
	case errors.Is(err, orders.ErrStoreInactive):
		return "store_inactive"
	case errors.Is(err, orders.ErrCustomerNameRequired):
		return "customer_name_required"
	case errors.Is(err, orders.ErrCustomerEmailRequired):
		return "customer_email_required"
	case errors.Is(err, orders.ErrInvalidDeliveryMethod):
		return "invalid_delivery_method"
	case errors.Is(err, orders.ErrPickupStoreRequired):
		return "pickup_store_required"
	case errors.Is(err, orders.ErrPickupStoreNotAllowed):
		return "pickup_store_not_allowed"
	}
	return "validation_error"
}
