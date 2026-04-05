package versions

import (
	"fmt"
	"strconv"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

// NewGetCmd creates a command to get a package version for an organization
func NewGetCmd() *cobra.Command {
	var (
		owner       string
		packageType string
		exporter    cmdutil.Exporter
	)

	cmd := &cobra.Command{
		Use:   "get <package-name> <version-id>",
		Short: "Get a package version for an organization",
		Long:  `Gets a specific package version in an organization.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			packageName := args[0]
			versionID, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid version ID '%s': %w", args[1], err)
			}
			ctx := cmd.Context()
			repo, err := parser.Repository(parser.RepositoryOwnerWithHost(owner))
			if err != nil {
				return err
			}
			g, err := gh.NewGitHubClientWithRepo(repo)
			if err != nil {
				return err
			}
			version, err := gh.GetOrgPackageVersion(ctx, g, repo, packageType, packageName, versionID)
			if err != nil {
				return err
			}
			r := render.NewRenderer(exporter)
			r.RenderPackageVersion(version)
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVarP(&owner, "owner", "o", "", "[HOST/]OWNER (defaults to current repository owner)")
	cmdutil.StringEnumFlag(cmd, &packageType, "type", "T", "", gh.PackageTypes, "Package type")
	_ = cmd.MarkFlagRequired("type")
	cmdutil.AddFormatFlags(cmd, &exporter)
	return cmd
}
