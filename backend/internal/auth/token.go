package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrInvalidToken is returned by Parse for a token that is malformed, carries
// a bad signature or has expired.
var ErrInvalidToken = errors.New("auth: invalid bearer token")

// tokenTTL is how long an issued bearer token stays valid.
const tokenTTL = 12 * time.Hour

// tokenClaims is the signed payload: the employee id and the expiry (Unix
// seconds, UTC).
type tokenClaims struct {
	Sub int64 `json:"sub"`
	Exp int64 `json:"exp"`
}

// TokenIssuer signs and verifies bearer tokens with HMAC-SHA256 over the
// secret from the environment (AUTH_SECRET). The token is
// base64url(payload).base64url(hmac).
type TokenIssuer struct {
	secret []byte
}

// NewTokenIssuer builds an issuer with the configured signing secret.
func NewTokenIssuer(secret string) *TokenIssuer {
	return &TokenIssuer{secret: []byte(secret)}
}

// Issue returns a signed bearer token for the employee.
func (t *TokenIssuer) Issue(employee Employee) (string, error) {
	claims := tokenClaims{
		Sub: employee.ID,
		Exp: time.Now().Add(tokenTTL).Unix(),
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("auth: encode token payload: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return encoded + "." + t.sign(encoded), nil
}

// Parse verifies a bearer token and returns the employee it belongs to. Only
// the employee id is carried by the token; name and e-mail are looked up when
// needed.
func (t *TokenIssuer) Parse(token string) (Employee, error) {
	payload, signature, found := strings.Cut(token, ".")
	if !found || payload == "" || signature == "" {
		return Employee{}, ErrInvalidToken
	}
	if !hmac.Equal([]byte(t.sign(payload)), []byte(signature)) {
		return Employee{}, ErrInvalidToken
	}

	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return Employee{}, ErrInvalidToken
	}
	var claims tokenClaims
	if err := json.Unmarshal(raw, &claims); err != nil {
		return Employee{}, ErrInvalidToken
	}
	if claims.Sub <= 0 {
		return Employee{}, ErrInvalidToken
	}
	if time.Now().Unix() >= claims.Exp {
		return Employee{}, ErrInvalidToken
	}
	return Employee{ID: claims.Sub}, nil
}

// sign computes the base64url HMAC-SHA256 of the encoded payload.
func (t *TokenIssuer) sign(encoded string) string {
	mac := hmac.New(sha256.New, t.secret)
	mac.Write([]byte(encoded))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// employeeContextKey is the private type of the request-context key, so no
// other package can collide with it.
type employeeContextKey struct{}

// WithEmployee stores the authenticated employee in the request context.
func WithEmployee(ctx context.Context, employee Employee) context.Context {
	return context.WithValue(ctx, employeeContextKey{}, employee)
}

// EmployeeFromContext returns the authenticated employee installed by
// RequireAuth, if any.
func EmployeeFromContext(ctx context.Context) (Employee, bool) {
	employee, ok := ctx.Value(employeeContextKey{}).(Employee)
	return employee, ok
}
