package auth

import (
	"errors"

	// bcrypt is installed here (declared in go.mod) for the login ticket that
	// hashes and verifies employee passwords; the blank import keeps it a
	// direct dependency until then.
	_ "golang.org/x/crypto/bcrypt"
)

// errNotImplemented marks the issuer as inert for this ticket. The login and
// bearer-middleware ticket fills these in.
var errNotImplemented = errors.New("auth: token issuer is not implemented yet")

// TokenIssuer signs and verifies bearer tokens. Its bodies are intentionally
// inert in this skeleton ticket; the signatures are the shared contract.
type TokenIssuer struct {
	secret []byte
}

// NewTokenIssuer builds an issuer with the configured signing secret.
func NewTokenIssuer(secret string) *TokenIssuer {
	return &TokenIssuer{secret: []byte(secret)}
}

// Issue returns a signed bearer token for the employee.
func (t *TokenIssuer) Issue(employee Employee) (string, error) {
	return "", errNotImplemented
}

// Parse verifies a bearer token and returns the employee it belongs to.
func (t *TokenIssuer) Parse(token string) (Employee, error) {
	return Employee{}, errNotImplemented
}
