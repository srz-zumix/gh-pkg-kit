package credentialprovider

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-pkg-kit/pkg/nuget"
)

func NewUninstallCmd() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall the .NET NuGet credential provider",
		Long: "Removes CredentialProvider.GhPkgKit from ~/.nuget/plugins/netcore or the --dir plugin root.\n" +
			"Directories without the gh-pkg-kit generation marker and symbolic links are not removed. A missing provider is ignored.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				var err error
				dir, err = nuget.DefaultProviderDir()
				if err != nil {
					return fmt.Errorf("failed to resolve NuGet plugin directory: %w", err)
				}
			}
			if err := nuget.RemoveCredentialProvider(dir); err != nil {
				return fmt.Errorf("failed to uninstall NuGet credential provider: %w", err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "Netcore plugin root (default: ~/.nuget/plugins/netcore)")
	return cmd
}
