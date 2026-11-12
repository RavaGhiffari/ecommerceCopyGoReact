// Package authctx carries "who is making this request" through context.Context.
//
// The Authenticate middleware stores a Principal; handlers read it. Living in
// its own small package lets middleware and modules share it without import
// cycles, and keeps services free of Gin: they receive plain values.
package authctx

import (
	"context"
	"slices"

	"github.com/google/uuid"
)

const (
	RoleCustomer = "customer"
	RoleSeller   = "seller"
	RoleAdmin    = "admin"
)

// Principal is the authenticated user behind a request.
type Principal struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	Roles     []string // loaded from the database on every request, never from the client
}

func (p Principal) HasRole(role string) bool { return slices.Contains(p.Roles, role) }

type key struct{}

func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, key{}, p)
}

// FromContext returns the principal and true, or false for anonymous requests.
func FromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(key{}).(Principal)
	return p, ok
}
