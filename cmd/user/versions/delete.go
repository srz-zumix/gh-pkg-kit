package versions

import (
	"fmt"
	"strconv"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
)

// NewDeleteCmd creates a command to delete a package version for a user
func NewDeleteCmd() *cobra.Command {
	var (
		owner       string
		packageType string
	)

	cmd := &cobra.Command{
		Use:   "delete <package-name> <version-id>",
		Short: "Delete a package version for a user",
		Long:  `Deletes a specific package version for a user. If the package is public and the package version has more than 5,000 downloads, you cannot delete the package version.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			packageName := args[0]
			versionID, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid version ID '%s': %w", args[1], err)
			}
			ctx := cmd.Context()
			g, err := gh.NewGitHubClient()
			if err != nil {
				return err
			}
			err = gh.DeleteUserPackageVersion(ctx, g, owner, packageType, packageName, versionID)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Version %d of package '%s' deleted for user '%s'\n", versionID, packageName, owner)
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVarP(&owner, "owner", "o", "", "Owner name (defaults to authenticated user)")
	cmdutil.StringEnumFlag(cmd, &packageType, "type", "T", "", gh.PackageTypes, "Package type")
	_ = cmd.MarkFlagRequired("type")
	return cmd
}
