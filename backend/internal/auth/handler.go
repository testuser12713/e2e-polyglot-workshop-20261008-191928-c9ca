package auth

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"workshop-api/internal/httpx"
)

// Handler serves the public login endpoint.
type Handler struct {
	store  *Store
	issuer *TokenIssuer
}

// NewHandler builds the login handler on the shared store and token issuer.
func NewHandler(store *Store, issuer *TokenIssuer) *Handler {
	return &Handler{store: store, issuer: issuer}
}

// loginRequest is the POST /api/auth/login body.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// loginResponse is the successful 200 body: the bearer token and the public
// employee fields (never the password hash).
type loginResponse struct {
	Token    string   `json:"token"`
	Employee Employee `json:"employee"`
}

// Login handles POST /api/auth/login. It verifies the e-mail and bcrypt
// password against the employees table and answers 200 with a token or 401
// with the uniform error body.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_request", "request body must be valid JSON")
		return
	}
	if req.Email == "" || req.Password == "" {
		log.Printf("auth: login rejected (empty email or password field)")
		writeInvalidCredentials(w)
		return
	}

	employee, err := h.store.FindByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, ErrEmployeeNotFound) {
			log.Printf("auth: login rejected (unknown email)")
			writeInvalidCredentials(w)
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(employee.PasswordHash), []byte(req.Password)); err != nil {
		log.Printf("auth: login rejected (password mismatch for known email)")
		writeInvalidCredentials(w)
		return
	}

	token, err := h.issuer.Issue(employee)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	httpx.JSON(w, http.StatusOK, loginResponse{Token: token, Employee: employee})
}

// writeInvalidCredentials answers 401 with the uniform error body. The same
// answer is used for an unknown e-mail and a wrong password so the endpoint
// never reveals which accounts exist.
func writeInvalidCredentials(w http.ResponseWriter) {
	httpx.Error(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
}
