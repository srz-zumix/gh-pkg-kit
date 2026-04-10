package maven

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-pkg-kit/pkg/migrator"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

// NewDownloadCmd creates a command to download Maven artifacts from GitHub Packages
func NewDownloadCmd() *cobra.Command {
	var (
		repo    string
		version string
		outDir  string
	)

	cmd := &cobra.Command{
		Use:   "download <package-name>",
		Short: "Download Maven artifacts from GitHub Packages",
		Long: `Downloads .pom and .jar files from the GitHub Maven registry.
Accepts both colon-separated format (e.g. com.example:my-artifact) and the
GitHub Packages dot-separated format (e.g. com.example.my-artifact).
Version defaults to the latest version if not specified.
The repository defaults to the current repository if --repo is not specified.
Output files are written to --output-dir (default: current directory) as <artifactId>-<version>.<ext>.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packageName := args[0]

			_, artifactID, err := gh.ParseMavenPackageName(packageName)
			if err != nil {
				return fmt.Errorf("invalid package name: %w", err)
			}

			r, err := parser.Repository(parser.RepositoryInput(repo))
			if err != nil {
				return fmt.Errorf("failed to resolve repository: %w", err)
			}
			if r.Name == "" {
				return fmt.Errorf("repository name is required for Maven; specify --repo as [host/]owner/repo")
			}

			g, err := gh.NewGitHubClientWithRepo(r)
			if err != nil {
				return err
			}

			ctx := cmd.Context()

			// If version is not specified, get the latest version
			if version == "" {
				versions, _, err := migrator.ListFilteredVersions(ctx, g, r.Owner, "maven", packageName, nil, 1, "", "")
				if err != nil {
					return fmt.Errorf("failed to list package versions: %w", err)
				}
				if len(versions) == 0 {
					return fmt.Errorf("no versions found for package '%s'", packageName)
				}
				version = versions[0].GetName()
			}

			// Download artifacts
			artifacts, err := gh.DownloadMavenArtifacts(ctx, g, r, packageName, version)
			if err != nil {
				return fmt.Errorf("failed to download '%s' version '%s': %w", packageName, version, err)
			}

			// Write each artifact to file
			if outDir != "" {
				if err := os.MkdirAll(outDir, 0755); err != nil {
					return fmt.Errorf("failed to create output directory '%s': %w", outDir, err)
				}
			}
			for _, artifact := range artifacts {
				// Sanitize components from the API to prevent path traversal.
				safeVersion := strings.ReplaceAll(filepath.Base(version), "..", "_")
				safeClassifier := strings.ReplaceAll(filepath.Base(artifact.Classifier), "..", "_")
				var filename string
				if safeClassifier != "" && safeClassifier != "." {
					filename = fmt.Sprintf("%s-%s-%s.%s", artifactID, safeVersion, safeClassifier, artifact.Ext)
				} else {
					filename = fmt.Sprintf("%s-%s.%s", artifactID, safeVersion, artifact.Ext)
				}
				destPath := filename
				if outDir != "" {
					destPath = filepath.Join(outDir, filename)
				}
				// Ensure the resolved path stays within the intended directory.
				absOut, _ := filepath.Abs(outDir)
				absDest, _ := filepath.Abs(destPath)
				if !strings.HasPrefix(absDest, absOut+string(filepath.Separator)) && absDest != absOut {
					return fmt.Errorf("resolved path %q escapes output directory %q", absDest, absOut)
				}
				if err := os.WriteFile(destPath, artifact.Data, 0644); err != nil {
					return fmt.Errorf("failed to write %s: %w", destPath, err)
				}
				logger.Info("Downloaded", "package", packageName, "version", version, "to", destPath)
			}

			return nil
		},
	}

	f := cmd.Flags()
	f.StringVarP(&repo, "repo", "R", "", "Repository in [host/]owner/repo format (defaults to current repository)")
	f.StringVar(&version, "version", "", "Package version to download (defaults to latest)")
	f.StringVarP(&outDir, "output-dir", "o", "", "Output directory (default: current directory)")

	return cmd
}
