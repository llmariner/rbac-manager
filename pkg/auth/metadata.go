package auth

import (
	"context"
	"net/http"
	"strings"

	"google.golang.org/grpc/metadata"
)

// CarryMetadata extracts relevant metadata from the incoming context
// and appends that to the outgoing context.
func CarryMetadata(ctx context.Context) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}

	headers := []string{
		authHeader,
		orgHeader,
		projectHeader,
	}
	for _, header := range headers {
		key := strings.ToLower(header)
		v, ok := md[key]
		if !ok {
			continue
		}

		ctx = metadata.AppendToOutgoingContext(ctx, header, v[0])
	}
	return ctx
}

// CarryMetadataFromHTTPHeader extracts relevant metadata from HTTP headers
// and appends that to the outgoing context.
func CarryMetadataFromHTTPHeader(ctx context.Context, header http.Header) context.Context {
	keys := []string{
		authHeader,
		orgHeader,
		projectHeader,
	}
	for _, k := range keys {
		v, ok := header[k]
		if !ok {
			continue
		}
		ctx = metadata.AppendToOutgoingContext(ctx, k, v[0])
	}
	return ctx
}
