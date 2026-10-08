package httpapi

import (
	"net/http"
	"os"
	"strings"

	"workshop-api/internal/auth"
	"workshop-api/internal/httpx"
)

// RequireAuth guards the workshop subtree. It verifies the bearer token signed
// with AUTH_SECRET, puts the employee into the request context and answers 401
// with the uniform error body when the header is missing or invalid.
func RequireAuth(next http.Handler) http.Handler {
	// AUTH_SECRET is read where the middleware is built (after config.Load),
	// never at package import time.
	issuer := auth.NewTokenIssuer(os.Getenv("AUTH_SECRET"))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r.Header.Get("Authorization"))
		if !ok {
			writeUnauthorized(w)
			return
		}

		employee, err := issuer.Parse(token)
		if err != nil {
			writeUnauthorized(w)
			return
		}

		next.ServeHTTP(w, r.WithContext(auth.WithEmployee(r.Context(), employee)))
	})
}

// bearerToken extracts the token from an "Authorization: Bearer <token>"
// header. The scheme is matched case-insensitively; the token must be present.
func bearerToken(header string) (string, bool) {
	scheme, token, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return "", false
	}
	return token, true
}

// writeUnauthorized answers 401 with the uniform error body.
func writeUnauthorized(w http.ResponseWriter) {
	httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing or invalid bearer token")
}
