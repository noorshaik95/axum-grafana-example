// Package tenants resolves tenant primary-admin contact details used when
// fanning out broadcasts or incident notifications. In production this hits
// tenant-service over gRPC; tests inject a StaticResolver.
package tenants

import "context"

// Admin is the minimal primary-admin profile used for email delivery.
type Admin struct {
	UserID   string
	Email    string
	TenantID string
}

// Resolver returns the primary admin(s) for one or more tenants.
type Resolver interface {
	// PrimaryAdmin returns the one admin address emails should fan out to
	// for a given tenant. Returns (nil, nil) when the tenant has no admin
	// registered — callers skip without erroring.
	PrimaryAdmin(ctx context.Context, tenantID string) (*Admin, error)
	// AllTenantIDs lists every tenant id. Used by the broadcast handler
	// when targets.all_tenants is true.
	AllTenantIDs(ctx context.Context) ([]string, error)
}

// StaticResolver is an in-memory implementation used for tests and dev.
type StaticResolver struct {
	Admins map[string]Admin // tenantID -> admin
}

// NewStaticResolver constructs a StaticResolver from a slice of admins.
func NewStaticResolver(admins []Admin) *StaticResolver {
	m := make(map[string]Admin, len(admins))
	for _, a := range admins {
		m[a.TenantID] = a
	}
	return &StaticResolver{Admins: m}
}

// PrimaryAdmin returns the configured admin for tenantID.
func (r *StaticResolver) PrimaryAdmin(_ context.Context, tenantID string) (*Admin, error) {
	a, ok := r.Admins[tenantID]
	if !ok {
		return nil, nil
	}
	return &a, nil
}

// AllTenantIDs returns the tenant ids registered with the resolver.
func (r *StaticResolver) AllTenantIDs(_ context.Context) ([]string, error) {
	ids := make([]string, 0, len(r.Admins))
	for id := range r.Admins {
		ids = append(ids, id)
	}
	return ids, nil
}
