package migrator

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/srz-zumix/go-gh-extension/pkg/gh"
)

// ListFilteredVersions detects the owner type, lists package versions, and applies version filters.
// Returns the filtered versions and the detected owner type (needed for delete operations).
func ListFilteredVersions(ctx context.Context, client *gh.GitHubClient, owner, packageType, packageName string, versionIDs []int64, latest int, since, until string) ([]*PackageVersion, gh.OwnerType, error) {
	ownerType, err := gh.DetectOwnerType(ctx, client, owner)
	if err != nil {
		return nil, ownerType, fmt.Errorf("failed to detect owner type: %w", err)
	}
	versions, err := gh.ListPackageVersionsByOwnerType(ctx, client, ownerType, owner, packageType, packageName)
	if err != nil {
		return nil, ownerType, fmt.Errorf("failed to list versions: %w", err)
	}
	filter, err := BuildVersionFilter(versionIDs, latest, since, until)
	if err != nil {
		return nil, ownerType, err
	}
	filtered := gh.FilterVersions(versions, filter)
	// Sort ascending by creation date so the newest version is pushed last,
	// ensuring it becomes "latest" in GitHub Packages.
	slices.SortStableFunc(filtered, func(a, b *PackageVersion) int {
		if a.CreatedAt == nil && b.CreatedAt == nil {
			return 0
		}
		if a.CreatedAt == nil {
			return 1
		}
		if b.CreatedAt == nil {
			return -1
		}
		return a.CreatedAt.Compare(b.CreatedAt.Time)
	})
	return filtered, ownerType, nil
}

// BuildVersionFilter creates a VersionFilter from flag values.
func BuildVersionFilter(versionIDs []int64, latest int, since, until string) (gh.VersionFilter, error) {
	filter := gh.VersionFilter{
		VersionIDs: versionIDs,
		Latest:     latest,
	}
	if since != "" {
		t, err := ParseDate(since)
		if err != nil {
			return filter, fmt.Errorf("invalid --since value '%s': %w", since, err)
		}
		filter.Since = &t
	}
	if until != "" {
		t, err := ParseDate(until)
		if err != nil {
			return filter, fmt.Errorf("invalid --until value '%s': %w", until, err)
		}
		filter.Until = &t
	}
	return filter, nil
}

// ParseDate parses a date string in RFC3339 or YYYY-MM-DD format.
func ParseDate(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return t, nil
	}
	t, err = time.Parse("2006-01-02", s)
	if err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("expected RFC3339 or YYYY-MM-DD format")
}
