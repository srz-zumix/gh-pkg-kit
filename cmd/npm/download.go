package npm

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-pkg-kit/pkg/migrator"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

// NewDownloadCmd creates a command to download an npm package from GitHub Packages
func NewDownloadCmd() *cobra.Command {
	var (
		owner   string
		version string
		output  string
	)

	cmd := &cobra.Command{
		Use:   "download <package-name>",
		Short: "Download an npm package from GitHub Packages",
		Long: `Downloads a .tgz tarball file from the GitHub npm registry.
Version defaults to the latest version if not specified.
The owner is resolved from the current repository if --owner is not specified.
The output file defaults to <package-name>.<version>.tgz in the current directory.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packageName := args[0]

			repo, err := parser.Repository(parser.RepositoryOwnerWithHost(owner))
			if err != nil {
				return fmt.Errorf("failed to resolve owner: %w", err)
			}

			g, err := gh.NewGitHubClientWithRepo(repo)
			if err != nil {
				return err
			}

			ctx := cmd.Context()

			// If version is not specified, get the latest version
			if version == "" {
				versions, _, err := migrator.ListFilteredVersions(ctx, g, repo.Owner, "npm", packageName, nil, 1, "", "")
				if err != nil {
					return fmt.Errorf("failed to list package versions: %w", err)
				}
				if len(versions) == 0 {
					return fmt.Errorf("no versions found for package '%s'", packageName)
				}
				version = versions[0].GetName()
			}

			destPath := output
			if destPath == "" {
				// Sanitize package name for use as a filename:
				// scoped packages (e.g. "@scope/pkg") contain '/' which would be
				// interpreted as a path separator by os.WriteFile.
				safePackageName := strings.ReplaceAll(packageName, "/", "-")
				destPath = fmt.Sprintf("%s.%s.tgz", safePackageName, version)
			}

			// Download the npm package tarball
			tarballData, err := gh.DownloadNPMPackage(ctx, g, repo, packageName, version)
			if err != nil {
				return fmt.Errorf("failed to download '%s' version '%s': %w", packageName, version, err)
			}

			// Write to file
			if err := os.WriteFile(destPath, tarballData, 0644); err != nil {
				return fmt.Errorf("failed to write tarball to file: %w", err)
			}

			logger.Info("Downloaded", "package", packageName, "version", version, "to", destPath)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVarP(&owner, "owner", "o", "", "[HOST/]OWNER (defaults to current repository owner)")
	f.StringVar(&version, "version", "", "Package version to download (defaults to latest)")
	f.StringVar(&output, "output", "", "Output file path (default: <package-name>.<version>.tgz)")

	return cmd
}
