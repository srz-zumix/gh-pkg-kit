package cmd

import (
	"github.com/spf13/cobra"
	nugetCmd "github.com/srz-zumix/gh-pkg-kit/cmd/nuget"
)

func newNuGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nuget",
		Short: "NuGet package operations",
		Long:  `NuGet package operations for GitHub Packages.`,
	}
	cmd.AddCommand(nugetCmd.NewDownloadCmd())
	return cmd
}

func init() {
	rootCmd.AddCommand(newNuGetCmd())
}
