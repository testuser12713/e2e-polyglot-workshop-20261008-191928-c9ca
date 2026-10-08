package httpapi

import "workshop-api/internal/httpx"

// The canonical response helpers live in internal/httpx so the feature handler
// packages can use them without an import cycle (httpapi imports those
// packages). These aliases keep the httpapi.respond.go surface the router
// uses.
type (
	// ErrorBody is the uniform error envelope.
	ErrorBody = httpx.ErrorBody
	// ErrorDetail is the code/message pair.
	ErrorDetail = httpx.ErrorDetail
)

// JSON writes v as a JSON body with the given status code.
var JSON = httpx.JSON

// Error writes the uniform error body.
var Error = httpx.Error

// NotFound writes the uniform 404 body.
var NotFound = httpx.NotFound

// NotImplemented writes the uniform 501 body of a stub route.
var NotImplemented = httpx.NotImplemented
