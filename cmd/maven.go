package cmd

import (
	"github.com/spf13/cobra"
	mavenCmd "github.com/srz-zumix/gh-pkg-kit/cmd/maven"
)

func newMavenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "maven",
		Short: "Maven package operations",
		Long:  `Maven package operations for GitHub Packages.`,
	}
	cmd.AddCommand(mavenCmd.NewDownloadCmd())
	return cmd
}

func init() {
	rootCmd.AddCommand(newMavenCmd())
}
