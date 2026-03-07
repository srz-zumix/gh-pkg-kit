package user

import (
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
)

// NewRestoreCmd creates a command to restore a package for a user
func NewRestoreCmd() *cobra.Command {
	var (
		owner       string
		packageType string
	)

	cmd := &cobra.Command{
		Use:   "restore <package-name>",
		Short: "Restore a package for a user",
		Long: `Restores an entire package for a user.
The package must have been deleted within the last 30 days, and the same package namespace and version must still be available.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packageName := args[0]
			ctx := cmd.Context()
			g, err := gh.NewGitHubClient()
			if err != nil {
				return err
			}
			err = gh.RestoreUserPackage(ctx, g, owner, packageType, packageName)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Package '%s' restored for user '%s'\n", packageName, owner)
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVarP(&owner, "owner", "o", "", "Owner name (defaults to authenticated user)")
	cmdutil.StringEnumFlag(cmd, &packageType, "type", "T", "", gh.PackageTypes, "Package type")
	_ = cmd.MarkFlagRequired("type")
	return cmd
}
