package nuget

import (
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-pkg-kit/cmd/nuget/credentialprovider"
)

func NewCredentialProviderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "credential-provider",
		Short: "Manage a NuGet credential provider backed by gh auth",
	}
	cmd.AddCommand(credentialprovider.NewRunCmd())
	cmd.AddCommand(credentialprovider.NewInstallCmd())
	cmd.AddCommand(credentialprovider.NewUninstallCmd())
	return cmd
}
