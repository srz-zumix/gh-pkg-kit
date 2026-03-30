package cmd

import (
	"github.com/spf13/cobra"
	dockerCmd "github.com/srz-zumix/gh-pkg-kit/cmd/docker"
)

func newDockerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docker",
		Short: "Docker package operations",
		Long:  `Docker package operations for GitHub Packages.`,
	}
	cmd.AddCommand(dockerCmd.NewPullCmd())
	return cmd
}

func init() {
	rootCmd.AddCommand(newDockerCmd())
}
