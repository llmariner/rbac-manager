package auth

import (
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

// HeaderMatcher is a custom header matcher for GRPC gateway.
func HeaderMatcher(key string) (string, bool) {
	switch key {
	case orgHeader, projectHeader:
		return key, true
	default:
		return runtime.DefaultHeaderMatcher(key)
	}
}
