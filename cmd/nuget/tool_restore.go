package nuget

import (
	"fmt"

	"github.com/spf13/cobra"
	nugetConfig "github.com/srz-zumix/gh-pkg-kit/pkg/nuget"
)

// NewToolRestoreCmd creates a command that runs 'dotnet tool restore' with
// a temporary GitHub Packages credential provider backed by gh auth.
func NewToolRestoreCmd() *cobra.Command {
	var (
		configFile string
		workDir    string
	)

	cmd := &cobra.Command{
		Use:   "tool-restore [-- dotnet-tool-restore-args...]",
		Short: "Run dotnet tool restore with a temporary gh auth credential provider",
		Long: `Runs 'dotnet tool restore' with a temporary GitHub Packages credential provider
that retrieves credentials from gh auth. No permanent provider installation is required,
and NuGet.Config is not modified.

Requires .NET SDK 8+ and access to nuget.org to build the provider. The provider is
removed on exit, including when --work-dir is specified. Existing explicit plugin
paths are preserved for the restore subprocess.

Extra arguments after -- are passed through to 'dotnet tool restore'.`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := nugetConfig.RunToolRestore(cmd.Context(), configFile, workDir, args); err != nil {
				return fmt.Errorf("failed to restore NuGet tools: %w", err)
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&configFile, "configfile", "", "Path to NuGet.Config (dotnet's default discovery if not specified)")
	f.StringVar(&workDir, "work-dir", "", "Parent directory for the temporary credential provider (default: system temp directory; provider deleted on exit)")

	return cmd
}
