package credentialprovider

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-pkg-kit/pkg/nuget"
)

func NewInstallCmd() *cobra.Command {
	return newInstallCmd(nuget.InstallCredentialProvider)
}

func newInstallCmd(install func(context.Context, string, bool) (string, error)) *cobra.Command {
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
			if dir == "" {
				var err error
				dir, err = nuget.DefaultProviderDir()
				if err != nil {
					return fmt.Errorf("failed to resolve NuGet plugin directory: %w", err)
				}
			}
			path, err := install(cmd.Context(), dir, force)
			if err != nil {
				return fmt.Errorf("failed to install NuGet credential provider: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Installed %s\nRequires .NET 8 or later and gh on PATH at runtime\n", path)
			for _, name := range []string{"NUGET_PLUGIN_PATHS", "NUGET_NETCORE_PLUGIN_PATHS"} {
				if os.Getenv(name) != "" {
					fmt.Fprintf(cmd.ErrOrStderr(), "%s overrides convention-based plugin discovery; unset it or include the installed DLL path\n", name)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "Netcore plugin root (default: ~/.nuget/plugins/netcore)")
	cmd.Flags().BoolVar(&force, "force", false, "Replace an existing gh-pkg-kit-managed provider installation")
	return cmd
}
