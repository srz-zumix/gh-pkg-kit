package migrate

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-pkg-kit/pkg/migrator"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

// NewMavenCmd creates a command to migrate Maven packages between owners
func NewMavenCmd() *cobra.Command {
	var (
		src        string
		dst        string
		srcToken   string
		dstToken   string
		deleteFlag bool
		dryRun     bool
		overwrite  bool
		versionIDs []int64
		latest     int
		since      string
		until      string
	)

	cmd := &cobra.Command{
		Use:   "maven <package-name>",
		Short: "Migrate Maven packages between owners",
		Long: `Migrates Maven packages from one owner to another within GitHub Packages.
Downloads .pom and .jar files from the source Maven registry and pushes them to the destination.
Accepts both colon-separated format (e.g. com.example:my-artifact) and the
GitHub Packages dot-separated format (e.g. com.example.my-artifact).
The source repository is resolved from the current repository if --src is not specified.
Both --src and --dst must include a repository name ([host/]owner/repo) because the Maven
GitHub Packages registry URL includes the repository context.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			srcPackage := args[0]

			// Validate Maven package name format
			if _, _, err := gh.ParseMavenPackageName(srcPackage); err != nil {
				return fmt.Errorf("invalid package name: %w", err)
			}

			srcRepo, err := parser.Repository(parser.RepositoryInput(src))
			if err != nil {
				return fmt.Errorf("failed to resolve source repository: %w", err)
			}
			destRepo, err := parser.Repository(parser.RepositoryOwnerOrRepo(dst))
			if err != nil {
				return fmt.Errorf("failed to parse destination repository: %w", err)
			}
			clients, err := migrator.SetupClients(srcRepo, destRepo, srcToken, dstToken)
			if err != nil {
				return err
			}

			if clients.SrcRepo.Name == "" {
				return fmt.Errorf("source repository name is required for Maven; specify --src as [host/]owner/repo")
			}
			if clients.DestRepo.Name == "" {
				return fmt.Errorf("--dst must include a repository name for Maven (e.g. owner/repo)")
			}

			ctx := cmd.Context()

			// List source versions and apply filters
			versions, srcOwnerType, err := migrator.ListFilteredVersions(ctx, clients.SrcClient, clients.SrcRepo.Owner, "maven", srcPackage, versionIDs, latest, since, until)
			if err != nil {
				return err
			}

			if len(versions) == 0 {
				logger.Info("No versions to migrate")
				return nil
			}

			if dryRun {
				logger.Info("Dry run: migration plan",
					"src", parser.GetRepositoryFullNameWithHost(clients.SrcRepo),
					"dest", parser.GetRepositoryFullNameWithHost(clients.DestRepo),
					"package", srcPackage,
					"versions", len(versions),
				)
				r := render.NewRenderer(nil)
				return r.RenderPackageVersions(versions, nil)
			}

			// Migrate each version
			migrated, failures := migrator.MigrateMaven(ctx, clients.SrcClient, clients.DestClient, clients.SrcRepo, clients.DestRepo, srcPackage, versions, overwrite)

			// Delete migrated versions if requested
			var deleteFailures []string
			if deleteFlag && len(migrated) > 0 {
				deleteFailures = migrator.DeleteMigratedVersions(ctx, clients.SrcClient, srcOwnerType, clients.SrcRepo.Owner, "maven", srcPackage, migrated)
			}

			// Report
			logger.Info("Migration complete", "migrated", len(migrated), "failed", len(failures), "delete_failed", len(deleteFailures))
			var errs []string
			if len(failures) > 0 {
				errs = append(errs, fmt.Sprintf("some versions failed to migrate: %s", strings.Join(failures, "; ")))
			}
			if len(deleteFailures) > 0 {
				errs = append(errs, fmt.Sprintf("some source versions failed to delete: %s", strings.Join(deleteFailures, "; ")))
			}
			if len(errs) > 0 {
				return fmt.Errorf("%s", strings.Join(errs, "\n"))
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVarP(&src, "src", "s", "", "Source [host/]owner/repo (default: current repository)")
	f.StringVarP(&dst, "dst", "d", "", "Destination [host/]owner/repo")
	_ = cmd.MarkFlagRequired("dst")
	f.StringVar(&srcToken, "src-token", "", "Access token for the source owner (overrides gh auth token for source; fallback: $GH_SRC_TOKEN)")
	f.StringVar(&dstToken, "dst-token", "", "Access token for the destination owner (overrides gh auth token for destination; fallback: $GH_DST_TOKEN)")
	f.BoolVar(&deleteFlag, "delete", false, "Delete source versions after successful migration")
	f.BoolVar(&overwrite, "overwrite", false, "Overwrite existing versions at the destination (delete and re-push on 409 conflict)")
	f.BoolVarP(&dryRun, "dry-run", "n", false, "Show what would be migrated without performing the migration")
	f.Int64SliceVar(&versionIDs, "version", nil, "Migrate specific version(s) by ID (can be specified multiple times)")
	f.IntVarP(&latest, "latest", "l", 0, "Migrate latest N versions (by creation date)")
	f.StringVar(&since, "since", "", "Migrate versions created on or after this date (RFC3339 or YYYY-MM-DD)")
	f.StringVar(&until, "until", "", "Migrate versions created on or before this date (RFC3339 or YYYY-MM-DD)")

	return cmd
}
