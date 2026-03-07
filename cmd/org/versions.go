package org

import (
	"github.com/spf13/cobra"
	versionsCmd "github.com/srz-zumix/gh-pkg-kit/cmd/org/versions"
)

// NewVersionsCmd creates a parent command for package version operations for an organization
func NewVersionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "versions",
		Short: "Package version operations for an organization",
		Long:  `Manage package versions in an organization.`,
	}
	cmd.AddCommand(versionsCmd.NewListCmd())
	cmd.AddCommand(versionsCmd.NewGetCmd())
	cmd.AddCommand(versionsCmd.NewDeleteCmd())
	cmd.AddCommand(versionsCmd.NewRestoreCmd())
	return cmd
}

