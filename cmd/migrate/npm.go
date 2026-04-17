package migrate

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-pkg-kit/pkg/migrator"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

// NewNPMCmd creates a command to migrate npm packages between owners
func NewNPMCmd() *cobra.Command {
	var (
		src                        string
		dst                        string
		srcToken                   string
		dstToken                   string
		deleteFlag                 bool
		dryRun                     bool
		overwrite                  bool
		skipRewritePackageJSON     bool
		versionFilter              []string
		latest                     int
		since                      string
		until                       string
	)

	cmd := &cobra.Command{
		Use:   "npm <package-name>",
		Short: "Migrate npm packages between owners",
		Long: `Migrates npm packages from one owner to another within GitHub Packages.
Downloads .tgz tarball files from the source npm registry and pushes them to the destination.
The source owner is resolved from the current repository if --src is not specified.
The source and destination owner types (organization or user) are detected automatically.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			srcPackage := args[0]

			srcRepo, err := parser.Repository(parser.RepositoryOwnerWithHost(src))
			if err != nil {
				return fmt.Errorf("failed to resolve source owner: %w", err)
			}
			destRepo, err := parser.Repository(parser.RepositoryOwnerOrRepo(dst))
			if err != nil {
				return fmt.Errorf("failed to parse destination repository: %w", err)
			}
			clients, err := migrator.SetupClients(srcRepo, destRepo, srcToken, dstToken)
			if err != nil {
				return err
			}

			// Validate: cannot skip rewriting when destination repository is specified
			if skipRewritePackageJSON && clients.DestRepo.Name != "" {
				return fmt.Errorf("cannot skip rewriting package.json when destination repository is specified; either provide only [host/]owner for --dst or omit --skip-rewrite-package-json")
			}

			ctx := cmd.Context()

			// If no repository name was given for the destination, resolve it from the source package metadata.
			if clients.DestRepo.Name == "" {
				if err := migrator.ResolveDestRepo(ctx, clients, "npm", srcPackage); err != nil {
					return fmt.Errorf("failed to resolve destination repository name: %w", err)
				}
			}

			// List source versions and apply filters
			versions, srcOwnerType, err := migrator.ListFilteredVersions(ctx, clients.SrcClient, clients.SrcRepo.Owner, "npm", srcPackage, versionFilter, latest, since, until)
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
				r.RenderPackageVersions(versions, nil)
				return nil
			}

			// Migrate each version
			migrated, failures := migrator.MigrateNPM(ctx, clients.SrcClient, clients.DestClient, clients.SrcRepo, clients.DestRepo, srcPackage, versions, skipRewritePackageJSON, overwrite)

			// Delete migrated versions if requested
			var deleteFailures []string
			if deleteFlag && len(migrated) > 0 {
				deleteFailures = migrator.DeleteMigratedVersions(ctx, clients.SrcClient, srcOwnerType, clients.SrcRepo.Owner, "npm", srcPackage, migrated)
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
	f.StringVarP(&src, "src", "s", "", "Source [host/]owner (default: current repository owner)")
	f.StringVarP(&dst, "dst", "d", "", "Destination [host/]owner[/repo]")
	_ = cmd.MarkFlagRequired("dst")
	f.StringVar(&srcToken, "src-token", "", "Access token for the source owner (overrides gh auth token for source; fallback: $GH_SRC_TOKEN)")
	f.StringVar(&dstToken, "dst-token", "", "Access token for the destination owner (overrides gh auth token for destination; fallback: $GH_DST_TOKEN)")
	f.BoolVar(&deleteFlag, "delete", false, "Delete source versions after successful migration")
	f.BoolVarP(&dryRun, "dry-run", "n", false, "Show what would be migrated without performing the migration")
	f.BoolVar(&overwrite, "overwrite", false, "Overwrite existing versions at destination by deleting them before pushing")
	f.BoolVar(&skipRewritePackageJSON, "skip-rewrite-package-json", false, "Skip rewriting package.json in the tarball (by default, the repository field is rewritten to reflect destination URL)")
	f.StringSliceVar(&versionFilter, "version", nil, "Migrate specific version(s) by ID or name (can be specified multiple times)")
	f.IntVarP(&latest, "latest", "l", 0, "Migrate latest N versions (by creation date)")
	f.StringVar(&since, "since", "", "Migrate versions created on or after this date (RFC3339 or YYYY-MM-DD)")
	f.StringVar(&until, "until", "", "Migrate versions created on or before this date (RFC3339 or YYYY-MM-DD)")

	return cmd
}
