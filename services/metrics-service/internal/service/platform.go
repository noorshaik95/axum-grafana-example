package service

import (
	"context"
	"encoding/json"
	"time"

	"slate/services/metrics-service/internal/cache"
	"slate/services/metrics-service/internal/incident"
	"slate/services/metrics-service/internal/models"
)

// PlatformCacheKey is the Redis key for the extended platform metrics blob.
const PlatformCacheKey = "platform:metrics:platform"

// PlatformCacheTTL is the TTL for the cached platform stats.
const PlatformCacheTTL = 5 * time.Minute

// TenantMAU holds monthly active users for a single tenant.
type TenantMAU struct {
	TenantID string `json:"tenant_id"`
	MAU      int    `json:"mau"`
}

// ExtendedPlatformMetrics augments the base platform metrics with
// W12.4 additions: per-tenant MAU, 30-day signups, uptime, active incidents.
type ExtendedPlatformMetrics struct {
	*models.PlatformMetrics
	TenantMAU       []TenantMAU `json:"tenant_mau"`
	SignupsLast30d  int         `json:"signups_last_30d"`
	UptimePct       float64     `json:"uptime_pct"`
	ActiveIncidents int         `json:"active_incidents"`
	GeneratedAt     time.Time   `json:"generated_at"`
}

// PlatformSource produces the raw inputs for ExtendedPlatformMetrics. The
// repository satisfies this interface; tests can provide fakes.
type PlatformSource interface {
	GetPlatformMetrics(ctx context.Context) (*models.PlatformMetrics, error)
	GetTenantMAU(ctx context.Context) ([]TenantMAU, error)
	GetSignupsLast30d(ctx context.Context) (int, error)
	GetUptimePct(ctx context.Context) (float64, error)
}

// PlatformService returns cached extended platform metrics.
type PlatformService struct {
	source  PlatformSource
	incid   incident.Client
	cache   cache.Cache
	ttl     time.Duration
	now     func() time.Time
}

// NewPlatformService constructs a PlatformService.
func NewPlatformService(src PlatformSource, incid incident.Client, c cache.Cache) *PlatformService {
	if incid == nil {
		incid = incident.NewStub()
	}
	return &PlatformService{
		source: src,
		incid:  incid,
		cache:  c,
		ttl:    PlatformCacheTTL,
		now:    time.Now,
	}
}

// Get returns the extended platform metrics, using the cache when fresh.
func (s *PlatformService) Get(ctx context.Context) (*ExtendedPlatformMetrics, error) {
	if s.cache != nil {
		if raw, ok := s.cache.Get(PlatformCacheKey); ok {
			var cached ExtendedPlatformMetrics
			if err := json.Unmarshal(raw, &cached); err == nil {
				return &cached, nil
			}
		}
	}

	base, err := s.source.GetPlatformMetrics(ctx)
	if err != nil {
		return nil, err
	}
	tenantMAU, err := s.source.GetTenantMAU(ctx)
	if err != nil {
		return nil, err
	}
	signups, err := s.source.GetSignupsLast30d(ctx)
	if err != nil {
		return nil, err
	}
	uptime, err := s.source.GetUptimePct(ctx)
	if err != nil {
		return nil, err
	}
	incidents, err := s.incid.ActiveCount(ctx)
	if err != nil {
		// Incident service being down should not break platform stats.
		incidents = 0
	}

	out := &ExtendedPlatformMetrics{
		PlatformMetrics: base,
		TenantMAU:       tenantMAU,
		SignupsLast30d:  signups,
		UptimePct:       uptime,
		ActiveIncidents: incidents,
		GeneratedAt:     s.now(),
	}
	if s.cache != nil {
		if buf, err := json.Marshal(out); err == nil {
			s.cache.Set(PlatformCacheKey, buf, s.ttl)
		}
	}
	return out, nil
}

// Invalidate clears the cached platform metrics.
func (s *PlatformService) Invalidate() {
	if s.cache != nil {
		s.cache.Delete(PlatformCacheKey)
	}
}
