package credentialprovider

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-pkg-kit/pkg/nuget"
)

func NewInstallCmd() *cobra.Command {
	var dir string
	var force bool
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install the .NET NuGet credential provider",
		Long: "Publishes the bundled .NET credential provider into ~/.nuget/plugins/netcore/CredentialProvider.GhPkgKit.\n" +
			"Requires .NET SDK 8 or later and access to nuget.org during installation; runtime use requires .NET 8 or later and gh.\n" +
			"--dir overrides the netcore plugin root. --force replaces only an existing gh-pkg-kit-managed installation.\n" +
			"NuGet discovers the default location automatically unless plugin path environment variables override discovery.\n" +
			"Authenticate using gh auth login before restoring packages; no token is stored in the provider directory.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := nuget.InstallCredentialProvider(cmd.Context(), dir, force)
			if err != nil {
				return fmt.Errorf("failed to install NuGet credential provider: %w", err)
			}
			nuget.ReportCredentialProviderInstallation(path)
			return nil
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "Netcore plugin root (default: ~/.nuget/plugins/netcore)")
	cmd.Flags().BoolVar(&force, "force", false, "Replace an existing gh-pkg-kit-managed provider installation")
	return cmd
}
