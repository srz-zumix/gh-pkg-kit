package migrator

import (
	"context"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
)

// MigrateRubyGems migrates RubyGems packages from source to destination.
func MigrateRubyGems(
	ctx context.Context,
	srcClient *gh.GitHubClient,
	destClient *gh.GitHubClient,
	srcRepo, destRepo repository.Repository,
	packageName string,
	selectVersions []*PackageVersion,
) ([]int64, []string) {
	var migrated []int64
	var failures []string

	if len(selectVersions) == 0 {
		return migrated, failures
	}

	for _, v := range selectVersions {
		versionName := v.GetName()
		logger.Info("Migrating RubyGems package", "package", packageName, "version", versionName)

		// Download .gem file
		gemData, err := gh.DownloadRubyGemsPackage(ctx, srcClient, srcRepo, packageName, versionName)
		if err != nil {
			logger.Error("Failed to download RubyGems package", "version", versionName, "error", err)
			failures = append(failures, fmt.Sprintf("version %d (%s): download failed: %v", v.GetID(), versionName, err))
			continue
		}

		// Rewrite github_repo metadata to reference the destination repository.
		// GitHub validates that github_repo points to a repository on the target instance,
		// so migrating across instances requires updating this field.
		gemData, err = gh.RewriteRubyGemsGitHubRepo(gemData, destRepo)
		if err != nil {
			logger.Error("Failed to rewrite gem metadata", "version", versionName, "error", err)
			failures = append(failures, fmt.Sprintf("version %d (%s): rewrite failed: %v", v.GetID(), versionName, err))
			continue
		}

		// Push to destination
		if err := gh.PushRubyGemsPackage(ctx, destClient, destRepo, gemData); err != nil {
			logger.Error("Failed to push RubyGems package", "version", versionName, "error", err)
			failures = append(failures, fmt.Sprintf("version %d (%s): push failed: %v", v.GetID(), versionName, err))
			continue
		}

		logger.Info("Migrated RubyGems package", "version", versionName)
		migrated = append(migrated, v.GetID())
	}

	return migrated, failures
}
