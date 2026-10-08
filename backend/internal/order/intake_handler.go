package order

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"workshop-api/internal/customer"
	"workshop-api/internal/httpx"
	"workshop-api/internal/vehicle"
)

// maxIntakeBody caps the appointment request body so a malformed or oversized
// payload cannot exhaust the server.
const maxIntakeBody = 1 << 20

// IntakeHandler serves the public appointment request.
type IntakeHandler struct {
	store *Store
}

// NewIntakeHandler builds the intake handler on the shared order store.
func NewIntakeHandler(store *Store) *IntakeHandler {
	return &IntakeHandler{store: store}
}

// Create handles POST /api/appointments: it validates the request, upserts the
// customer and the vehicle through the existing helpers, creates the order in
// status requested and answers 201 {"order_id","order_number"}.
func (h *IntakeHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req IntakeRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxIntakeBody))
	if err := decoder.Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_request", "request body must be a valid JSON object")
		return
	}
	if err := req.Validate(); err != nil {
		httpx.Error(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	ctx := r.Context()
	name := strings.TrimSpace(req.Name)
	email := strings.TrimSpace(req.Email)
	phone := strings.TrimSpace(req.Phone)
	plate := strings.TrimSpace(req.Plate)

	customerStore := customer.NewStore(h.store.DB)
	cust, err := customerStore.UpsertCustomer(ctx, name, email, phone)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "could not store customer")
		return
	}

	vehicleStore := vehicle.NewStore(h.store.DB)
	veh, err := vehicleStore.UpsertByPlate(ctx, vehicle.Vehicle{
		Plate:   plate,
		Brand:   strings.TrimSpace(req.Brand),
		Model:   strings.TrimSpace(req.Model),
		Mileage: *req.Mileage,
	})
	if errors.Is(err, vehicle.ErrPlateTaken) {
		// The plate already belongs to a known vehicle: reuse it instead of
		// failing the appointment.
		veh, err = h.store.findVehicleByPlate(ctx, plate)
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "could not store vehicle")
		return
	}

	preferredDate := strings.TrimSpace(req.PreferredDate)
	created, err := h.store.CreateOrder(ctx, CreateOrderParams{
		CustomerID:    cust.ID,
		VehicleID:     veh.ID,
		PreferredDate: &preferredDate,
		Description:   strings.TrimSpace(req.Description),
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "could not create order")
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]any{
		"order_id":     created.ID,
		"order_number": created.OrderNumber,
	})
}
