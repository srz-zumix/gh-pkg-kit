package versions

import (
	"fmt"
	"strconv"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

// NewDeleteCmd creates a command to delete a package version for an organization
func NewDeleteCmd() *cobra.Command {
	var (
		owner       string
		packageType string
	)

	cmd := &cobra.Command{
		Use:   "delete <package-name> <version-id>",
		Short: "Delete a package version for an organization",
		Long:  `Deletes a specific package version in an organization. If the package is public and the package version has more than 5,000 downloads, you cannot delete the package version.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			packageName := args[0]
			versionID, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid version ID '%s': %w", args[1], err)
			}
			ctx := cmd.Context()
			repo, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return err
			}
			g, err := gh.NewGitHubClientWithRepo(repo)
			if err != nil {
				return err
			}
			if owner == "" {
				owner = repo.Owner
			}
			err = gh.DeleteOrgPackageVersion(ctx, g, repo, packageType, packageName, versionID)
			if err != nil {
				return err
			}
			logger.Info("Version deleted", "version", versionID, "package", packageName, "org", owner)
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVarP(&owner, "owner", "o", "", "Owner name (defaults to current repository owner)")
	cmdutil.StringEnumFlag(cmd, &packageType, "type", "T", "", gh.PackageTypes, "Package type")
	_ = cmd.MarkFlagRequired("type")
	return cmd
}
