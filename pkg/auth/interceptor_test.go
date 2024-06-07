package auth

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	v1 "github.com/llm-operator/rbac-manager/api/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestNewInterceptor(t *testing.T) {
	tests := []struct {
		name          string
		config        Config
		expectedError bool
	}{
		{
			name: "Valid Config with AccessResource",
			config: Config{
				RBACServerAddr: "localhost:50051",
				AccessResource: "resource",
			},
			expectedError: false,
		},
		{
			name: "Valid Config with GetAccessResourceForGRPCRequest and GetAccessResourceForHTTPRequest",
			config: Config{
				RBACServerAddr: "localhost:50051",
				GetAccessResourceForGRPCRequest: func(fullMethod string) string {
					return "resource"
				},
				GetAccessResourceForHTTPRequest: func(method string, u url.URL) string {
					return "resource"
				},
			},
			expectedError: false,
		},
		{
			name: "Invalid Config",
			config: Config{
				RBACServerAddr: "localhost:50051",
			},
			expectedError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			_, err := NewInterceptor(ctx, test.config)
			if test.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUnary(t *testing.T) {
	interceptor := &Interceptor{
		client: &fakeInternalServerClient{
			t:              t,
			wantResource:   "test.resource",
			wantCapability: "read",
		},
		getAccessResourceForGRPCRequest: func(fullMethod string) string {
			return "test.resource"
		},
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer token"))
	info := &grpc.UnaryServerInfo{FullMethod: "/test.server/GetTest"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) { return "ok", nil }
	interceptorFunc := interceptor.Unary()

	resp, err := interceptorFunc(ctx, nil, info, handler)
	assert.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

func TestInterceptHTTPRequest(t *testing.T) {
	interceptor := &Interceptor{
		client: &fakeInternalServerClient{
			t:              t,
			wantResource:   "resource",
			wantCapability: "read",
		},
		getAccessResourceForHTTPRequest: func(method string, u url.URL) string {
			return "resource"
		},
	}

	req := &http.Request{
		Method: http.MethodGet,
		Header: http.Header{"Authorization": []string{"Bearer token"}},
		URL:    &url.URL{},
	}

	statusCode, userInfo, err := interceptor.InterceptHTTPRequest(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, statusCode)
	assert.NotNil(t, userInfo)
}

type fakeInternalServerClient struct {
	t *testing.T

	wantResource   string
	wantCapability string
}

func (f *fakeInternalServerClient) Authorize(ctx context.Context, in *v1.AuthorizeRequest, opts ...grpc.CallOption) (*v1.AuthorizeResponse, error) {
	assert.Equal(f.t, f.wantResource, in.AccessResource)
	assert.Equal(f.t, f.wantCapability, in.Capability)

	return &v1.AuthorizeResponse{
		Authorized:   true,
		User:         &v1.User{Id: "u0"},
		Organization: &v1.Organization{Id: "o0"},
		Project:      &v1.Project{Id: "p0"},
	}, nil
}

func (f *fakeInternalServerClient) AuthorizeWorker(ctx context.Context, in *v1.AuthorizeWorkerRequest, opts ...grpc.CallOption) (*v1.AuthorizeWorkerResponse, error) {
	return &v1.AuthorizeWorkerResponse{
		Authorized: true,
		Cluster:    &v1.Cluster{Id: "c0"},
	}, nil
}
