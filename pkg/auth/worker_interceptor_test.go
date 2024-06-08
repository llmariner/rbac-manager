package auth

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestWorkerUnary(t *testing.T) {
	interceptor := &WorkerInterceptor{
		client: &fakeInternalServerClient{
			t: t,
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

func TestWorkerInterceptHTTPRequest(t *testing.T) {
	interceptor := &WorkerInterceptor{
		client: &fakeInternalServerClient{
			t: t,
		},
	}

	req := &http.Request{
		Method: http.MethodGet,
		Header: http.Header{"Authorization": []string{"Bearer token"}},
		URL:    &url.URL{},
	}

	statusCode, clusterInfo, err := interceptor.InterceptHTTPRequest(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, statusCode)
	assert.NotNil(t, clusterInfo)
}
