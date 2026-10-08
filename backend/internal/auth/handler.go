package auth

import (
	"net/http"

	"workshop-api/internal/httpx"
)

// Handler serves the public login endpoint. Stub for this ticket.
type Handler struct {
	store  *Store
	issuer *TokenIssuer
}

// NewHandler builds the login handler on the shared store and token issuer.
func NewHandler(store *Store, issuer *TokenIssuer) *Handler {
	return &Handler{store: store, issuer: issuer}
}

// Login handles POST /api/auth/login (implemented by a later ticket).
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	httpx.NotImplemented(w, "POST /api/auth/login")
}
