package user

import (
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

// NewListCmd creates a command to list packages for a user
func NewListCmd() *cobra.Command {
	var (
		owner       string
		packageType string
		visibility  string
		exporter    cmdutil.Exporter
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List packages for a user",
		Long:  `Lists all packages in a user's namespace for which the requesting user has access.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			g, err := gh.NewGitHubClient()
			if err != nil {
				return err
			}
			packages, err := gh.ListUserPackages(ctx, g, owner, packageType, visibility)
			if err != nil {
				return err
			}
			r := render.NewRenderer(exporter)
			r.RenderPackages(packages, nil)
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVarP(&owner, "owner", "o", "", "Owner name (defaults to authenticated user)")
	cmdutil.StringEnumFlag(cmd, &packageType, "type", "T", "", gh.PackageTypes, "Package type")
	cmdutil.StringEnumFlag(cmd, &visibility, "visibility", "V", "", gh.PackageVisibilityList, "Package visibility")
	cmdutil.AddFormatFlags(cmd, &exporter)
	return cmd
}
