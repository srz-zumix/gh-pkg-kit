package migrator

import (
	"context"
	"fmt"

	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
)

// DeleteMigratedVersions deletes the specified package versions and returns failure messages.
func DeleteMigratedVersions(ctx context.Context, client *gh.GitHubClient, ownerType gh.OwnerType, owner, pkgType, pkgName string, versionIDs []int64) []string {
	var failures []string
	for _, vID := range versionIDs {
		if err := gh.DeletePackageVersionByOwnerType(ctx, client, ownerType, owner, pkgType, pkgName, vID); err != nil {
			logger.Error("Failed to delete source version", "version_id", vID, "error", err)
			failures = append(failures, fmt.Sprintf("version %d: %v", vID, err))
		} else {
			logger.Info("Deleted source version", "version_id", vID)
		}
	}
	return failures
}
