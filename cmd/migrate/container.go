package migrate

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/auth"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

// NewContainerCmd creates a command to migrate container packages between owners
func NewContainerCmd() *cobra.Command {
	var (
		from       string
		to         string
		deleteFlag bool
		dryRun     bool
		versionIDs []int64
		latest     int
		since      string
		until      string
	)

	cmd := &cobra.Command{
		Use:   "container <package-name>",
		Short: "Migrate container packages between owners",
		Long: `Migrates container (OCI/Docker) packages from one owner to another within GitHub Packages.
Uses the OCI Distribution API to copy image manifests and blobs, including multi-architecture images.
The source owner is resolved from the current repository if --from is not specified.
The source and destination owner types (organization or user) are detected automatically.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			srcPackage := args[0]

			repo, err := parser.Repository(parser.RepositoryOwnerWithHost(from))
			if err != nil {
				return fmt.Errorf("failed to resolve source owner: %w", err)
			}
			srcOwner := repo.Owner

			dest, err := parser.ParsePackageRef(to, srcPackage)
			if err != nil {
				return err
			}

			// Default dest host to source host
			destHost := dest.Host
			if destHost == "" {
				destHost = repo.Host
			}

			srcClient, destClient, err := gh.NewGitHubClientWith2Hosts(repo.Host, destHost)
			if err != nil {
				return err
			}
			ctx := cmd.Context()

			return migrateContainer(ctx, srcClient, destClient, migrateContainerOptions{
				packageType: "container",
				srcHost:     repo.Host,
				destHost:    destHost,
				srcOwner:    srcOwner,
				srcPackage:  srcPackage,
				destOwner:   dest.Owner,
				destPackage: dest.Package,
				deleteFlag:  deleteFlag,
				dryRun:      dryRun,
				versionIDs:  versionIDs,
				latest:      latest,
				since:       since,
				until:       until,
			})
		},
	}

	f := cmd.Flags()
	f.StringVar(&from, "from", "", "Source [host/]owner (default: current repository owner)")
	f.StringVar(&to, "to", "", "Destination [host/]owner[/package-name] (host detected by '.' in first segment)")
	_ = cmd.MarkFlagRequired("to")
	f.BoolVar(&deleteFlag, "delete", false, "Delete source versions after successful migration")
	f.BoolVar(&dryRun, "dry-run", false, "Show what would be migrated without performing the migration")
	f.Int64SliceVar(&versionIDs, "version", nil, "Migrate specific version(s) by ID (can be specified multiple times)")
	f.IntVarP(&latest, "latest", "n", 0, "Migrate latest N versions (by creation date)")
	f.StringVar(&since, "since", "", "Migrate versions created on or after this date (RFC3339 or YYYY-MM-DD)")
	f.StringVar(&until, "until", "", "Migrate versions created on or before this date (RFC3339 or YYYY-MM-DD)")

	return cmd
}

type migrateContainerOptions struct {
	packageType string
	srcHost     string
	destHost    string
	srcOwner    string
	srcPackage  string
	destOwner   string
	destPackage string
	deleteFlag  bool
	dryRun      bool
	versionIDs  []int64
	latest      int
	since       string
	until       string
}

func migrateContainer(ctx context.Context, srcClient *gh.GitHubClient, destClient *gh.GitHubClient, opts migrateContainerOptions) error {
	// Detect source owner type
	srcOwnerType, err := gh.DetectOwnerType(ctx, srcClient, opts.srcOwner)
	if err != nil {
		return err
	}

	// List source versions
	versions, err := gh.ListPackageVersionsByOwnerType(ctx, srcClient, srcOwnerType, opts.srcOwner, opts.packageType, opts.srcPackage)
	if err != nil {
		return err
	}

	// Apply version filters
	filter, err := buildVersionFilter(opts.versionIDs, opts.latest, opts.since, opts.until)
	if err != nil {
		return err
	}
	versions = gh.FilterVersions(versions, filter)

	if len(versions) == 0 {
		logger.Info("No versions to migrate")
		return nil
	}

	// OCI image references must use lowercase path components per the OCI Distribution Spec.
	srcBase := gh.ContainerImageBase(opts.srcHost, opts.srcOwner, opts.srcPackage)
	destBase := gh.ContainerImageBase(opts.destHost, opts.destOwner, opts.destPackage)

	if opts.dryRun {
		srcImage := srcBase
		destImage := destBase
		logger.Info("Dry run: migration plan", "src", srcImage, "dest", destImage, "versions", len(versions))
		r := render.NewRenderer(nil)
		r.RenderPackageVersions(versions, nil)
		return nil
	}

	// Get auth keychain for container registries
	keychain, err := registryKeychain(ctx, opts.srcHost, srcClient, opts.destHost, destClient)
	if err != nil {
		return fmt.Errorf("failed to get authentication: %w", err)
	}
	craneAuth := crane.WithAuthFromKeychain(keychain)

	// Migrate each version
	var migrated []int64
	var failures []string
	for _, v := range versions {
		tags := gh.GetVersionTags(v)
		if len(tags) == 0 {
			// Use digest-based reference for untagged versions
			digest := gh.GetVersionDigest(v)
			if digest == "" {
				logger.Warn("Skipping version with no tags and no digest", "version_id", v.GetID(), "name", v.GetName())
				failures = append(failures, fmt.Sprintf("version %d (%s): no tags or digest", v.GetID(), v.GetName()))
				continue
			}
			srcRef := srcBase + "@" + digest
			destRef := destBase + "@" + digest
			logger.Info("Migrating image by digest", "src", srcRef, "dest", destRef)
			if err := crane.Copy(srcRef, destRef, craneAuth); err != nil {
				err = withPackageAuthHint(err, opts.srcHost, opts.destHost)
				logger.Error("Failed to migrate image", "src", srcRef, "error", err)
				failures = append(failures, fmt.Sprintf("version %d (%s): %v", v.GetID(), v.GetName(), err))
				continue
			}
			migrated = append(migrated, v.GetID())
			continue
		}
		tagFailed := false
		for _, tag := range tags {
			srcRef := srcBase + ":" + tag
			destRef := destBase + ":" + tag
			logger.Info("Migrating image", "src", srcRef, "dest", destRef)
			if err := crane.Copy(srcRef, destRef, craneAuth); err != nil {
				err = withPackageAuthHint(err, opts.srcHost, opts.destHost)
				logger.Error("Failed to migrate image", "src", srcRef, "error", err)
				failures = append(failures, fmt.Sprintf("version %d tag %s: %v", v.GetID(), tag, err))
				tagFailed = true
			}
		}
		if !tagFailed {
			migrated = append(migrated, v.GetID())
		}
	}

	// Delete migrated versions if requested
	if opts.deleteFlag && len(migrated) > 0 {
		for _, vID := range migrated {
			if err := gh.DeletePackageVersionByOwnerType(ctx, srcClient, srcOwnerType, opts.srcOwner, opts.packageType, opts.srcPackage, vID); err != nil {
				logger.Error("Failed to delete source version", "version_id", vID, "error", err)
			} else {
				logger.Info("Deleted source version", "version_id", vID)
			}
		}
	}

	// Report
	logger.Info("Migration complete", "migrated", len(migrated), "failed", len(failures))
	if len(failures) > 0 {
		return fmt.Errorf("some versions failed to migrate: %s", strings.Join(failures, "; "))
	}
	return nil
}

// withPackageAuthHint wraps DENIED/UNAUTHORIZED container registry errors with a hint.
// GHES container registries require a classic PAT with read:packages/write:packages scope.
// OAuth App tokens (from `gh auth login --web`) may be rejected even with write:packages scope.
// Set the token via .env or environment variable:
//   - github.com:  GH_TOKEN=<classic-PAT>
//   - GHES:        GH_ENTERPRISE_TOKEN=<classic-PAT>
func withPackageAuthHint(err error, srcHost, destHost string) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if !strings.Contains(msg, "DENIED") && !strings.Contains(msg, "UNAUTHORIZED") {
		return err
	}
	hosts := []string{srcHost}
	if destHost != srcHost {
		hosts = append(hosts, destHost)
	}
	var hints []string
	for _, h := range hosts {
		envVar := "GH_ENTERPRISE_TOKEN"
		if h == "github.com" {
			envVar = "GH_TOKEN"
		}
		hints = append(hints, fmt.Sprintf("  %s=<classic-PAT>  # host: %s", envVar, h))
	}
	return fmt.Errorf("%w\nhint: container registry auth requires a classic PAT with read:packages/write:packages scope.\nOAuth App tokens may be rejected by GHES container registries.\nSet in .env or environment:\n%s", err, strings.Join(hints, "\n"))
}

// buildVersionFilter creates a VersionFilter from flag values.
func buildVersionFilter(versionIDs []int64, latest int, since, until string) (gh.VersionFilter, error) {
	filter := gh.VersionFilter{
		VersionIDs: versionIDs,
		Latest:     latest,
	}
	if since != "" {
		t, err := parseDate(since)
		if err != nil {
			return filter, fmt.Errorf("invalid --since value '%s': %w", since, err)
		}
		filter.Since = &t
	}
	if until != "" {
		t, err := parseDate(until)
		if err != nil {
			return filter, fmt.Errorf("invalid --until value '%s': %w", until, err)
		}
		filter.Until = &t
	}
	return filter, nil
}

// parseDate parses a date string in RFC3339 or YYYY-MM-DD format.
func parseDate(s string) (time.Time, error) {
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

// ghKeychain implements authn.Keychain, resolving credentials per container registry.
type ghKeychain struct {
	// registryToHost maps container registry host to GitHub host.
	registryToHost map[string]string
	// registryToLogin maps container registry host to GitHub login.
	registryToLogin map[string]string
}

func (k *ghKeychain) Resolve(target authn.Resource) (authn.Authenticator, error) {
	githubHost, ok := k.registryToHost[target.RegistryStr()]
	if !ok {
		return authn.Anonymous, nil
	}
	token, _ := auth.TokenForHost(githubHost)
	if token == "" {
		return nil, fmt.Errorf("no GitHub token found for host '%s'; run 'gh auth login' first", githubHost)
	}
	login := k.registryToLogin[target.RegistryStr()]
	return authn.FromConfig(authn.AuthConfig{
		Username: login,
		Password: token,
	}), nil
}

// registryKeychain creates a keychain that resolves credentials for the source and destination registries.
func registryKeychain(ctx context.Context, srcHost string, srcG *gh.GitHubClient, destHost string, destG *gh.GitHubClient) (authn.Keychain, error) {
	srcToken, _ := auth.TokenForHost(srcHost)
	if srcToken == "" {
		return nil, fmt.Errorf("no GitHub token found for host '%s'; run 'gh auth login' first", srcHost)
	}
	srcUser, err := gh.GetLoginUser(ctx, srcG)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub username for host '%s': %w", srcHost, err)
	}
	kc := &ghKeychain{
		registryToHost:  map[string]string{gh.ContainerRegistry(srcHost): srcHost},
		registryToLogin: map[string]string{gh.ContainerRegistry(srcHost): srcUser.GetLogin()},
	}
	if srcHost != destHost {
		destToken, _ := auth.TokenForHost(destHost)
		if destToken == "" {
			return nil, fmt.Errorf("no GitHub token found for host '%s'; run 'gh auth login' first", destHost)
		}
		destUser, err := gh.GetLoginUser(ctx, destG)
		if err != nil {
			return nil, fmt.Errorf("failed to get GitHub username for host '%s': %w", destHost, err)
		}
		kc.registryToHost[gh.ContainerRegistry(destHost)] = destHost
		kc.registryToLogin[gh.ContainerRegistry(destHost)] = destUser.GetLogin()
	}
	return kc, nil
}
