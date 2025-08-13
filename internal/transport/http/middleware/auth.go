package middleware

import "context"

// ctxKey is an unexported type for context keys defined in this package.
// This prevents collisions with context keys defined in other packages.
type ctxKey int

const (
	// ctxKeyIdentity is the context key for the Identity struct.
	ctxKeyIdentity ctxKey = iota
)

// Identity represents the authenticated user's identity.
type Identity struct {
	UserID   int64
	Plan     string
	APIKeyID *int64
	RateRPM  *int
}

// IdentityFrom extracts the Identity struct from the context.
func IdentityFrom(ctx context.Context) (Identity, bool) {
	v := ctx.Value(ctxKeyIdentity)
	id, ok := v.(Identity)
	return id, ok
}
