package cmd

import (
	"github.com/spf13/cobra"
	orgCmd "github.com/srz-zumix/gh-pkg-kit/cmd/org"
)

func newOrgCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "org",
		Short: "Organization package operations",
		Long:  `Organization package operations for GitHub Packages.`,
	}
	cmd.AddCommand(orgCmd.NewListCmd())
	cmd.AddCommand(orgCmd.NewGetCmd())
	cmd.AddCommand(orgCmd.NewDeleteCmd())
	cmd.AddCommand(orgCmd.NewRestoreCmd())
	cmd.AddCommand(orgCmd.NewVersionsCmd())
	return cmd
}

func init() {
	rootCmd.AddCommand(newOrgCmd())
}
