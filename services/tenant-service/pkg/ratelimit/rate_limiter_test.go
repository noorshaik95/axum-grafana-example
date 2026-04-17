package ratelimit

import (
	"sync"
	"testing"
	"time"
)

func TestMemoryRateLimiter_AllowWithinLimit(t *testing.T) {
	limiter := &memoryRateLimiter{
		limits: make(map[string]*limitEntry),
	}

	// Test: 5 requests within limit
	key := "test-key"
	limit := 5
	window := 60

	for i := 0; i < 5; i++ {
		allowed, _ := limiter.checkLimit(key, limit, window)
		if !allowed {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}
}

func TestMemoryRateLimiter_ExceedsLimit(t *testing.T) {
	limiter := &memoryRateLimiter{
		limits: make(map[string]*limitEntry),
	}

	key := "test-key"
	limit := 5
	window := 60

	// Make 5 requests (at limit)
	for i := 0; i < 5; i++ {
		_, _ = limiter.checkLimit(key, limit, window)
	}

	// 6th request should be denied
	allowed, _ := limiter.checkLimit(key, limit, window)
	if allowed {
		t.Error("Request should be denied after exceeding limit")
	}
}

func TestMemoryRateLimiter_WindowReset(t *testing.T) {
	limiter := &memoryRateLimiter{
		limits: make(map[string]*limitEntry),
	}

	key := "test-key"
	limit := 3
	window := 1 // 1 second window

	// Make 3 requests (at limit)
	for i := 0; i < 3; i++ {
		_, _ = limiter.checkLimit(key, limit, window)
	}

	// Should be denied
	allowed, _ := limiter.checkLimit(key, limit, window)
	if allowed {
		t.Error("Request should be denied")
	}

	// Wait for window to expire
	time.Sleep(1100 * time.Millisecond)

	// Should be allowed after window reset
	allowed2, _ := limiter.checkLimit(key, limit, window)
	if !allowed2 {
		t.Error("Request should be allowed after window reset")
	}
}

func TestMemoryRateLimiter_MultipleKeys(t *testing.T) {
	limiter := &memoryRateLimiter{
		limits: make(map[string]*limitEntry),
	}

	limit := 3
	window := 60

	// Different keys should have independent limits
	for i := 0; i < 3; i++ {
		allowed1, _ := limiter.checkLimit("key1", limit, window)
		if !allowed1 {
			t.Error("key1 should be allowed")
		}
		allowed2, _ := limiter.checkLimit("key2", limit, window)
		if !allowed2 {
			t.Error("key2 should be allowed")
		}
	}

	// Both keys should now be at limit
	allowed3, _ := limiter.checkLimit("key1", limit, window)
	if allowed3 {
		t.Error("key1 should be denied")
	}
	allowed4, _ := limiter.checkLimit("key2", limit, window)
	if allowed4 {
		t.Error("key2 should be denied")
	}
}

func TestMemoryRateLimiter_Concurrent(t *testing.T) {
	limiter := &memoryRateLimiter{
		limits: make(map[string]*limitEntry),
	}

	key := "concurrent-key"
	limit := 100
	window := 60
	numGoroutines := 10
	requestsPerGoroutine := 10

	var wg sync.WaitGroup
	allowed := make([]bool, numGoroutines*requestsPerGoroutine)
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				result, _ := limiter.checkLimit(key, limit, window)
				mu.Lock()
				allowed[goroutineID*requestsPerGoroutine+j] = result
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// Count allowed requests
	allowedCount := 0
	for _, a := range allowed {
		if a {
			allowedCount++
		}
	}

	// Should allow exactly the limit (100)
	if allowedCount != limit {
		t.Errorf("Expected %d allowed requests, got %d", limit, allowedCount)
	}
}

func TestRateLimiter_AllowCreateTenant(t *testing.T) {
	config := &Config{
		CreateTenantLimit:  5,
		CreateTenantWindow: 3600,
		OperationLimit:     100,
		OperationWindow:    60,
	}

	limiter, err := NewRateLimiter(config)
	if err != nil {
		t.Fatalf("Failed to create rate limiter: %v", err)
	}

	clientIP := "192.168.1.1"

	// Test: 5 tenant creations should be allowed
	for i := 0; i < 5; i++ {
		allowed, _, err := limiter.AllowCreateTenant(clientIP)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !allowed {
			t.Errorf("Tenant creation %d should be allowed", i+1)
		}
	}

	// 6th creation should be denied
	allowed, retryAfter, err := limiter.AllowCreateTenant(clientIP)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if allowed {
		t.Error("6th tenant creation should be denied")
	}
	if retryAfter == 0 {
		t.Error("Retry-after should be set")
	}
}

func TestRateLimiter_AllowOperation(t *testing.T) {
	config := &Config{
		CreateTenantLimit:  5,
		CreateTenantWindow: 3600,
		OperationLimit:     10,
		OperationWindow:    60,
	}

	limiter, err := NewRateLimiter(config)
	if err != nil {
		t.Fatalf("Failed to create rate limiter: %v", err)
	}

	clientIP := "192.168.1.2"
	operation := "get_tenant"

	// Test: 10 operations should be allowed
	for i := 0; i < 10; i++ {
		allowed, _, err := limiter.AllowOperation(clientIP, operation)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !allowed {
			t.Errorf("Operation %d should be allowed", i+1)
		}
	}

	// 11th operation should be denied
	allowed, _, err := limiter.AllowOperation(clientIP, operation)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if allowed {
		t.Error("11th operation should be denied")
	}
}

func TestRateLimiter_DifferentIPsIndependent(t *testing.T) {
	config := &Config{
		CreateTenantLimit:  3,
		CreateTenantWindow: 3600,
		OperationLimit:     100,
		OperationWindow:    60,
	}

	limiter, err := NewRateLimiter(config)
	if err != nil {
		t.Fatalf("Failed to create rate limiter: %v", err)
	}

	// Different IPs should have independent limits
	for i := 0; i < 3; i++ {
		allowed1, _, _ := limiter.AllowCreateTenant("192.168.1.1")
		allowed2, _, _ := limiter.AllowCreateTenant("192.168.1.2")

		if !allowed1 || !allowed2 {
			t.Error("Both IPs should be allowed")
		}
	}

	// Both should be at limit now
	allowed1, _, _ := limiter.AllowCreateTenant("192.168.1.1")
	allowed2, _, _ := limiter.AllowCreateTenant("192.168.1.2")

	if allowed1 || allowed2 {
		t.Error("Both IPs should be denied")
	}
}

func TestMemoryRateLimiter_Cleanup(t *testing.T) {
	limiter := &memoryRateLimiter{
		limits: make(map[string]*limitEntry),
	}

	// Add some entries
	_, _ = limiter.checkLimit("key1", 5, 1) // 1 second window
	_, _ = limiter.checkLimit("key2", 5, 1)
	_, _ = limiter.checkLimit("key3", 5, 1)

	if len(limiter.limits) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(limiter.limits))
	}

	// Wait for expiry
	time.Sleep(1100 * time.Millisecond)

	// Trigger cleanup manually (in production, this runs in background)
	limiter.mu.Lock()
	now := time.Now()
	for key, entry := range limiter.limits {
		entry.mu.Lock()
		if now.After(entry.windowEnd) {
			delete(limiter.limits, key)
		}
		entry.mu.Unlock()
	}
	limiter.mu.Unlock()

	if len(limiter.limits) != 0 {
		t.Errorf("Expected 0 entries after cleanup, got %d", len(limiter.limits))
	}
}

func TestNewRateLimiter_WithRedis(t *testing.T) {
	config := &Config{
		CreateTenantLimit:   5,
		CreateTenantWindow:  3600,
		OperationLimit:      100,
		OperationWindow:     60,
		RedisAddr:           "localhost:6379",
		RedisPassword:       "",
		RedisDB:             0,
	}

	limiter, err := NewRateLimiter(config)
	if err != nil {
		t.Fatalf("Failed to create rate limiter: %v", err)
	}

	if limiter == nil {
		t.Fatal("Expected rate limiter to be created")
	}

	// Test basic functionality (will use fallback if Redis isn't available)
	allowed, _, err := limiter.AllowCreateTenant("test-ip")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !allowed {
		t.Error("First request should be allowed")
	}
}

func TestNewRateLimiter_WithoutRedis(t *testing.T) {
	config := &Config{
		CreateTenantLimit:   5,
		CreateTenantWindow:  3600,
		OperationLimit:      100,
		OperationWindow:     60,
		// No Redis configured
		RedisAddr:     "",
		RedisPassword: "",
		RedisDB:       0,
	}

	limiter, err := NewRateLimiter(config)
	if err != nil {
		t.Fatalf("Failed to create rate limiter: %v", err)
	}

	if limiter == nil {
		t.Fatal("Expected rate limiter to be created")
	}

	// Should use in-memory fallback
	allowed, retryAfter, err := limiter.AllowCreateTenant("test-ip")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !allowed {
		t.Error("First request should be allowed")
	}

	if retryAfter != 0 {
		t.Errorf("Expected retry-after 0 for allowed request, got %d", retryAfter)
	}
}

func TestRateLimiter_AllowOperationWithLimit(t *testing.T) {
	config := &Config{
		CreateTenantLimit:   5,
		CreateTenantWindow:  3600,
		OperationLimit:      3,
		OperationWindow:     60,
		RedisAddr:           "",
	}

	limiter, err := NewRateLimiter(config)
	if err != nil {
		t.Fatalf("Failed to create rate limiter: %v", err)
	}

	clientIP := "192.168.1.100"
	operation := "GetTenant"

	// Test: 3 operations should be allowed
	for i := 0; i < 3; i++ {
		allowed, _, err := limiter.AllowOperation(clientIP, operation)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !allowed {
			t.Errorf("Operation %d should be allowed", i+1)
		}
	}

	// 4th operation should be denied
	allowed, retryAfter, err := limiter.AllowOperation(clientIP, operation)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if allowed {
		t.Error("4th operation should be denied")
	}
	if retryAfter == 0 {
		t.Error("Retry-after should be set for rate limited request")
	}
}

func TestMemoryRateLimiter_Cleanup_ActiveEntries(t *testing.T) {
	limiter := &memoryRateLimiter{
		limits: make(map[string]*limitEntry),
	}

	// Add some active entries (not expired)
	future := time.Now().Add(10 * time.Minute)
	limiter.limits["key1"] = &limitEntry{
		count:     1,
		windowEnd: future,
	}
	limiter.limits["key2"] = &limitEntry{
		count:     2,
		windowEnd: future,
	}

	// Run cleanup manually
	limiter.mu.Lock()
	now := time.Now()
	for key, entry := range limiter.limits {
		entry.mu.Lock()
		if now.After(entry.windowEnd) {
			delete(limiter.limits, key)
		}
		entry.mu.Unlock()
	}
	limiter.mu.Unlock()

	// Active entries should not be deleted
	if len(limiter.limits) != 2 {
		t.Errorf("Expected 2 active entries, got %d", len(limiter.limits))
	}
}

func TestRateLimiter_DifferentOperations(t *testing.T) {
	config := &Config{
		CreateTenantLimit:   5,
		CreateTenantWindow:  3600,
		OperationLimit:      2,
		OperationWindow:     60,
		RedisAddr:           "",
	}

	limiter, err := NewRateLimiter(config)
	if err != nil {
		t.Fatalf("Failed to create rate limiter: %v", err)
	}

	clientIP := "192.168.1.200"

	// Use up quota for "GetTenant"
	_, _, _ = limiter.AllowOperation(clientIP, "GetTenant")
	_, _, _ = limiter.AllowOperation(clientIP, "GetTenant")

	// GetTenant should be rate limited
	allowed, _, _ := limiter.AllowOperation(clientIP, "GetTenant")
	if allowed {
		t.Error("GetTenant should be rate limited")
	}

	// But ListTenants should still be allowed (different operation)
	allowed, _, _ = limiter.AllowOperation(clientIP, "ListTenants")
	if !allowed {
		t.Error("ListTenants should still be allowed (different operation key)")
	}
}
