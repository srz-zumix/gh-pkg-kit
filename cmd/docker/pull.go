package docker

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-pkg-kit/pkg/migrator"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

// NewPullCmd creates a command to pull a docker image from GitHub Packages to a local tarball
func NewPullCmd() *cobra.Command {
	var (
		owner  string
		tag    string
		output string
		dryRun bool
	)

	cmd := &cobra.Command{
		Use:   "pull <package-name>",
		Short: "Pull a docker image from GitHub Packages to a local tarball",
		Long: "Pulls a docker image from the GitHub Container Registry and saves it as a Docker-loadable tarball.\n" +
			"The tag defaults to \"latest\" if not specified.\n" +
			"The owner is resolved from the current repository if --owner is not specified.\n" +
			"The output file defaults to <package-name>-<tag>.tar in the current directory.\n" +
			"The saved tarball can be loaded with: docker load -i <output-file>",
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

			return migrator.PullContainerToFile(ctx, g, migrator.PullContainerOptions{
				PackageType: "docker",
				Src:         repo,
				SrcPackage:  packageName,
				Tag:         tag,
				Output:      output,
				DryRun:      dryRun,
			})
		},
	}

	f := cmd.Flags()
	f.StringVarP(&owner, "owner", "o", "", "[host/]owner (defaults to current repository owner)")
	f.StringVarP(&tag, "tag", "t", "", "Image tag to pull (default: \"latest\")")
	f.StringVar(&output, "output", "", "Output file path (default: <package-name>-<tag>.tar)")
	f.BoolVarP(&dryRun, "dry-run", "n", false, "Show what would be pulled without performing the pull")

	return cmd
}
