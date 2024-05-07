package server

import (
	"context"
	"fmt"
	"net"

	v1 "github.com/llm-operator/rbac-manager/api/v1"
	"github.com/llm-operator/rbac-manager/server/internal/apikey"
	"github.com/llm-operator/rbac-manager/server/internal/config"
	"github.com/llm-operator/rbac-manager/server/internal/dex"
	"google.golang.org/grpc"
)

type apikeyCache interface {
	GetAPIKey(keyID string) (*apikey.K, bool)
}

type tokenIntrospector interface {
	TokenIntrospect(token string) (*dex.Introspection, error)
}

// New returns a new Server.
func New(issuerURL string, apiKeyCache apikeyCache, debug *config.DebugConfig) *Server {
	return &Server{
		tokenIntrospector: dex.NewDefaultClient(issuerURL),

		apiKeyCache: apiKeyCache,

		userOrgMapper:    debug.UserOrgMap,
		orgRoleMapper:    debug.OrgRoleMap,
		roleScopesMapper: debug.RoleScopesMap,
	}
}

// Server implementes the gRPC interface.
type Server struct {
	v1.UnimplementedRbacInternalServiceServer

	tokenIntrospector tokenIntrospector

	apiKeyCache apikeyCache

	// TODO(aya): replace this after implementing the user-manager.
	userOrgMapper    map[string]string
	orgRoleMapper    map[string]string
	roleScopesMapper map[string][]string
}

// Run starts the gRPC server.
func (s *Server) Run(ctx context.Context, port int) error {
	serv := grpc.NewServer()
	v1.RegisterRbacInternalServiceServer(serv, s)
	return listenAndServe(serv, port)
}

// listenAndServe is a helper function for starting a gRPC server.
func listenAndServe(grpcServer *grpc.Server, port int) error {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	if err := grpcServer.Serve(l); err != nil {
		return fmt.Errorf("failed to start gRPC server: %v", err)
	}
	return nil
}
