package org

import (
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

// NewListCmd creates a command to list packages for an organization
func NewListCmd() *cobra.Command {
	var (
		owner       string
		packageType string
		visibility  string
		exporter    cmdutil.Exporter
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List packages for an organization",
		Long:  `Lists packages in an organization readable by the user.`,
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
			packages, err := gh.ListOrgPackages(ctx, g, repo, packageType, visibility)
			if err != nil {
				return err
			}
			r := render.NewRenderer(exporter)
			r.RenderPackages(packages, nil)
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVarP(&owner, "owner", "o", "", "[HOST/]OWNER (defaults to current repository owner)")
	cmdutil.StringEnumFlag(cmd, &packageType, "type", "T", "", gh.PackageTypes, "Package type")
	cmdutil.StringEnumFlag(cmd, &visibility, "visibility", "V", "", gh.PackageVisibilityList, "Package visibility")
	cmdutil.AddFormatFlags(cmd, &exporter)
	return cmd
}
