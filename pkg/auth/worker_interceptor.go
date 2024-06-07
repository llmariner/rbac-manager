package auth

import (
	"context"

	rbacv1 "github.com/llm-operator/rbac-manager/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// WorkerConfig is the configuration for a WorkerInterceptor.
type WorkerConfig struct {
	RBACServerAddr string
}

// NewWorkerInterceptor creates a new WorkerInterceptor.
func NewWorkerInterceptor(ctx context.Context, c WorkerConfig) (*WorkerInterceptor, error) {
	conn, err := grpc.DialContext(ctx, c.RBACServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &WorkerInterceptor{client: rbacv1.NewRbacInternalServiceClient(conn)}, nil
}

// WorkerInterceptor is an authentication interceptor for requests from worker clusters.
type WorkerInterceptor struct {
	client rbacv1.RbacInternalServiceClient
}

// Unary returns a unary server interceptor.
func (a *WorkerInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		token, err := ExtractTokenFromContext(ctx)
		if err != nil {
			return nil, err
		}
		aresp, err := a.client.AuthorizeWorker(ctx, &rbacv1.AuthorizeWorkerRequest{Token: token})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to authorize: %v", err)
		}
		if !aresp.Authorized {
			return nil, status.Errorf(codes.PermissionDenied, "permission denied")
		}

		ctx = AppendClusterInfoToContext(ctx, newClusterInfoFromAuthorizeResponse(aresp))
		return handler(ctx, req)
	}
}
