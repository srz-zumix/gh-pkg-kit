package migrator

import (
	"context"
	"errors"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

// MigrateNPM migrates npm packages from source to destination.
func MigrateNPM(
	ctx context.Context,
	srcClient *gh.GitHubClient,
	destClient *gh.GitHubClient,
	srcRepo, destRepo repository.Repository,
	packageName string,
	selectVersions []*PackageVersion,
	skipRewritePackageJSON bool,
	overwrite bool,
) ([]int64, []string) {
	var migrated []int64
	var failures []string

	if len(selectVersions) == 0 {
		return migrated, failures
	}

	// Detect destination owner type once (needed for delete-on-overwrite).
	var destOwnerType gh.OwnerType
	var destOwnerTypeDetected bool
	// destVersionCache maps destination version name→ID, populated lazily on the
	// first overwrite conflict to avoid a repeated ListPackageVersions API call
	// per conflicting version. Entries are removed after successful deletion.
	var destVersionCache map[string]int64

	// Generate destination repository URL for package.json rewriting
	destRepoURL := ""
	if !skipRewritePackageJSON {
		destRepoURL = parser.GetRepositoryURL(destRepo)
	}

	for _, v := range selectVersions {
		versionName := v.GetName()
		logger.Info("Migrating npm package", "package", packageName, "version", versionName)

		// Download tarball
		tarballData, err := gh.DownloadNPMPackage(ctx, srcClient, srcRepo, packageName, versionName)

		if err != nil {
			logger.Error("Failed to download npm package", "version", versionName, "error", err)
			failures = append(failures, fmt.Sprintf("version %d (%s): download failed: %v", v.GetID(), versionName, err))
			continue
		}

		// Rewrite package.json if needed
		var modified []byte
		if !skipRewritePackageJSON {
			rewritten, err := gh.RewriteNPMPackageJSON(tarballData, destRepoURL)
			if err != nil {
				logger.Error("Failed to rewrite npm package.json", "version", versionName, "error", err)
				failures = append(failures, fmt.Sprintf("version %d (%s): rewrite failed: %v", v.GetID(), versionName, err))
				continue
			}
			modified = rewritten
		} else {
			modified = tarballData
		}

		// Push to destination
		if err := gh.PushNPMPackage(ctx, destClient, destRepo, packageName, modified); err != nil {
			if overwrite && errors.Is(err, gh.ErrPackageVersionExists) {
				// Detect dest owner type on first need.
				if !destOwnerTypeDetected {
					destOwnerType, err = gh.DetectOwnerType(ctx, destClient, destRepo.Owner)
					if err != nil {
						logger.Error("Failed to detect destination owner type", "error", err)
						failures = append(failures, fmt.Sprintf("version %d (%s): overwrite failed: %v", v.GetID(), versionName, err))
						continue
					}
					destOwnerTypeDetected = true
				}

				// Populate the version name→ID cache once on the first conflict.
				if destVersionCache == nil {
					versions, listErr := gh.ListPackageVersionsByOwnerType(ctx, destClient, destOwnerType, destRepo.Owner, "npm", packageName)
					if listErr != nil {
						logger.Error("Failed to list destination versions for overwrite", "error", listErr)
						failures = append(failures, fmt.Sprintf("version %d (%s): overwrite failed: %v", v.GetID(), versionName, listErr))
						continue
					}
					destVersionCache = make(map[string]int64, len(versions))
					for _, dv := range versions {
						destVersionCache[dv.GetName()] = dv.GetID()
					}
				}

				// Find and delete the existing version at the destination.
				destVersionID, ok := destVersionCache[versionName]
				if !ok {
					logger.Error("Destination version not found for overwrite", "version", versionName)
					failures = append(failures, fmt.Sprintf("version %d (%s): overwrite failed: version not found at destination", v.GetID(), versionName))
					continue
				}
				if deleteErr := gh.DeletePackageVersionByOwnerType(ctx, destClient, destOwnerType, destRepo.Owner, "npm", packageName, destVersionID); deleteErr != nil {
					logger.Error("Failed to delete existing destination version for overwrite", "version", versionName, "error", deleteErr)
					failures = append(failures, fmt.Sprintf("version %d (%s): overwrite failed: %v", v.GetID(), versionName, deleteErr))
					continue
				}
				delete(destVersionCache, versionName)

				// Retry push after deletion.
				if err := gh.PushNPMPackage(ctx, destClient, destRepo, packageName, modified); err != nil {
					logger.Error("Failed to push npm package after overwrite delete", "version", versionName, "error", err)
					failures = append(failures, fmt.Sprintf("version %d (%s): push failed: %v", v.GetID(), versionName, err))
					continue
				}
			} else {
				logger.Error("Failed to push npm package", "version", versionName, "error", err)
				failures = append(failures, fmt.Sprintf("version %d (%s): push failed: %v", v.GetID(), versionName, err))
				continue
			}
		}

		logger.Info("Migrated npm package", "version", versionName)
		migrated = append(migrated, v.GetID())
	}

	return migrated, failures
}
