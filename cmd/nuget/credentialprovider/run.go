package credentialprovider

import (
	"fmt"
	"slices"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-pkg-kit/pkg/nuget"
)

func NewRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "run [-Plugin]",
		Short:              "Serve NuGet authentication requests using gh auth tokens",
		Long:               "Runs the installed .NET credential provider using the default netcore plugin directory.\nWith -Plugin, forwards stdin/stdout to the NuGet.Protocol implementation. Without -Plugin, prints an installation hint.\nNuGet normally launches the installed DLL directly. Requires .NET 8 or later and gh on PATH.",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if slices.Contains(args, "--help") || slices.Contains(args, "-h") {
				return cmd.Help()
			}
			if len(args) == 0 {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "Run 'gh pkg-kit nuget credential-provider install' to install the NuGet plugin shim")
				return err
			}
			if len(args) != 1 || args[0] != "-Plugin" {
				return fmt.Errorf("run accepts only the NuGet -Plugin argument")
			}
			cmd.SilenceUsage = true
			if err := nuget.RunInstalledCredentialProvider(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr()); err != nil {
				return fmt.Errorf("failed to serve NuGet credential requests: %w", err)
			}
			return nil
		},
	}
}
