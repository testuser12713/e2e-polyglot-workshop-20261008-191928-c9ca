package vehicle

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"workshop-api/internal/httpx"
)

// Handler serves the public vehicle endpoint.
type Handler struct {
	store *Store
}

// NewHandler builds the vehicle handler on the shared store.
func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// createRequest is the body of POST /api/vehicles.
type createRequest struct {
	Plate   string `json:"plate"`
	Brand   string `json:"brand"`
	Model   string `json:"model"`
	Mileage *int   `json:"mileage"`
}

// Create handles POST /api/vehicles: it stores the vehicle and answers 201 with
// the stored record, 400 with the uniform error body for missing or malformed
// values, or 409 with the uniform error body when the plate is already taken.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "request body is not valid JSON")
		return
	}

	plate := strings.TrimSpace(req.Plate)
	brand := strings.TrimSpace(req.Brand)
	model := strings.TrimSpace(req.Model)

	if plate == "" {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "plate is required")
		return
	}
	if brand == "" {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "brand is required")
		return
	}
	if model == "" {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "model is required")
		return
	}
	if req.Mileage == nil {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "mileage is required")
		return
	}
	if *req.Mileage < 0 {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "mileage must not be negative")
		return
	}

	v := Vehicle{
		Plate:   plate,
		Brand:   brand,
		Model:   model,
		Mileage: *req.Mileage,
	}
	stored, err := h.store.UpsertByPlate(r.Context(), v)
	if errors.Is(err, ErrPlateTaken) {
		httpx.Error(w, http.StatusConflict, "plate_taken", "a vehicle with this plate already exists")
		return
	}
	if err != nil {
		log.Printf("create vehicle: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	httpx.JSON(w, http.StatusCreated, stored)
}
