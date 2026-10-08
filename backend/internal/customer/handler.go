package customer

import (
	"encoding/json"
	"log"
	"net/http"
	"net/mail"
	"strings"

	"workshop-api/internal/httpx"
)

// Handler serves the public customer endpoint.
type Handler struct {
	store *Store
}

// NewHandler builds the customer handler on the shared store.
func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// createRequest is the body of POST /api/customers.
type createRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

// Create handles POST /api/customers: it stores the customer and answers 201
// with the stored record including its stable id, or 400 with the uniform
// error body when a value is missing or malformed.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "request body is not valid JSON")
		return
	}

	name := strings.TrimSpace(req.Name)
	email := strings.TrimSpace(req.Email)
	phone := strings.TrimSpace(req.Phone)

	if name == "" {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "name is required")
		return
	}
	if email == "" {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "email is required")
		return
	}
	if _, err := mail.ParseAddress(email); err != nil || !strings.Contains(email, "@") {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "email is malformed")
		return
	}
	if phone == "" {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "phone is required")
		return
	}

	customer, err := h.store.UpsertCustomer(r.Context(), name, email, phone)
	if err != nil {
		log.Printf("create customer: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	httpx.JSON(w, http.StatusCreated, customer)
}
