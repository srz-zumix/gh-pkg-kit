package user

import (
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

// NewListCmd creates a command to list packages for a user
func NewListCmd() *cobra.Command {
	var (
		owner       string
		packageType string
		visibility  string
		fields      []string
		exporter    cmdutil.Exporter
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List packages for a user",
		Long:  `Lists all packages in a user's namespace for which the requesting user has access.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			repo, err := parser.Repository(parser.RepositoryOwnerWithHost(owner))
			if err != nil {
				return err
			}
			g, err := gh.NewGitHubClientWithRepo(repo)
			if err != nil {
				return err
			}
			packages, err := gh.ListUserPackages(ctx, g, repo.Owner, packageType, visibility)
			if err != nil {
				return err
			}
			r := render.NewRenderer(exporter)
			r.RenderPackages(packages, fields)
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVarP(&owner, "owner", "o", "", "Owner ([HOST/]OWNER, defaults to authenticated user)")
	cmdutil.StringEnumFlag(cmd, &packageType, "type", "T", "", gh.PackageTypes, "Package type")
	cmdutil.StringEnumFlag(cmd, &visibility, "visibility", "V", "", gh.PackageVisibilityList, "Package visibility")
	cmdutil.StringSliceEnumFlag(cmd, &fields, "field", "", nil, render.PackageFields, "Fields to display")
	cmdutil.AddFormatFlags(cmd, &exporter)
	return cmd
}
