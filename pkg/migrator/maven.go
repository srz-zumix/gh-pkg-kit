package migrator

import (
	"context"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
)

// MigrateMaven migrates Maven packages from source to destination.
// Both srcRepo and destRepo must have a non-empty Name field because the Maven
// GitHub Packages registry URL includes the repository name.
func MigrateMaven(
	ctx context.Context,
	srcClient *gh.GitHubClient,
	destClient *gh.GitHubClient,
	srcRepo, destRepo repository.Repository,
	packageName string,
	selectVersions []*PackageVersion,
	overwrite bool,
) ([]int64, []string) {
	var migrated []int64
	var failures []string

	if srcRepo.Name == "" || destRepo.Name == "" {
		return migrated, []string{
			fmt.Sprintf(
				"source and destination repositories for Maven migration must include a repository name in [HOST/]OWNER/REPO format: src=%q dest=%q",
				srcRepo.Name,
				destRepo.Name,
			),
		}
	}

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

	for _, v := range selectVersions {
		versionName := v.GetName()
		logger.Info("Migrating Maven package", "package", packageName, "version", versionName)

		// Download all primary artifacts (.pom + .jar)
		artifacts, err := gh.DownloadMavenArtifacts(ctx, srcClient, srcRepo, packageName, versionName)
		if err != nil {
			logger.Error("Failed to download Maven artifacts", "version", versionName, "error", err)
			failures = append(failures, fmt.Sprintf("version %d (%s): download failed: %v", v.GetID(), versionName, err))
			continue
		}

		// Push each artifact to destination.
		// Maven artifacts are immutable; a 409 means the artifact already exists at the
		// destination. With --overwrite the entire package version is deleted and all
		// artifacts are re-pushed from the beginning; without it the version is skipped.
		var pushFailed bool
		var alreadyExists bool
		var needsOverwrite bool
		for _, artifact := range artifacts {
			if err := gh.PushMavenArtifact(ctx, destClient, destRepo, packageName, versionName, artifact.Classifier, artifact.Ext, artifact.Data); err != nil {
				if gh.IsMavenConflictError(err) {
					if overwrite {
						needsOverwrite = true
						break
					}
					logger.Info("Maven artifact already exists at destination, skipping", "version", versionName, "ext", artifact.Ext)
					alreadyExists = true
					break
				}
				logger.Error("Failed to push Maven artifact", "version", versionName, "ext", artifact.Ext, "error", err)
				failures = append(failures, fmt.Sprintf("version %d (%s): push %s failed: %v", v.GetID(), versionName, artifact.Ext, err))
				pushFailed = true
				break
			}
		}

		// Handle overwrite: delete the destination version/package, then re-push all artifacts.
		if needsOverwrite {
			// Detect dest owner type on first need.
			if !destOwnerTypeDetected {
				var detectErr error
				destOwnerType, detectErr = gh.DetectOwnerType(ctx, destClient, destRepo.Owner)
				if detectErr != nil {
					logger.Error("Failed to detect destination owner type", "error", detectErr)
					failures = append(failures, fmt.Sprintf("version %d (%s): overwrite failed: %v", v.GetID(), versionName, detectErr))
					continue
				}
				destOwnerTypeDetected = true
			}
			// Populate the version name→ID cache once on the first conflict.
			if destVersionCache == nil {
				versions, listErr := gh.ListPackageVersionsByOwnerType(ctx, destClient, destOwnerType, destRepo.Owner, "maven", packageName)
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
			// Delete the existing destination version.
			destVersionID, ok := destVersionCache[versionName]
			if !ok {
				logger.Error("Destination version not found for overwrite", "version", versionName)
				failures = append(failures, fmt.Sprintf("version %d (%s): overwrite failed: version not found at destination", v.GetID(), versionName))
				continue
			}
			// Try deleting the single version first. GitHub API returns 400
			// when attempting to delete the last remaining version of a package;
			// in that case fall back to deleting the entire package.
			deleteErr := gh.DeletePackageVersionByOwnerType(ctx, destClient, destOwnerType, destRepo.Owner, "maven", packageName, destVersionID)
			if deleteErr != nil && gh.IsHTTPBadRequest(deleteErr) {
				deleteErr = gh.DeletePackageByOwnerType(ctx, destClient, destOwnerType, destRepo.Owner, "maven", packageName)
			}
			if deleteErr != nil {
				logger.Error("Failed to delete existing destination version for overwrite", "version", versionName, "error", deleteErr)
				failures = append(failures, fmt.Sprintf("version %d (%s): overwrite failed: %v", v.GetID(), versionName, deleteErr))
				continue
			}
			delete(destVersionCache, versionName)

			// Re-push all artifacts from the beginning after deletion.
			for _, artifact := range artifacts {
				if retryErr := gh.PushMavenArtifact(ctx, destClient, destRepo, packageName, versionName, artifact.Classifier, artifact.Ext, artifact.Data); retryErr != nil {
					logger.Error("Failed to push Maven artifact after overwrite delete", "version", versionName, "ext", artifact.Ext, "error", retryErr)
					failures = append(failures, fmt.Sprintf("version %d (%s): push %s failed: %v", v.GetID(), versionName, artifact.Ext, retryErr))
					pushFailed = true
					break
				}
			}
		}

		if pushFailed {
			continue
		}
		if alreadyExists {
			logger.Info("Skipped Maven package version (already at destination)", "version", versionName)
			continue
		}

		logger.Info("Migrated Maven package", "version", versionName, "artifacts", len(artifacts))
		migrated = append(migrated, v.GetID())
	}

	return migrated, failures
}
