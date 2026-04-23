package userauth

import (
	"context"
	"fmt"
	"strings"

	commontracing "slate/libs/common-go/tracing"
	userpb "slate/services/user-auth-service/api/proto"

	"google.golang.org/grpc"
)

// GRPCResolver calls user-auth-service's ResolveUsername RPC to map parsed
// `@username` tokens to canonical user IDs. Satisfies the Resolver interface.
//
// Response-key convention (per plan/CONTRACTS.md § user.UserService ResolveUsername):
// map keys are lowercase-normalized. We preserve the caller's original casing
// in our returned map so downstream code can render the original display form,
// but we look up against the lowercase key the server returned.
type GRPCResolver struct {
	client userpb.UserServiceClient
}

// NewGRPCResolver wraps a live UserService client.
func NewGRPCResolver(conn grpc.ClientConnInterface) *GRPCResolver {
	return &GRPCResolver{client: userpb.NewUserServiceClient(conn)}
}

// NewGRPCResolverWithClient is useful for tests that inject a fake client.
func NewGRPCResolverWithClient(client userpb.UserServiceClient) *GRPCResolver {
	return &GRPCResolver{client: client}
}

func (g *GRPCResolver) ResolveUsernames(ctx context.Context, tenantID string, usernames []string) (map[string]string, error) {
	if len(usernames) == 0 {
		return map[string]string{}, nil
	}

	// Propagate trace context + correlation headers outbound.
	ctx = commontracing.InjectTraceparent(ctx)

	resp, err := g.client.ResolveUsername(ctx, &userpb.ResolveUsernameRequest{
		Usernames: usernames,
		TenantId:  tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("user-auth ResolveUsername: %w", err)
	}

	// Server returns lowercase-normalized keys; re-key by the caller's original
	// username so the rest of the pipeline doesn't have to rediscover casing.
	out := make(map[string]string, len(usernames))
	for _, orig := range usernames {
		if id, ok := resp.GetUserIdsByUsername()[strings.ToLower(orig)]; ok && id != "" {
			out[orig] = id
		}
	}
	return out, nil
}
