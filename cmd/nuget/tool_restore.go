package nuget

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2/pkg/auth"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
)

// nugetConfiguration represents the XML structure of a NuGet.Config file
type nugetConfiguration struct {
	XMLName                  xml.Name                `xml:"configuration"`
	PackageSources           *nugetPackageSources    `xml:"packageSources"`
	PackageSourceCredentials *nugetSourceCredentials `xml:"packageSourceCredentials"`
	Other                    []nugetRawSection       `xml:",any"`
}

type nugetPackageSources struct {
	Items []nugetAddItem `xml:",any"`
}

type nugetAddItem struct {
	XMLName xml.Name
	Key     string `xml:"key,attr,omitempty"`
	Value   string `xml:"value,attr,omitempty"`
}

type nugetSourceCredentials struct {
	Sources []nugetCredentialSource `xml:",any"`
}

type nugetCredentialSource struct {
	XMLName xml.Name
	Items   []nugetAddItem `xml:"add"`
}

type nugetRawSection struct {
	XMLName xml.Name
	Content []byte `xml:",innerxml"`
}

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

			nugetConfigPath := resolveNuGetConfigPath(configFile)
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
					if err := writeNuGetConfigWithCredentials(nugetConfigPath, dstConfig, token); err != nil {
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

// resolveNuGetConfigPath finds the NuGet.Config file.
// If configFile is specified, returns it. Otherwise searches current directory
// and parent directories for NuGet.Config or nuget.config.
func resolveNuGetConfigPath(configFile string) string {
	if configFile != "" {
		if _, err := os.Stat(configFile); err == nil {
			return configFile
		}
		return ""
	}

	// Search current directory and parent directories
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		for _, name := range []string{"NuGet.Config", "nuget.config", "NuGet.config", "nuget.Config"} {
			p := filepath.Join(dir, name)
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// isGitHubPackagesURL returns true if the URL points to a GitHub Packages NuGet registry.
// Matches github.com ("nuget.pkg.github.com"), GHES subdomain style ("nuget.<host>"),
// and the legacy GHES path style ("/_registry/nuget/").
func isGitHubPackagesURL(rawURL string) bool {
	if strings.Contains(rawURL, "nuget.pkg.github.com") || strings.Contains(rawURL, "/_registry/nuget/") {
		return true
	}
	// GHES uses "https://nuget.<host>/<owner>/..." — match any nuget.* subdomain.
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return strings.HasPrefix(u.Hostname(), "nuget.")
}

// writeNuGetConfigWithCredentials reads the NuGet.Config at srcPath, fills in GitHub
// Packages credentials using the provided token, and writes the result to dstPath.
// dstPath may equal srcPath to overwrite the original file in place.
func writeNuGetConfigWithCredentials(srcPath, dstPath, token string) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read NuGet.Config: %w", err)
	}

	var config nugetConfiguration
	if err := xml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse NuGet.Config: %w", err)
	}

	// Collect source keys that point to GitHub Packages
	githubSourceKeys := make(map[string]bool)
	if config.PackageSources != nil {
		for _, item := range config.PackageSources.Items {
			if item.XMLName.Local == "add" && isGitHubPackagesURL(item.Value) {
				githubSourceKeys[item.Key] = true
			}
		}
	}

	if len(githubSourceKeys) == 0 {
		return fmt.Errorf("no GitHub Packages sources found in NuGet.Config")
	}

	// Fill in credentials for GitHub Packages sources
	if config.PackageSourceCredentials == nil {
		config.PackageSourceCredentials = &nugetSourceCredentials{}
	}

	// Track which GitHub sources already have credential entries
	existingCreds := make(map[string]bool)
	for i, src := range config.PackageSourceCredentials.Sources {
		if githubSourceKeys[src.XMLName.Local] {
			existingCreds[src.XMLName.Local] = true
			// Replace the credential items with proper values
			config.PackageSourceCredentials.Sources[i].Items = []nugetAddItem{
				{XMLName: xml.Name{Local: "add"}, Key: "Username", Value: "gh-pkg-kit"},
				{XMLName: xml.Name{Local: "add"}, Key: "ClearTextPassword", Value: token},
			}
		}
	}

	// Add credential entries for GitHub sources that don't have them yet
	for key := range githubSourceKeys {
		if !existingCreds[key] {
			config.PackageSourceCredentials.Sources = append(config.PackageSourceCredentials.Sources, nugetCredentialSource{
				XMLName: xml.Name{Local: key},
				Items: []nugetAddItem{
					{XMLName: xml.Name{Local: "add"}, Key: "Username", Value: "gh-pkg-kit"},
					{XMLName: xml.Name{Local: "add"}, Key: "ClearTextPassword", Value: token},
				},
			})
		}
	}

	output, err := xml.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal NuGet.Config: %w", err)
	}

	xmlHeader := []byte(xml.Header)
	content := append(xmlHeader, output...)
	if err := os.WriteFile(dstPath, content, 0600); err != nil {
		return fmt.Errorf("failed to write NuGet.Config: %w", err)
	}

	return nil
}
