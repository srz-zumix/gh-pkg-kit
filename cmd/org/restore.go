package org

import (
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

// NewRestoreCmd creates a command to restore a package for an organization
func NewRestoreCmd() *cobra.Command {
	var (
		owner       string
		packageType string
	)

	cmd := &cobra.Command{
		Use:   "restore <package-name>",
		Short: "Restore a package for an organization",
		Long: `Restores an entire package in an organization.
The package must have been deleted within the last 30 days, and the same package namespace and version must still be available.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packageName := args[0]
			ctx := cmd.Context()
			repo, err := parser.Repository(parser.RepositoryOwnerWithHost(owner))
			if err != nil {
				return err
			}
			g, err := gh.NewGitHubClientWithRepo(repo)
			if err != nil {
				return err
			}
			err = gh.RestoreOrgPackage(ctx, g, repo, packageType, packageName)
			if err != nil {
				return err
			}
			logger.Info("Package restored", "package", packageName, "org", repo.Owner)
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVarP(&owner, "owner", "o", "", "Owner ([HOST/]OWNER, defaults to current repository owner)")
	cmdutil.StringEnumFlag(cmd, &packageType, "type", "T", "", gh.PackageTypes, "Package type")
	_ = cmd.MarkFlagRequired("type")
	return cmd
}
