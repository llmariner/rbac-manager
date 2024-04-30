package main

import (
	"context"
	"log"

	"github.com/llm-operator/rbac-manager/server/internal/config"
	"github.com/llm-operator/rbac-manager/server/internal/server"
	"github.com/spf13/cobra"
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
	srv := server.New(c.IssuerURL, &c.Debug)
	return srv.Run(ctx, c.InternalGRPCPort)
}

func init() {
	runCmd.Flags().StringP(flagConfig, "c", "", "Configuration file path")
	_ = runCmd.MarkFlagRequired(flagConfig)
}
