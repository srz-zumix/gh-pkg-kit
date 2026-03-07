package versions

import (
	"fmt"
	"strconv"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
)

// NewRestoreCmd creates a command to restore a package version for a user
func NewRestoreCmd() *cobra.Command {
	var (
		owner       string
		packageType string
	)

	cmd := &cobra.Command{
		Use:   "restore <package-name> <version-id>",
		Short: "Restore a package version for a user",
		Long: `Restores a specific package version for a user.
The package must have been deleted within the last 30 days, and the same package namespace and version must still be available.`,
		Args: cobra.ExactArgs(2),
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
			err = gh.RestoreUserPackageVersion(ctx, g, owner, packageType, packageName, versionID)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Version %d of package '%s' restored for user '%s'\n", versionID, packageName, owner)
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVarP(&owner, "owner", "o", "", "Owner name (defaults to authenticated user)")
	cmdutil.StringEnumFlag(cmd, &packageType, "type", "T", "", gh.PackageTypes, "Package type")
	_ = cmd.MarkFlagRequired("type")
	return cmd
}
