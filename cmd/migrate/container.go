package migrate

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-pkg-kit/pkg/migrator"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

// NewContainerCmd creates a command to migrate container packages between owners
func NewContainerCmd() *cobra.Command {
	return newContainerMigrateCmd("container", false)
}

// newContainerMigrateCmd creates a migrate command for the given container-based package type.
// When requireRepo is true, --src must include the repository name ([HOST/]OWNER/REPO).
func newContainerMigrateCmd(packageType string, requireRepo bool) *cobra.Command {
	var (
		src           string
		dst           string
		srcToken      string
		dstToken      string
		deleteFlag    bool
		dryRun        bool
		rewriteLabels bool
		versions      []string
		latest        int
		since         string
		until         string
	)

	cmd := &cobra.Command{
		Use:   packageType + " <package-name>",
		Short: "Migrate " + packageType + " packages between owners",
		Long: fmt.Sprintf(`Migrates %s packages from one owner to another within GitHub Packages.
Uses the OCI Distribution API to copy image manifests and blobs, including multi-architecture images.
The source owner is resolved from the current repository if --src is not specified.
The source and destination owner types (organization or user) are detected automatically.`, packageType),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			srcPackage := args[0]

			var srcParseOpt parser.RepositoryOption
			if requireRepo {
				srcParseOpt = parser.RepositoryOwnerOrRepo(src)
			} else {
				srcParseOpt = parser.RepositoryOwnerWithHost(src)
			}
			srcRepo, err := parser.Repository(srcParseOpt)
			if err != nil {
				if requireRepo {
					return fmt.Errorf("failed to resolve source repository: %w", err)
				}
				return fmt.Errorf("failed to resolve source owner: %w", err)
			}
			if requireRepo && srcRepo.Name == "" {
				return fmt.Errorf("--src must include the repository name ([HOST/]OWNER/REPO) for %s packages", packageType)
			}
			destRepo, err := parser.Repository(parser.RepositoryOwnerOrRepo(dst))
			if err != nil {
				return fmt.Errorf("failed to parse destination repository: %w", err)
			}
			clients, err := migrator.SetupClients(srcRepo, destRepo, srcToken, dstToken)
			if err != nil {
				return err
			}

			ctx := cmd.Context()

			return migrator.MigrateContainer(ctx, clients.SrcClient, clients.DestClient, migrator.ContainerOptions{
				PackageType:   packageType,
				Src:           clients.SrcRepo,
				SrcPackage:    srcPackage,
				Dest:          clients.DestRepo,
				DestPackage:   srcPackage,
				DeleteFlag:    deleteFlag,
				DryRun:        dryRun,
				RewriteLabels: rewriteLabels,
				Versions:      versions,
				Latest:        latest,
				Since:         since,
				Until:         until,
			})
		},
	}

	f := cmd.Flags()
	srcDesc := "Source [host/]owner (default: current repository owner)"
	if requireRepo {
		srcDesc = "Source [host/]owner/repo (repository name is required)"
	}
	f.StringVarP(&src, "src", "s", "", srcDesc)
	f.StringVarP(&dst, "dst", "d", "", "Destination [host/]owner/[repo]")
	_ = cmd.MarkFlagRequired("dst")
	f.StringVar(&srcToken, "src-token", "", "Access token for the source owner (overrides gh auth token for source; fallback: $GH_SRC_TOKEN)")
	f.StringVar(&dstToken, "dst-token", "", "Access token for the destination owner (overrides gh auth token for destination; fallback: $GH_DST_TOKEN)")
	f.BoolVar(&deleteFlag, "delete", false, "Delete source versions after successful migration")
	f.BoolVarP(&dryRun, "dry-run", "n", false, "Show what would be migrated without performing the migration")
	f.BoolVar(&rewriteLabels, "rewrite-labels", false, "Rewrite OCI image config labels (e.g. org.opencontainers.image.source) to reflect the destination owner/host (changes image digest)")
	f.StringSliceVar(&versions, "version", nil, "Migrate specific version(s) by ID or name (can be specified multiple times)")
	f.IntVarP(&latest, "latest", "l", 0, "Migrate latest N versions (by creation date)")
	f.StringVar(&since, "since", "", "Migrate versions created on or after this date (RFC3339 or YYYY-MM-DD)")
	f.StringVar(&until, "until", "", "Migrate versions created on or before this date (RFC3339 or YYYY-MM-DD)")

	return cmd
}
