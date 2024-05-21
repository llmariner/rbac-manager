package auth

import (
	"context"
	"strings"

	"google.golang.org/grpc/metadata"
)

// CarryMetadata extracts relevant metadata from the incoming context
// and append that to the outgoing context.
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
