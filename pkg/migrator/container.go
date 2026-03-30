package migrator

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2/pkg/auth"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

// ContainerOptions holds the options for migrating container/docker packages.
type ContainerOptions struct {
	PackageType   string
	Src           repository.Repository
	SrcPackage    string
	Dest          repository.Repository
	DestPackage   string
	DeleteFlag    bool
	DryRun        bool
	RewriteLabels bool
	VersionIDs    []int64
	Latest        int
	Since         string
	Until         string
}

// MigrateContainer migrates container/docker packages between owners.
func MigrateContainer(ctx context.Context, srcClient *gh.GitHubClient, destClient *gh.GitHubClient, opts ContainerOptions) error {
	versions, srcOwnerType, err := ListFilteredVersions(ctx, srcClient, opts.Src.Owner, opts.PackageType, opts.SrcPackage, opts.VersionIDs, opts.Latest, opts.Since, opts.Until)
	if err != nil {
		return err
	}

	if len(versions) == 0 {
		logger.Info("No versions to migrate")
		return nil
	}

	// OCI image references must use lowercase path components per the OCI Distribution Spec.
	srcBase := gh.ContainerImageBase(opts.Src, opts.SrcPackage)
	dstBase := gh.ContainerImageBase(opts.Dest, opts.DestPackage)

	if opts.DryRun {
		srcImage := srcBase
		dstImage := dstBase
		logger.Info("Dry run: migration plan", "src", srcImage, "dest", dstImage, "versions", len(versions))
		r := render.NewRenderer(nil)
		r.RenderPackageVersions(versions, nil)
		return nil
	}

	// Get auth keychain for container registries
	keychain, err := registryKeychain(ctx, opts.Src.Host, srcClient, opts.Dest.Host, destClient)
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
			dstRef := dstBase + "@" + digest
			logger.Info("Migrating image by digest", "src", srcRef, "dest", dstRef)
			if err := crane.Copy(srcRef, dstRef, craneAuth); err != nil {
				err = withPackageAuthHint(err, opts.Src.Host, opts.Dest.Host)
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
			dstRef := dstBase + ":" + tag
			logger.Info("Migrating image", "src", srcRef, "dest", dstRef)
			if err := copyImage(srcRef, dstRef, craneAuth, opts); err != nil {
				err = withPackageAuthHint(err, opts.Src.Host, opts.Dest.Host)
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
	var deleteFailures []string
	if opts.DeleteFlag && len(migrated) > 0 {
		for _, vID := range migrated {
			if err := gh.DeletePackageVersionByOwnerType(ctx, srcClient, srcOwnerType, opts.Src.Owner, opts.PackageType, opts.SrcPackage, vID); err != nil {
				logger.Error("Failed to delete source version", "version_id", vID, "error", err)
				deleteFailures = append(deleteFailures, fmt.Sprintf("version %d: %v", vID, err))
			} else {
				logger.Info("Deleted source version", "version_id", vID)
			}
		}
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
}

// copyImage copies an image from srcRef to destRef.
// When opts.RewriteLabels is true, OCI annotation labels (e.g. org.opencontainers.image.source)
// are rewritten to reflect the destination owner/host before pushing.
func copyImage(srcRef, destRef string, craneAuth crane.Option, opts ContainerOptions) error {
	if !opts.RewriteLabels {
		return crane.Copy(srcRef, destRef, craneAuth)
	}
	return copyAndRewriteLabels(srcRef, destRef, opts, craneAuth)
}

// copyAndRewriteLabels pulls an image/index from srcRef, rewrites OCI config labels,
// and pushes the result to destRef.
func copyAndRewriteLabels(srcRef, destRef string, opts ContainerOptions, craneAuth crane.Option) error {
	src, err := name.ParseReference(srcRef)
	if err != nil {
		return fmt.Errorf("failed to parse source reference: %w", err)
	}
	dest, err := name.ParseReference(destRef)
	if err != nil {
		return fmt.Errorf("failed to parse destination reference: %w", err)
	}

	o := crane.GetOptions(craneAuth)
	remoteOpts := o.Remote

	desc, err := remote.Get(src, remoteOpts...)
	if err != nil {
		return fmt.Errorf("failed to get image descriptor: %w", err)
	}

	switch desc.MediaType {
	case types.OCIImageIndex, types.DockerManifestList:
		idx, err := desc.ImageIndex()
		if err != nil {
			return err
		}
		modified, err := rewriteIndexLabels(idx, opts.Src.Host, opts.Src.Owner, opts.Dest.Host, opts.Dest.Owner)
		if err != nil {
			return err
		}
		return remote.WriteIndex(dest, modified, remoteOpts...)
	default:
		img, err := desc.Image()
		if err != nil {
			return err
		}
		modified, err := rewriteImageLabels(img, opts.Src.Host, opts.Src.Owner, opts.Dest.Host, opts.Dest.Owner)
		if err != nil {
			return err
		}
		return remote.Write(dest, modified, remoteOpts...)
	}
}

// rewriteImageLabels rewrites OCI annotation labels on an image config
// to reflect the new owner/host when migrating.
func rewriteImageLabels(img v1.Image, srcHost, srcOwner, destHost, destOwner string) (v1.Image, error) {
	cfg, err := img.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("failed to get image config: %w", err)
	}
	if cfg == nil {
		return img, nil
	}
	labels := cfg.Config.Labels
	if len(labels) == 0 {
		return img, nil
	}

	oldPrefix := fmt.Sprintf("https://%s/%s", srcHost, srcOwner)
	newPrefix := fmt.Sprintf("https://%s/%s", destHost, destOwner)

	modified := false
	newLabels := make(map[string]string, len(labels))
	for k, v := range labels {
		newV := v
		if strings.Contains(v, oldPrefix) {
			newV = strings.ReplaceAll(v, oldPrefix, newPrefix)
		}
		if newV != v {
			modified = true
			logger.Info("Rewriting label", "key", k, "old", v, "new", newV)
		}
		newLabels[k] = newV
	}

	if !modified {
		return img, nil
	}

	cfg.Config.Labels = newLabels
	return mutate.ConfigFile(img, cfg)
}

// rewriteIndexLabels rewrites OCI annotation labels on all platform images in an image index.
func rewriteIndexLabels(idx v1.ImageIndex, srcHost, srcOwner, destHost, destOwner string) (v1.ImageIndex, error) {
	manifest, err := idx.IndexManifest()
	if err != nil {
		return nil, fmt.Errorf("failed to get index manifest: %w", err)
	}

	var adds []mutate.IndexAddendum
	for _, desc := range manifest.Manifests {
		img, err := idx.Image(desc.Digest)
		if err != nil {
			return nil, fmt.Errorf("failed to get image for digest %s: %w", desc.Digest, err)
		}
		rewritten, err := rewriteImageLabels(img, srcHost, srcOwner, destHost, destOwner)
		if err != nil {
			return nil, fmt.Errorf("failed to rewrite labels for digest %s: %w", desc.Digest, err)
		}
		adds = append(adds, mutate.IndexAddendum{
			Add: rewritten,
			Descriptor: v1.Descriptor{
				Platform:    desc.Platform,
				Annotations: desc.Annotations,
			},
		})
	}

	return mutate.AppendManifests(empty.Index, adds...), nil
}

// PullContainerOptions holds options for pulling a container image to a local file.
type PullContainerOptions struct {
	PackageType string
	Src         repository.Repository
	SrcPackage  string
	Tag         string
	Output      string
	DryRun      bool
}

// PullContainerToFile pulls a container image tag from GitHub Packages and saves it as a Docker-loadable tarball.
func PullContainerToFile(ctx context.Context, client *gh.GitHubClient, opts PullContainerOptions) error {
	tag := opts.Tag
	if tag == "" {
		tag = "latest"
	}
	imageBase := gh.ContainerImageBase(opts.Src, opts.SrcPackage)
	imageRef := imageBase + ":" + tag

	output := opts.Output
	if output == "" {
		// Use only the base name of the package to avoid creating unexpected subdirectories.
		pkgBase := filepath.Base(opts.SrcPackage)
		output = fmt.Sprintf("%s-%s.tar", pkgBase, tag)
	}

	if opts.DryRun {
		logger.Info("Dry run: would pull image", "src", imageRef, "output", output)
		return nil
	}

	keychain, err := registryKeychain(ctx, opts.Src.Host, client, opts.Src.Host, client)
	if err != nil {
		return fmt.Errorf("failed to get authentication: %w", err)
	}
	craneAuth := crane.WithAuthFromKeychain(keychain)

	logger.Info("Pulling image", "src", imageRef)
	img, err := crane.Pull(imageRef, craneAuth)
	if err != nil {
		err = withPackageAuthHint(err, opts.Src.Host, opts.Src.Host)
		return fmt.Errorf("failed to pull image '%s': %w", imageRef, err)
	}

	logger.Info("Saving image", "output", output)
	if err := crane.SaveLegacy(img, imageRef, output); err != nil {
		return fmt.Errorf("failed to save image to '%s': %w", output, err)
	}

	logger.Info("Pulled image", "src", imageRef, "to", output)
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
