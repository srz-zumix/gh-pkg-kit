package versions

import (
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

// NewListCmd creates a command to list package versions for a user
func NewListCmd() *cobra.Command {
	var (
		owner       string
		packageType string
		state       string
		exporter    cmdutil.Exporter
	)

	cmd := &cobra.Command{
		Use:   "list <package-name>",
		Short: "List package versions for a user",
		Long:  `Lists package versions for a package owned by a user.`,
		Args:  cobra.ExactArgs(1),
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
			versions, err := gh.ListUserPackageVersions(ctx, g, repo.Owner, packageType, packageName, state)
			if err != nil {
				return err
			}
			r := render.NewRenderer(exporter)
			r.RenderPackageVersions(versions, nil)
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVarP(&owner, "owner", "o", "", "Owner ([HOST/]OWNER, defaults to authenticated user)")
	cmdutil.StringEnumFlag(cmd, &packageType, "type", "T", "", gh.PackageTypes, "Package type")
	_ = cmd.MarkFlagRequired("type")
	cmdutil.StringEnumFlag(cmd, &state, "state", "s", "", []string{"active", "deleted"}, "Package state")
	cmdutil.AddFormatFlags(cmd, &exporter)
	return cmd
}
