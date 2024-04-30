package auth

import (
	"context"
	"fmt"
	"strings"

	rbacv1 "github.com/llm-operator/rbac-manager/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// NewInterceptor creates a new Interceptor.
func NewInterceptor(ctx context.Context, rbacServerAddr, accessResource string) (*Interceptor, error) {
	conn, err := grpc.DialContext(ctx, rbacServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Interceptor{
		client:         rbacv1.NewRbacInternalServiceClient(conn),
		accessResource: accessResource,
	}, nil
}

// Interceptor is an authentication interceptor.
type Interceptor struct {
	client rbacv1.RbacInternalServiceClient

	accessResource string
}

// Unary returns a unary server interceptor.
func (a *Interceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		token, err := extractTokenFromContext(ctx)
		if err != nil {
			return nil, err
		}

		ms := strings.Split(info.FullMethod, "/")
		method := ms[len(ms)-1]

		var cap string
		switch {
		case strings.HasPrefix(method, "Get"),
			strings.HasPrefix(method, "List"):
			cap = "read"
		default:
			cap = "write"
		}

		user, err := a.client.Authorize(ctx, &rbacv1.AuthorizeRequest{
			Token: token,
			Scope: fmt.Sprintf("%s.%s", a.accessResource, cap),
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to authorize: %v", err)
		}
		if !user.Authorized {
			return nil, status.Errorf(codes.PermissionDenied, "permission denied")
		}

		return handler(ctx, req)
	}
}

func extractTokenFromContext(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Errorf(codes.InvalidArgument, "missing metadata")
	}
	auth := md["authorization"]
	if len(auth) < 1 {
		return "", status.Errorf(codes.Unauthenticated, "missing authorization")
	}
	return strings.TrimPrefix(auth[0], "Bearer "), nil
}
