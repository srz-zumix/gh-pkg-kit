package user

import (
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
)

// NewDeleteCmd creates a command to delete a package for a user
func NewDeleteCmd() *cobra.Command {
	var (
		owner       string
		packageType string
	)

	cmd := &cobra.Command{
		Use:   "delete <package-name>",
		Short: "Delete a package for a user",
		Long:  `Deletes an entire package for a user. You cannot delete a public package if any version of the package has more than 5,000 downloads.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packageName := args[0]
			ctx := cmd.Context()
			g, err := gh.NewGitHubClient()
			if err != nil {
				return err
			}
			err = gh.DeleteUserPackage(ctx, g, owner, packageType, packageName)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Package '%s' deleted for user '%s'\n", packageName, owner)
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVarP(&owner, "owner", "o", "", "Owner name (defaults to authenticated user)")
	cmdutil.StringEnumFlag(cmd, &packageType, "type", "T", "", gh.PackageTypes, "Package type")
	_ = cmd.MarkFlagRequired("type")
	return cmd
}
