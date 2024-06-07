package auth

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/grpc/metadata"
)

const (
	envVarname = "LLMO_CLUSTER_REGISTRATION_KEY"
)

// AppendWorkerAuthorization appends the authorization to the context for a request
// from a worker cluster.
func AppendWorkerAuthorization(ctx context.Context) context.Context {
	key := os.Getenv(envVarname)
	auth := fmt.Sprintf("Bearer %s", key)
	return metadata.AppendToOutgoingContext(ctx, "Authorization", auth)
}
