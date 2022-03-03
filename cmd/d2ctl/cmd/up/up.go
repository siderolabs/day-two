// Package up contains the commands for upping a stack.
package up

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/talos-systems/day-two/cmd/d2ctl/cmd"
	"github.com/talos-systems/day-two/pkg/pulumi"
)

var configPath string

func init() {
	cmd.RootCmd.AddCommand(upCmd)
	upCmd.PersistentFlags().StringVar(&configPath, "config-path", "", "Path to config.yaml")

	err := upCmd.MarkPersistentFlagRequired("config-path")
	if err != nil {
		fmt.Printf("failed to mark config-path as required: %q", err)
		os.Exit(1)
	}
}

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Setup helm charts",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		return pulumi.Up(ctx, configPath)
	},
}