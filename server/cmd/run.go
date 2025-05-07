package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-logr/stdr"
	cv1 "github.com/llmariner/cluster-manager/api/v1"
	"github.com/llmariner/rbac-manager/server/internal/cache"
	"github.com/llmariner/rbac-manager/server/internal/config"
	"github.com/llmariner/rbac-manager/server/internal/monitoring"
	"github.com/llmariner/rbac-manager/server/internal/server"
	"github.com/llmariner/rbac-manager/server/internal/token"
	uv1 "github.com/llmariner/user-manager/api/v1"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	flagConfig               = "config"
	monitoringRunnerInterval = 10 * time.Second
)

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
	logger := stdr.New(log.Default())
	log := logger.WithName("boot")

	log.Info("Starting internal-grpc server...", "port", c.InternalGRPCPort)

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

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	cstore := cache.NewStore(uClient, cClient)
	errCh := make(chan error)
	go func() {
		errCh <- cstore.Sync(ctx, c.CacheConfig.SyncInterval)
	}()

	// We could wait for the cache to be populated before starting the server, but
	// we intentionally avoid that here to avoid hard dependency to user-manager-server.
	// TODO(kenji): Consider revisit this.

	ta, err := token.NewValidator(ctx, c.JWKSURL, token.ValidatorOpts{Refresh: 1 * time.Hour})
	if err != nil {
		return err
	}
	srv := server.New(ta, cstore, c.RoleScopesMap)
	go func() {
		errCh <- srv.Run(ctx, c.InternalGRPCPort)
	}()

	m := monitoring.NewMetricsMonitor(cstore, logger)
	go func() {
		errCh <- m.Run(ctx, monitoringRunnerInterval)
	}()

	defer m.UnregisterAllCollectors()

	go func() {
		log := logger.WithName("metrics")
		log.Info("Starting metrics server...", "port", c.MonitoringPort)
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		errCh <- http.ListenAndServe(fmt.Sprintf(":%d", c.MonitoringPort), mux)
		log.Info("Stopped metrics server")
	}()

	select {
	case err := <-errCh:
		return err
	case sig := <-sigCh:
		log.Info("Got signal, waiting for graceful shutdown", "signal", sig, "delay", c.GracefulShutdownDelay)
		time.Sleep(c.GracefulShutdownDelay)

		log.Info("Starting graceful shutdown.")
		srv.GracefulStop()

		return nil
	}
}

func init() {
	runCmd.Flags().StringP(flagConfig, "c", "", "Configuration file path")
	_ = runCmd.MarkFlagRequired(flagConfig)
}
