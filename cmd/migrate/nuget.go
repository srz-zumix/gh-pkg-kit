package migrate

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-pkg-kit/pkg/migrator"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

// NewNuGetCmd creates a command to migrate NuGet packages between owners
func NewNuGetCmd() *cobra.Command {
	var (
		src                     string
		dst                     string
		srcToken                string
		dstToken                string
		deleteFlag              bool
		dryRun                  bool
		skipRewriteRepository   bool
		versionIDs              []int64
		latest                  int
		since                   string
		until                   string
	)

	cmd := &cobra.Command{
		Use:   "nuget <package-name>",
		Short: "Migrate NuGet packages between owners",
		Long: `Migrates NuGet packages from one owner to another within GitHub Packages.
Downloads .nupkg files from the source NuGet registry and pushes them to the destination.
The source owner is resolved from the current repository if --src is not specified.
The source and destination owner types (organization or user) are detected automatically.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			srcPackage := args[0]

			// Setup repositories and clients
			clients, err := migrator.SetupClientsAndRepositories(src, dst, srcToken, dstToken)
			if err != nil {
				return err
			}

			// Validate: cannot skip rewriting when destination repository is specified
			if skipRewriteRepository && clients.DestRepo.Name != "" {
				return fmt.Errorf("cannot skip rewriting <repository> element when destination repository is specified; either provide only [host/]owner for --dst or omit --skip-rewrite-repository")
			}

			ctx := cmd.Context()

			// List source versions and apply filters
			versions, srcOwnerType, err := migrator.ListFilteredVersions(ctx, clients.SrcClient, clients.SrcRepo.Owner, "nuget", srcPackage, versionIDs, latest, since, until)
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

			// Generate repoURL from destination host/owner/repo
			repoURL := parser.GetRepositoryURL(clients.DestRepo)

			// Migrate each version
			var migrated []int64
			var failures []string
			for _, v := range versions {
				versionName := v.GetName()
				logger.Info("Migrating NuGet package", "package", srcPackage, "version", versionName)

				tmpSrc, err := gh.DownloadNuGetPackage(ctx, clients.SrcClient, clients.SrcRepo, srcPackage, versionName, "")
				if err != nil {
					logger.Error("Failed to download NuGet package", "version", versionName, "error", err)
					failures = append(failures, fmt.Sprintf("version %d (%s): download failed: %v", v.GetID(), versionName, err))
					continue
				}

				var rewritten *os.File
				if !skipRewriteRepository {
					// Rewrite the <repository> element in the .nuspec before pushing.
					rewriteResult, err := gh.RewriteNuPkgRepository(tmpSrc, repoURL, "")
					tmpSrc.Close()
					os.Remove(tmpSrc.Name())
					if err != nil {
						logger.Error("Failed to rewrite nuspec repository", "version", versionName, "error", err)
						failures = append(failures, fmt.Sprintf("version %d (%s): nuspec rewrite failed: %v", v.GetID(), versionName, err))
						continue
					}
					rewritten = rewriteResult
				} else {
					// Rewriting is skipped; use the original file
					rewritten = tmpSrc
				}

				pushErr := gh.PushNuGetPackage(ctx, clients.DestClient, clients.DestRepo, rewritten)
				rewritten.Close()
				os.Remove(rewritten.Name())
				if pushErr != nil {
					logger.Error("Failed to push NuGet package", "version", versionName, "error", pushErr)
					failures = append(failures, fmt.Sprintf("version %d (%s): push failed: %v", v.GetID(), versionName, pushErr))
					continue
				}

				logger.Info("Migrated NuGet package", "version", versionName)
				migrated = append(migrated, v.GetID())
			}

			// Delete migrated versions if requested
			var deleteFailures []string
			if deleteFlag && len(migrated) > 0 {
				deleteFailures = migrator.DeleteMigratedVersions(ctx, clients.SrcClient, srcOwnerType, clients.SrcRepo.Owner, "nuget", srcPackage, migrated)
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
	f.BoolVar(&skipRewriteRepository, "skip-rewrite-repository", false, "Skip rewriting <repository> element in .nuspec (by default, the element is rewritten to reflect destination URL)")
	f.Int64SliceVar(&versionIDs, "version", nil, "Migrate specific version(s) by ID (can be specified multiple times)")
	f.IntVarP(&latest, "latest", "l", 0, "Migrate latest N versions (by creation date)")
	f.StringVar(&since, "since", "", "Migrate versions created on or after this date (RFC3339 or YYYY-MM-DD)")
	f.StringVar(&until, "until", "", "Migrate versions created on or before this date (RFC3339 or YYYY-MM-DD)")

	return cmd
}
