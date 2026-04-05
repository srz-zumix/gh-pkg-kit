package container

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-pkg-kit/pkg/migrator"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

// NewPullCmd creates a command to pull a container image from ghcr.io to a local tarball.
func NewPullCmd() *cobra.Command {
	return NewPullCmdFor("container", false)
}

// NewPullCmdFor creates a pull command for the given container-based package type.
// When requireRepo is true, --owner must include the repository name ([host/]owner/repo).
func NewPullCmdFor(packageType string, requireRepo bool) *cobra.Command {
	var (
		owner  string
		tag    string
		output string
		dryRun bool
	)

	registryDesc := "GitHub Container Registry (ghcr.io)"
	if packageType == "docker" {
		registryDesc = "legacy Docker Package Registry (docker.pkg.github.com)"
	}

	ownerDesc := "[host/]owner (defaults to current repository owner)"
	if requireRepo {
		ownerDesc = "[host/]owner/repo (repository name is required)"
	}

	cmd := &cobra.Command{
		Use:   "pull <package-name>",
		Short: "Pull a " + packageType + " image from GitHub Packages to a local tarball",
		Long: "Pulls a " + packageType + " image from the " + registryDesc + " and saves it as a Docker-loadable tarball.\n" +
			"The tag defaults to \"latest\" if not specified.\n" +
			"The output file defaults to <package-name>-<tag>.tar in the current directory.\n" +
			"The saved tarball can be loaded with: docker load -i <output-file>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packageName := args[0]

			var ownerOpt parser.RepositoryOption
			if requireRepo {
				ownerOpt = parser.RepositoryOwnerOrRepo(owner)
			} else {
				ownerOpt = parser.RepositoryOwnerWithHost(owner)
			}
			repo, err := parser.Repository(ownerOpt)
			if err != nil {
				return fmt.Errorf("failed to resolve owner: %w", err)
			}
			if requireRepo && repo.Name == "" {
				return fmt.Errorf("--owner must include the repository name ([host/]owner/repo) for %s packages", packageType)
			}

			g, err := gh.NewGitHubClientWithRepo(repo)
			if err != nil {
				return err
			}

			ctx := cmd.Context()

			return migrator.PullContainerToFile(ctx, g, migrator.PullContainerOptions{
				PackageType: packageType,
				Src:         repo,
				SrcPackage:  packageName,
				Tag:         tag,
				Output:      output,
				DryRun:      dryRun,
			})
		},
	}

	f := cmd.Flags()
	f.StringVarP(&owner, "owner", "o", "", ownerDesc)
	f.StringVarP(&tag, "tag", "t", "", "Image tag to pull (default: \"latest\")")
	f.StringVar(&output, "output", "", "Output file path (default: <package-name>-<tag>.tar)")
	f.BoolVarP(&dryRun, "dry-run", "n", false, "Show what would be pulled without performing the pull")

	return cmd
}
