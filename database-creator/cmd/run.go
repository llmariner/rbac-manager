package main

import (
	"context"

	"github.com/llm-operator/common/pkg/db"
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

		c, err := parse(path)
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

func run(ctx context.Context, c *Config) error {
	// To create a new database if requested/needed.
	_, err := db.OpenDB(c.Storage.Config)
	return err

}

func init() {
	runCmd.Flags().StringP(flagConfig, "c", "", "Configuration file path")
	_ = runCmd.MarkFlagRequired(flagConfig)
}
