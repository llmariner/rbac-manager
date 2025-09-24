package server

import (
	"context"
	"fmt"
	"net"

	v1 "github.com/llmariner/rbac-manager/api/v1"
	"github.com/llmariner/rbac-manager/server/internal/cache"
	"github.com/llmariner/rbac-manager/server/internal/token"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

type cacheGetter interface {
	GetAPIKeyBySecret(secret string) (*cache.K, bool)

	GetClusterByRegistrationKey(key string) (*cache.C, bool)
	GetClustersByTenantID(tenantID string) []cache.C

	GetOrganizationByID(organizationID string) (*cache.O, bool)
	GetOrganizationsByUserID(userID string) []cache.OU

	GetProjectsByOrganizationID(organizationID string) []cache.P
	GetProjectByID(projectID string) (*cache.P, bool)
	GetProjectsByUserID(userID string) []cache.PU

	GetUserByID(id string) (*cache.U, bool)
}

// TokenIntrospector inspects the token.
type TokenIntrospector interface {
	TokenIntrospect(token string) (*token.Introspection, error)
}

// New returns a new Server.
func New(ti TokenIntrospector, cache cacheGetter, roleScopes map[string][]string) *Server {
	return &Server{
		tokenIntrospector: ti,

		cache: cache,

		roleScopesMapper: roleScopes,
	}
}

// Server implementes the gRPC interface.
type Server struct {
	v1.UnimplementedRbacInternalServiceServer

	srv *grpc.Server

	tokenIntrospector TokenIntrospector

	cache cacheGetter

	roleScopesMapper map[string][]string
}

// Run starts the gRPC server.
func (s *Server) Run(ctx context.Context, port int) error {
	serv := grpc.NewServer()
	v1.RegisterRbacInternalServiceServer(serv, s)
	reflection.Register(serv)

	healthCheck := health.NewServer()
	healthCheck.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(serv, healthCheck)

	s.srv = serv

	return listenAndServe(serv, port)
}

// GracefulStop stops the gRPC server gracefully.
func (s *Server) GracefulStop() {
	s.srv.GracefulStop()
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
