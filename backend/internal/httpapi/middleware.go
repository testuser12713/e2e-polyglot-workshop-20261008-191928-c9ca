package httpapi

import (
	"net/http"

	"workshop-api/internal/httpx"
)

// RequireAuth guards the workshop subtree. It is a stub for this ticket: the
// bearer middleware, the token issuance and the seeded employee are another
// ticket's work. Until then every protected request answers 501 with the
// uniform error body (never 500).
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpx.NotImplemented(w, "workshop authentication")
	})
}
