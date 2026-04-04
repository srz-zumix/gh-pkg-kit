package nuget

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2/pkg/auth"
	"github.com/spf13/cobra"
	nugetConfig "github.com/srz-zumix/gh-pkg-kit/pkg/nuget"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
)

// NewToolRestoreCmd creates a command that runs 'dotnet tool restore' with
// GitHub Packages credentials injected from the gh auth token.
func NewToolRestoreCmd() *cobra.Command {
	var (
		configFile string
		dryRun     bool
		workDir    string
		overwrite  bool
	)

	cmd := &cobra.Command{
		Use:   "tool-restore [-- dotnet-tool-restore-args...]",
		Short: "Run dotnet tool restore with gh auth credentials injected into NuGet.Config",
		Long: `Runs 'dotnet tool restore' after injecting GitHub Packages credentials from the
gh auth token into a NuGet.Config file.

By default, a temporary copy of the NuGet.Config is created with credentials injected.
With --overwrite, the credentials are written directly into the existing NuGet.Config.

Extra arguments after -- are passed through to 'dotnet tool restore'.`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Create temp directory for the temporary NuGet.Config (not needed when --overwrite is set)
			var tmpDir string
			if !overwrite {
				if workDir != "" {
					if err := os.MkdirAll(workDir, 0700); err != nil {
						return fmt.Errorf("failed to create work directory: %w", err)
					}
					tmpDir = workDir
					logger.Info("Using work directory", "dir", tmpDir)
				} else {
					var err error
					tmpDir, err = os.MkdirTemp("", "gh-pkg-kit-nuget-*")
					if err != nil {
						return fmt.Errorf("failed to create temp directory: %w", err)
					}
					defer func() { _ = os.RemoveAll(tmpDir) }()
				}
			}

			dotnetArgs := []string{"tool", "restore"}

			nugetConfigPath := nugetConfig.ResolveConfigPath(configFile)
			if configFile != "" && nugetConfigPath == "" {
				return fmt.Errorf("config file not found: %s", configFile)
			}
			if nugetConfigPath != "" {
				host, _ := auth.DefaultHost()
				token, _ := auth.TokenForHost(host)
				if token != "" {
					var dstConfig string
					if overwrite {
						dstConfig = nugetConfigPath
					} else {
						dstConfig = filepath.Join(tmpDir, "NuGet.Config")
					}
					if err := nugetConfig.WriteConfigWithCredentials(nugetConfigPath, dstConfig, token); err != nil {
						logger.Warn("Failed to inject credentials into NuGet.Config, using original", "error", err)
					} else {
						logger.Info("Injected gh auth credentials into NuGet.Config", "config", dstConfig)
						dotnetArgs = append(dotnetArgs, "--configfile", dstConfig)
					}
				}
			}

			dotnetArgs = append(dotnetArgs, args...)

			if dryRun {
				logger.Info("Dry run: would run: dotnet " + strings.Join(dotnetArgs, " "))
				return nil
			}

			logger.Info("Running: dotnet " + strings.Join(dotnetArgs, " "))
			dotnetCmd := exec.CommandContext(ctx, "dotnet", dotnetArgs...)
			dotnetCmd.Stdout = os.Stdout
			dotnetCmd.Stderr = os.Stderr
			dotnetCmd.Stdin = os.Stdin

			if err := dotnetCmd.Run(); err != nil {
				return fmt.Errorf("dotnet tool restore failed: %w", err)
			}

			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&configFile, "configfile", "", "Path to NuGet.Config (auto-detected if not specified)")
	f.BoolVarP(&dryRun, "dry-run", "n", false, "Show what would be done without running dotnet")
	f.StringVar(&workDir, "work-dir", "", "Working directory for temporary files (default: a temporary directory under the system temp dir, deleted on exit)")
	f.BoolVar(&overwrite, "overwrite", false, "Overwrite the existing NuGet.Config with injected credentials instead of using a temporary copy")

	return cmd
}
