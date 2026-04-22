package cache

import (
	"testing"
	"time"
)

func TestMemory_SetGet(t *testing.T) {
	c := NewMemory()
	c.Set("k", []byte("v"), time.Minute)

	got, ok := c.Get("k")
	if !ok {
		t.Fatal("expected hit")
	}
	if string(got) != "v" {
		t.Fatalf("expected v, got %s", got)
	}
}

func TestMemory_Miss(t *testing.T) {
	c := NewMemory()
	if _, ok := c.Get("missing"); ok {
		t.Fatal("expected miss")
	}
}

func TestMemory_Expiry(t *testing.T) {
	c := NewMemory()
	now := time.Now()
	c.setClock(func() time.Time { return now })

	c.Set("k", []byte("v"), 10*time.Second)
	if _, ok := c.Get("k"); !ok {
		t.Fatal("expected hit before expiry")
	}

	c.setClock(func() time.Time { return now.Add(11 * time.Second) })
	if _, ok := c.Get("k"); ok {
		t.Fatal("expected miss after expiry")
	}
}

func TestMemory_Delete(t *testing.T) {
	c := NewMemory()
	c.Set("k", []byte("v"), time.Minute)
	c.Delete("k")
	if _, ok := c.Get("k"); ok {
		t.Fatal("expected miss after delete")
	}
}

func TestMemory_DeletePrefix(t *testing.T) {
	c := NewMemory()
	c.Set("tenant:a:roster:c1", []byte("1"), time.Minute)
	c.Set("tenant:a:roster:c2", []byte("2"), time.Minute)
	c.Set("tenant:b:roster:c3", []byte("3"), time.Minute)

	c.DeletePrefix("tenant:a:roster:")

	if _, ok := c.Get("tenant:a:roster:c1"); ok {
		t.Fatal("expected c1 evicted")
	}
	if _, ok := c.Get("tenant:a:roster:c2"); ok {
		t.Fatal("expected c2 evicted")
	}
	if _, ok := c.Get("tenant:b:roster:c3"); !ok {
		t.Fatal("expected c3 retained")
	}
}

func TestMemory_SetCopiesValue(t *testing.T) {
	c := NewMemory()
	v := []byte("hello")
	c.Set("k", v, time.Minute)
	v[0] = 'X'

	got, _ := c.Get("k")
	if string(got) != "hello" {
		t.Fatalf("expected hello, got %s", got)
	}
}
