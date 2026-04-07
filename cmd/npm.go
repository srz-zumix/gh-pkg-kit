package cmd

import (
	"github.com/spf13/cobra"
	npmCmd "github.com/srz-zumix/gh-pkg-kit/cmd/npm"
)

func newNPMCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "npm",
		Short: "npm package operations",
		Long:  `npm package operations for GitHub Packages.`,
	}
	cmd.AddCommand(npmCmd.NewDownloadCmd())
	return cmd
}

func init() {
	rootCmd.AddCommand(newNPMCmd())
}
