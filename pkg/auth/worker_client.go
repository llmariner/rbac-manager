package auth

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"google.golang.org/grpc/metadata"
)

const (
	envVarName = "LLMO_CLUSTER_REGISTRATION_KEY"
)

// AppendWorkerAuthorization appends the authorization to the context for a request
// from a worker cluster.
func AppendWorkerAuthorization(ctx context.Context) context.Context {
	key := os.Getenv(envVarName)
	auth := fmt.Sprintf("Bearer %s", key)
	return metadata.AppendToOutgoingContext(ctx, "Authorization", auth)
}

// AppendWorkerAuthorizationToHeader appends the authorization to the HTTP header.
func AppendWorkerAuthorizationToHeader(req *http.Request) {
	key := os.Getenv(envVarName)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", key))
}

// ValidateClusterRegistrationKey validates the cluster registration key.
func ValidateClusterRegistrationKey() error {
	key := os.Getenv(envVarName)
	if key == "" {
		return fmt.Errorf("environment variable %s is not set", envVarName)
	}
	if key == "default-cluster-registration-key-secret" {
		// This is the default key configured in https://github.com/llmariner/cluster-manager/blob/v1.5.3/deployments/server/values.yaml#L127.
		// Skip the validation for backward compatibility.
		return nil
	}
	if !strings.HasPrefix(key, "clusterkey-") {
		return fmt.Errorf("invalid cluster registration key: %s", key)
	}
	return nil
}
