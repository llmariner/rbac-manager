package auth

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

func TestCarryMetadata(t *testing.T) {
	md := map[string][]string{
		strings.ToLower(authHeader):    {"a0"},
		strings.ToLower(orgHeader):     {"o0"},
		strings.ToLower(projectHeader): {"p0"},
	}

	ctx := metadata.NewIncomingContext(context.Background(), md)
	ctx = CarryMetadata(ctx)

	got, ok := metadata.FromOutgoingContext(ctx)
	assert.True(t, ok)
	assert.ElementsMatch(t, []string{"a0"}, got.Get(authHeader))
	assert.ElementsMatch(t, []string{"o0"}, got.Get(orgHeader))
	assert.ElementsMatch(t, []string{"p0"}, got.Get(projectHeader))
}

func TestCarryMetadataFromHTTPHeader(t *testing.T) {
	header := map[string][]string{
		authHeader:    {"a0"},
		orgHeader:     {"o0"},
		projectHeader: {"p0"},
	}

	ctx := CarryMetadataFromHTTPHeader(context.Background(), header)
	got, ok := metadata.FromOutgoingContext(ctx)
	assert.True(t, ok)
	assert.ElementsMatch(t, []string{"a0"}, got.Get(authHeader))
	assert.ElementsMatch(t, []string{"o0"}, got.Get(orgHeader))
	assert.ElementsMatch(t, []string{"p0"}, got.Get(projectHeader))
}
