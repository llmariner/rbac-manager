package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	rbacv1 "github.com/llmariner/rbac-manager/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	capRead  = "read"
	capWrite = "write"

	authHeader = "Authorization"
	// orgHeader is the header key for organization ID.
	// The header defined in https://platform.openai.com/docs/api-reference/authentication
	// "OpenAI-Organization", but what we receive is "Openai-Organization".
	orgHeader = "Openai-Organization"
	// projectHeader is the header key for project ID.
	projectHeader = "Openai-Project"
)

// Config is the configuration for an Interceptor.
type Config struct {
	RBACServerAddr string

	// AccessResource is the static resource name to access. This value or GetAccessResource functions must be set.
	AccessResource string
	// GetAccessResourceForGRPCRequest is a function to get the resource name from a gRPC method.
	GetAccessResourceForGRPCRequest func(fullMethod string) string
	// GetAccessResourceForHTTPRequest is a function to get the resource name from an HTTP request method and URL.
	GetAccessResourceForHTTPRequest func(method string, url url.URL) string
}

// NewInterceptor creates a new Interceptor.
func NewInterceptor(ctx context.Context, c Config) (*Interceptor, error) {
	conn, err := grpc.DialContext(ctx, c.RBACServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	i := &Interceptor{client: rbacv1.NewRbacInternalServiceClient(conn)}

	if c.AccessResource == "" &&
		c.GetAccessResourceForGRPCRequest == nil &&
		c.GetAccessResourceForHTTPRequest == nil {
		return nil, fmt.Errorf("AccessResource or GetAccessResource functions must be set")
	}
	if c.AccessResource != "" {
		i.getAccessResourceForGRPCRequest = func(string) string { return c.AccessResource }
		i.getAccessResourceForHTTPRequest = func(string, url.URL) string { return c.AccessResource }
	} else {
		i.getAccessResourceForGRPCRequest = c.GetAccessResourceForGRPCRequest
		i.getAccessResourceForHTTPRequest = c.GetAccessResourceForHTTPRequest
	}

	return i, nil
}

// Interceptor is an authentication interceptor.
type Interceptor struct {
	client rbacv1.RbacInternalServiceClient

	getAccessResourceForGRPCRequest func(fullMethod string) string
	getAccessResourceForHTTPRequest func(method string, url url.URL) string
}

// Unary returns a unary server interceptor.
func (a *Interceptor) Unary(excludeMethods ...string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		for _, m := range excludeMethods {
			if info.FullMethod == m {
				return handler(ctx, req)
			}
		}

		token, err := ExtractTokenFromContext(ctx)
		if err != nil {
			return nil, err
		}

		ms := strings.Split(info.FullMethod, "/")
		method := ms[len(ms)-1]

		var cap string
		switch {
		case strings.HasPrefix(method, "Get"),
			strings.HasPrefix(method, "List"):
			cap = capRead
		default:
			cap = capWrite
		}

		orgID := extractOrgIDFromContext(ctx)
		projectID := extractProjectIDFromContext(ctx)

		resource := a.getAccessResourceForGRPCRequest(info.FullMethod)

		aresp, err := a.authorize(ctx, token, resource, cap, orgID, projectID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to authorize: %v", err)
		}
		if !aresp.Authorized {
			return nil, status.Errorf(codes.PermissionDenied, "permission denied")
		}

		// TODO(aya): revisit this after implement org management
		ctx = AppendUserInfoToContext(ctx, newUserInfoFromAuthorizeResponse(aresp))
		return handler(ctx, req)
	}
}

// InterceptHTTPRequest intercepts an HTTP request and returns an HTTP status code.
func (a *Interceptor) InterceptHTTPRequest(req *http.Request) (int, UserInfo, error) {
	token, found := extractTokenFromHeader(req.Header)
	if !found {
		return http.StatusUnauthorized, UserInfo{}, fmt.Errorf("missing authorization")
	}

	orgID := extractOrgIDFromHeader(req.Header)
	projectID := extractProjectIDFromHeader(req.Header)

	var cap string
	switch req.Method {
	case http.MethodGet:
		cap = capRead
	default:
		cap = capWrite
	}

	resource := a.getAccessResourceForHTTPRequest(req.Method, *req.URL)

	resp, err := a.authorize(req.Context(), token, resource, cap, orgID, projectID)
	if err != nil {
		return http.StatusInternalServerError, UserInfo{}, fmt.Errorf("failed to authorize: %v", err)
	}
	if !resp.Authorized {
		return http.StatusUnauthorized, UserInfo{}, fmt.Errorf("permission denied")
	}

	// TODO(kenji): Return user info.

	return http.StatusOK, newUserInfoFromAuthorizeResponse(resp), nil
}

func (a *Interceptor) authorize(
	ctx context.Context,
	token string,
	resource string,
	cap string,
	orgID string,
	projectID string,
) (*rbacv1.AuthorizeResponse, error) {
	return a.client.Authorize(ctx, &rbacv1.AuthorizeRequest{
		Token:          token,
		AccessResource: resource,
		Capability:     cap,
		OrganizationId: orgID,
		ProjectId:      projectID,
	})
}

// ExtractTokenFromContext extracts a token from a context.
func ExtractTokenFromContext(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Errorf(codes.InvalidArgument, "missing metadata")
	}
	auth := md[strings.ToLower(authHeader)]
	if len(auth) < 1 {
		return "", status.Errorf(codes.Unauthenticated, "missing authorization")
	}
	return strings.TrimPrefix(auth[0], "Bearer "), nil
}

func extractOrgIDFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	org := md[strings.ToLower(orgHeader)]
	if len(org) < 1 {
		return ""
	}
	return org[0]
}

func extractProjectIDFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	project := md[strings.ToLower(projectHeader)]
	if len(project) < 1 {
		return ""
	}
	return project[0]
}

func extractTokenFromHeader(header http.Header) (string, bool) {
	auth := header[authHeader]
	if len(auth) < 1 {
		return "", false
	}
	return strings.TrimPrefix(auth[0], "Bearer "), true
}

func extractOrgIDFromHeader(header http.Header) string {
	v := header[orgHeader]
	if len(v) < 1 {
		return ""
	}
	return v[0]
}

func extractProjectIDFromHeader(header http.Header) string {
	v := header[projectHeader]
	if len(v) < 1 {
		return ""
	}
	return v[0]
}
