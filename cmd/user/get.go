package user

import (
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

// NewGetCmd creates a command to get a specific package for a user
func NewGetCmd() *cobra.Command {
	var (
		owner       string
		packageType string
		exporter    cmdutil.Exporter
	)

	cmd := &cobra.Command{
		Use:   "get <package-name>",
		Short: "Get a package for a user",
		Long:  `Gets a specific package metadata for a package owned by a user.`,
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
			pkg, err := gh.GetUserPackage(ctx, g, repo.Owner, packageType, packageName)
			if err != nil {
				return err
			}
			r := render.NewRenderer(exporter)
			r.RenderPackage(pkg)
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVarP(&owner, "owner", "o", "", "Owner ([HOST/]OWNER, defaults to authenticated user)")
	cmdutil.StringEnumFlag(cmd, &packageType, "type", "T", "", gh.PackageTypes, "Package type")
	_ = cmd.MarkFlagRequired("type")
	cmdutil.AddFormatFlags(cmd, &exporter)
	return cmd
}
