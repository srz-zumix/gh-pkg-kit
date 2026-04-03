package nuget

import (
	"archive/zip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2/pkg/auth"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

// dotnetToolsManifest represents the structure of .config/dotnet-tools.json
type dotnetToolsManifest struct {
	Version int                        `json:"version"`
	IsRoot  bool                       `json:"isRoot"`
	Tools   map[string]dotnetToolEntry `json:"tools"`
}

// dotnetToolEntry represents a single tool entry in the dotnet tools manifest
type dotnetToolEntry struct {
	Version  string   `json:"version"`
	Commands []string `json:"commands"`
}

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

// NewToolRestoreCmd creates a command to download NuGet tool packages from GitHub Packages
// and run 'dotnet tool restore' with a local source.
func NewToolRestoreCmd() *cobra.Command {
	var (
		owner        string
		toolManifest string
		configFile   string
		dryRun       bool
	)

	cmd := &cobra.Command{
		Use:   "tool-restore [-- dotnet-tool-restore-args...]",
		Short: "Download NuGet tools from GitHub Packages and run dotnet tool restore",
		Long: `Downloads .nupkg tool packages from GitHub Packages using the gh auth token,
then runs 'dotnet tool restore' with the downloaded packages as a local source.
This avoids the need to configure GitHub Packages credentials in nuget.config.

If a NuGet.Config file is found (via --configfile or auto-detected), a temporary
copy is created with GitHub Packages credentials filled in using the gh auth token.
This resolves parse errors caused by incomplete credential entries in nuget.config.

The tool manifest defaults to .config/dotnet-tools.json.
Packages that are not found on GitHub Packages are silently skipped,
allowing dotnet to resolve them from other configured NuGet sources.

Extra arguments after -- are passed through to 'dotnet tool restore'.`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if toolManifest == "" {
				toolManifest = filepath.Join(".config", "dotnet-tools.json")
			}

			manifest, err := readToolManifest(toolManifest)
			if err != nil {
				return fmt.Errorf("failed to read tool manifest '%s': %w", toolManifest, err)
			}

			if len(manifest.Tools) == 0 {
				logger.Info("No tools found in manifest", "manifest", toolManifest)
				return nil
			}

			repo, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("failed to resolve owner: %w", err)
			}

			g, err := gh.NewGitHubClientWithRepo(repo)
			if err != nil {
				return err
			}

			ctx := cmd.Context()

			// Create temp directory for downloaded packages and temporary config
			tmpDir, err := os.MkdirTemp("", "gh-pkg-kit-nuget-*")
			if err != nil {
				return fmt.Errorf("failed to create temp directory: %w", err)
			}
			defer os.RemoveAll(tmpDir)

			// Download each tool package from GitHub Packages
			downloaded := 0
			for packageName, tool := range manifest.Tools {
				destPath := filepath.Join(tmpDir, fmt.Sprintf("%s.%s.nupkg", packageName, tool.Version))

				if dryRun {
					logger.Info("Would download", "package", packageName, "version", tool.Version)
					downloaded++
					continue
				}

				f, err := gh.DownloadNuGetPackage(ctx, g, repo, packageName, tool.Version, destPath)
				if err != nil {
					logger.Warn("Skipping package (not found on GitHub Packages)", "package", packageName, "version", tool.Version, "error", err)
					continue
				}
				if closeErr := f.Close(); closeErr != nil {
					logger.Error("Failed to close downloaded file", "package", packageName, "error", closeErr)
				}

				// Validate the downloaded file is a valid nupkg (ZIP archive)
				if !isValidNupkg(destPath) {
					logger.Warn("Skipping package (downloaded file is not a valid nupkg)", "package", packageName, "version", tool.Version)
					_ = os.Remove(destPath)
					continue
				}

				logger.Info("Downloaded", "package", packageName, "version", tool.Version)
				downloaded++
			}

			if dryRun {
				logger.Info("Dry run: would run dotnet tool restore", "downloaded", downloaded, "total", len(manifest.Tools))
				return nil
			}

			if downloaded == 0 {
				logger.Warn("No packages downloaded from GitHub Packages, running dotnet tool restore without local source")
			}

			// Resolve the NuGet.Config path and create a temporary copy with credentials
			nugetConfigPath := resolveNuGetConfigPath(configFile)

			// Build dotnet tool restore command
			dotnetArgs := []string{"tool", "restore"}
			if downloaded > 0 {
				dotnetArgs = append(dotnetArgs, "--add-source", tmpDir)
			}

			if nugetConfigPath != "" {
				host := repo.Host
				if host == "" {
					host = "github.com"
				}
				token, _ := auth.TokenForHost(host)
				if token != "" {
					tmpConfig, err := createTempNuGetConfig(nugetConfigPath, tmpDir, token)
					if err != nil {
						logger.Warn("Failed to create temporary NuGet.Config, using original", "error", err)
					} else {
						logger.Info("Using temporary NuGet.Config with gh auth credentials", "config", tmpConfig)
						dotnetArgs = append(dotnetArgs, "--configfile", tmpConfig)
					}
				}
			}

			dotnetArgs = append(dotnetArgs, args...)

			logger.Info("Running dotnet tool restore", "args", dotnetArgs)
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
	f.StringVarP(&owner, "owner", "o", "", "Owner name (defaults to current repository owner)")
	f.StringVar(&toolManifest, "tool-manifest", "", "Path to the dotnet tool manifest (default: .config/dotnet-tools.json)")
	f.StringVar(&configFile, "configfile", "", "Path to NuGet.Config (auto-detected if not specified)")
	f.BoolVarP(&dryRun, "dry-run", "n", false, "Show what would be done without downloading or running dotnet")

	return cmd
}

// readToolManifest reads and parses the dotnet tool manifest file
func readToolManifest(path string) (*dotnetToolsManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var manifest dotnetToolsManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
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

// createTempNuGetConfig reads the NuGet.Config at srcPath, fills in GitHub Packages
// credentials using the provided token, and writes the result to a temporary file
// in tmpDir. Returns the path to the temporary config file.
func createTempNuGetConfig(srcPath, tmpDir, token string) (string, error) {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return "", fmt.Errorf("failed to read NuGet.Config: %w", err)
	}

	var config nugetConfiguration
	if err := xml.Unmarshal(data, &config); err != nil {
		return "", fmt.Errorf("failed to parse NuGet.Config: %w", err)
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
		return "", fmt.Errorf("no GitHub Packages sources found in NuGet.Config")
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
		return "", fmt.Errorf("failed to marshal NuGet.Config: %w", err)
	}

	tmpConfigPath := filepath.Join(tmpDir, "NuGet.Config")
	xmlHeader := []byte(xml.Header)
	content := append(xmlHeader, output...)
	if err := os.WriteFile(tmpConfigPath, content, 0600); err != nil {
		return "", fmt.Errorf("failed to write temporary NuGet.Config: %w", err)
	}

	return tmpConfigPath, nil
}

// isValidNupkg checks whether the file at path is a valid nupkg (ZIP archive).
func isValidNupkg(path string) bool {
	r, err := zip.OpenReader(path)
	if err != nil {
		return false
	}
	r.Close()
	return true
}
