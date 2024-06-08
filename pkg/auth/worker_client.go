package auth

import (
	"context"
	"fmt"
	"net/http"
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

// AppendWorkerAuthorizationToHeader appends the authorization to the HTTP header.
func AppendWorkerAuthorizationToHeader(req *http.Request) {
	key := os.Getenv(envVarname)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", key))
}
