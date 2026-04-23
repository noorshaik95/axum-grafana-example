package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMemoryRevoker(t *testing.T) {
	r := NewMemoryRevoker()
	ctx := context.Background()

	ok, err := r.IsRevoked(ctx, "jti-1")
	require.NoError(t, err)
	require.False(t, ok)

	require.NoError(t, r.Revoke(ctx, "jti-1", 50*time.Millisecond))
	ok, err = r.IsRevoked(ctx, "jti-1")
	require.NoError(t, err)
	require.True(t, ok)

	time.Sleep(60 * time.Millisecond)
	ok, err = r.IsRevoked(ctx, "jti-1")
	require.NoError(t, err)
	require.False(t, ok, "expired entry should be pruned on access")
}
