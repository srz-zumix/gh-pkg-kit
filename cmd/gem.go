package cmd

import (
	"github.com/spf13/cobra"
	gemCmd "github.com/srz-zumix/gh-pkg-kit/cmd/gem"
)

func newGemCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gem",
		Short: "RubyGems package operations",
		Long:  `RubyGems package operations for GitHub Packages.`,
	}
	cmd.AddCommand(gemCmd.NewDownloadCmd())
	return cmd
}

func init() {
	rootCmd.AddCommand(newGemCmd())
}
