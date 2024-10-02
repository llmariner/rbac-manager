package main

import (
	"context"
	"log"

	cv1 "github.com/llmariner/cluster-manager/api/v1"
	"github.com/llmariner/rbac-manager/server/internal/cache"
	"github.com/llmariner/rbac-manager/server/internal/config"
	"github.com/llmariner/rbac-manager/server/internal/server"
	uv1 "github.com/llmariner/user-manager/api/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const flagConfig = "config"

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "run",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := cmd.Flags().GetString(flagConfig)
		if err != nil {
			return err
		}

		c, err := config.Parse(path)
		if err != nil {
			return err
		}

		if err := c.Validate(); err != nil {
			return err
		}

		if err := run(cmd.Context(), &c); err != nil {
			return err
		}
		return nil
	},
}

func run(ctx context.Context, c *config.Config) error {
	log.Printf("Starting internal-grpc server on port %d", c.InternalGRPCPort)

	conn, err := grpc.NewClient(
		c.CacheConfig.UserManagerServerInternalAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return err
	}
	uClient := uv1.NewUsersInternalServiceClient(conn)

	conn, err = grpc.NewClient(
		c.CacheConfig.ClusterManagerServerInternalAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return err
	}
	cClient := cv1.NewClustersInternalServiceClient(conn)

	cstore := cache.NewStore(
		uClient,
		cClient,
		&c.Debug,
	)
	errCh := make(chan error)
	go func() {
		errCh <- cstore.Sync(ctx, c.CacheConfig.SyncInterval)
	}()

	// We could wait for the cache to be populated before starting the server, but
	// we intentionally avoid that here to avoid hard dependency to user-manager-server.
	// TODO(kenji): Consider revisit this.

	go func() {
		srv := server.New(c.DexServerAddr, cstore, c.RoleScopesMap)
		errCh <- srv.Run(ctx, c.InternalGRPCPort)
	}()

	return <-errCh
}

func init() {
	runCmd.Flags().StringP(flagConfig, "c", "", "Configuration file path")
	_ = runCmd.MarkFlagRequired(flagConfig)
}
