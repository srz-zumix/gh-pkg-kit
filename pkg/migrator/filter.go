package migrator

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/Masterminds/semver/v3"
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
	sortVersionsAscending(filtered)
	return filtered, ownerType, nil
}

// versionSemverCache is a lazily populated cache of parsed semver values,
// keyed by version name. It is used only when CreatedAt comparison is
// insufficient (both timestamps are equal or both are nil).
type versionSemverCache map[string]*semver.Version

// get returns the cached semver for name, parsing it on first access.
// The result is nil when the name is not valid semver.
func (c versionSemverCache) get(name string) *semver.Version {
	sv, ok := c[name]
	if !ok {
		sv, _ = semver.NewVersion(name)
		c[name] = sv
	}
	return sv
}

// sortVersionsAscending sorts versions in-place, oldest first, so that the
// newest version is pushed last and becomes "latest" in GitHub Packages.
// Sort order:
//  1. CreatedAt ascending; nil CreatedAt is treated as oldest (sort to front).
//  2. Semantic version ascending (fallback when timestamps are equal or both nil).
//  3. Name string ascending (final deterministic fallback).
func sortVersionsAscending(versions []*PackageVersion) {
	// Semver cache is populated lazily: only entries whose CreatedAt comparison
	// is inconclusive (equal or both nil) are ever parsed.
	cache := make(versionSemverCache)

	slices.SortStableFunc(versions, func(a, b *PackageVersion) int {
		aHasTime := a.CreatedAt != nil
		bHasTime := b.CreatedAt != nil
		if aHasTime && bHasTime {
			if cmp := a.CreatedAt.Compare(b.CreatedAt.Time); cmp != 0 {
				return cmp
			}
		} else if aHasTime {
			// b has no timestamp → treat b as older, so b comes first
			return 1
		} else if bHasTime {
			// a has no timestamp → treat a as older, so a comes first
			return -1
		}
		// Fallback: compare by semantic version (parsed and cached on demand)
		aSV := cache.get(a.GetName())
		bSV := cache.get(b.GetName())
		if aSV != nil && bSV != nil {
			if cmp := aSV.Compare(bSV); cmp != 0 {
				return cmp
			}
		} else if aSV != nil {
			return 1 // a is valid semver, b is not → a is newer
		} else if bSV != nil {
			return -1 // b is valid semver, a is not → b is newer
		}
		// Final fallback: lexicographic order by name
		if a.GetName() < b.GetName() {
			return -1
		} else if a.GetName() > b.GetName() {
			return 1
		}
		return 0
	})
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
