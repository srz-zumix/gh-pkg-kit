package migrator

import (
	"context"
	"errors"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
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

	// Generate destination repository URL for package.json rewriting
	destRepoURL := ""
	if !skipRewritePackageJSON {
		destHost := destRepo.Host
		if destHost == "" {
			destHost = "github.com"
		}
		if destRepo.Name != "" {
			destRepoURL = fmt.Sprintf("https://%s/%s/%s", destHost, destRepo.Owner, destRepo.Name)
		} else {
			destRepoURL = fmt.Sprintf("https://%s/%s", destHost, destRepo.Owner)
		}
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

				// Find and delete the existing version at the destination.
				if deleteErr := deleteDestPackageVersion(ctx, destClient, destOwnerType, destRepo.Owner, "npm", packageName, versionName); deleteErr != nil {
					logger.Error("Failed to delete existing destination version for overwrite", "version", versionName, "error", deleteErr)
					failures = append(failures, fmt.Sprintf("version %d (%s): overwrite failed: %v", v.GetID(), versionName, deleteErr))
					continue
				}

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

// deleteDestPackageVersion finds the version matching versionName at the destination and deletes it.
func deleteDestPackageVersion(ctx context.Context, client *gh.GitHubClient, ownerType gh.OwnerType, owner, packageType, packageName, versionName string) error {
	versions, err := gh.ListPackageVersionsByOwnerType(ctx, client, ownerType, owner, packageType, packageName)
	if err != nil {
		return fmt.Errorf("failed to list destination versions: %w", err)
	}
	for _, v := range versions {
		if v.GetName() == versionName {
			if err := gh.DeletePackageVersionByOwnerType(ctx, client, ownerType, owner, packageType, packageName, v.GetID()); err != nil {
				return fmt.Errorf("failed to delete version %d (%s): %w", v.GetID(), versionName, err)
			}
			return nil
		}
	}
	return fmt.Errorf("version %q not found at destination", versionName)
}
