package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"

	"slate/services/incident-service/internal/models"
)

// Cache wraps go-redis. When client is nil, methods are no-ops — this lets
// tests and REDIS_ENABLED=false installs run without Redis.
type Cache struct {
	client *redis.Client
}

// ActiveIncidentsKey is the single key that holds the JSON-encoded list of
// currently open + watching incidents, rewritten on every status change.
const ActiveIncidentsKey = "platform:incidents:active"

// ActiveIncidentsTTL per plan W4.4.
const ActiveIncidentsTTL = 30 * time.Second

func New(client *redis.Client) *Cache {
	return &Cache{client: client}
}

// SetActive replaces the cached active incidents list with a fresh snapshot.
// Called after every status transition.
func (c *Cache) SetActive(ctx context.Context, incidents []*models.Incident) error {
	if c == nil || c.client == nil {
		return nil
	}
	b, err := json.Marshal(incidents)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, ActiveIncidentsKey, b, ActiveIncidentsTTL).Err()
}

// GetActive returns cached incidents; (nil, false, nil) on cache miss.
func (c *Cache) GetActive(ctx context.Context) ([]*models.Incident, bool, error) {
	if c == nil || c.client == nil {
		return nil, false, nil
	}
	b, err := c.client.Get(ctx, ActiveIncidentsKey).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var out []*models.Incident
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, false, err
	}
	return out, true, nil
}

// Invalidate removes the active-incidents cache key. Called when the set of
// active incidents is uncertain (e.g. concurrent writes).
func (c *Cache) Invalidate(ctx context.Context) error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Del(ctx, ActiveIncidentsKey).Err()
}
