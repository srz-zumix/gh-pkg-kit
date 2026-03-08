package cmd

import (
	"github.com/spf13/cobra"
	userCmd "github.com/srz-zumix/gh-pkg-kit/cmd/user"
)

func newUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "User package operations",
		Long:  `User package operations for GitHub Packages.`,
	}
	cmd.AddCommand(userCmd.NewListCmd())
	cmd.AddCommand(userCmd.NewGetCmd())
	cmd.AddCommand(userCmd.NewDeleteCmd())
	cmd.AddCommand(userCmd.NewRestoreCmd())
	cmd.AddCommand(userCmd.NewVersionsCmd())
	return cmd
}

func init() {
	rootCmd.AddCommand(newUserCmd())
}
