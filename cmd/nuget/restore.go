package nuget

import (
	"fmt"

	"github.com/spf13/cobra"
	nugetConfig "github.com/srz-zumix/gh-pkg-kit/pkg/nuget"
)

func NewRestoreCmd() *cobra.Command {
	var (
		configFile string
		workDir    string
	)

	cmd := &cobra.Command{
		Use:   "restore [project-or-solution] [-- dotnet-restore-args...]",
		Short: "Run dotnet restore with a temporary gh auth credential provider",
		Long: `Runs 'dotnet restore' with a temporary GitHub Packages credential provider
that retrieves credentials from gh auth. No permanent provider installation is required,
and NuGet.Config is not modified.

Requires .NET SDK 8+ and access to nuget.org to build the provider. The provider is
removed on exit, including when --work-dir is specified. Existing explicit plugin
paths are preserved for the restore subprocess.

The optional project or solution and extra arguments after -- are passed through
to 'dotnet restore'. If no project or solution is specified, dotnet uses the current directory.`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := nugetConfig.RunRestore(cmd.Context(), configFile, workDir, args); err != nil {
				return fmt.Errorf("failed to restore NuGet packages: %w", err)
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&configFile, "configfile", "", "Path to NuGet.Config (dotnet's default discovery if not specified)")
	f.StringVar(&workDir, "work-dir", "", "Parent directory for the temporary credential provider (default: system temp directory; provider deleted on exit)")
	return cmd
}
