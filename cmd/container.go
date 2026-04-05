package cmd

import (
	"github.com/spf13/cobra"
	containerCmd "github.com/srz-zumix/gh-pkg-kit/cmd/container"
)

func newContainerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "container",
		Short: "Container package operations",
		Long:  `Container package operations for GitHub Packages.`,
	}
	cmd.AddCommand(containerCmd.NewPullCmd())
	return cmd
}

func init() {
	rootCmd.AddCommand(newContainerCmd())
}
