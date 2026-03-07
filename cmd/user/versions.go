package user

import (
	"github.com/spf13/cobra"
	versionsCmd "github.com/srz-zumix/gh-pkg-kit/cmd/user/versions"
)

// NewVersionsCmd creates a parent command for package version operations for a user
func NewVersionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "versions",
		Short: "Package version operations for a user",
		Long:  `Manage package versions for a user.`,
	}
	cmd.AddCommand(versionsCmd.NewListCmd())
	cmd.AddCommand(versionsCmd.NewGetCmd())
	cmd.AddCommand(versionsCmd.NewDeleteCmd())
	cmd.AddCommand(versionsCmd.NewRestoreCmd())
	return cmd
}

