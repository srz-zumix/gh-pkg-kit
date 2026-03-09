# migrate command specification

## Overview

The `migrate` command copies GitHub Packages from one owner (org or user) to another within GitHub Packages.
Each package type has its own subcommand under `migrate`, because the migration method differs by package type.

The source/destination owner type (organization or user) is determined internally via the GitHub Users API (`User.Type`).

## Command Structure

```sh
gh pkg-kit migrate <package-type> <package-name> --to <dest-owner>[/<dest-package-name>] [--from <source-owner>] [flags]
```

- `<package-type>`: One of `container`, `npm`, `maven`, `rubygems`, `nuget`, `docker`
- `<package-name>`: Source package name
- `--from`: Source [host/]owner. If omitted, resolved from the current repository owner.
- `--to`: Destination [host/]owner[/package-name]. If package name is omitted, the source package name is used. Host is detected by '.' in the first segment.

### Subcommands

```sh
gh pkg-kit migrate container ...
gh pkg-kit migrate npm ...
gh pkg-kit migrate maven ...
gh pkg-kit migrate rubygems ...
gh pkg-kit migrate nuget ...
gh pkg-kit migrate docker ...
```

## Common Flags

| Flag | Short | Description | Required | Default |
| ---- | ----- | ----------- | -------- | ------- |
| `--delete` | | Delete source versions after successful migration | No | `false` |
| `--dry-run` | | Show what would be migrated without performing the migration | No | `false` |
| `--from` | | Source [host/]owner | No | Current repository owner |
| `--latest` | `-n` | Migrate latest N versions (by creation date) | No | |
| `--rewrite-labels` | | Rewrite OCI image config labels to reflect the destination owner/host (container/docker only, changes image digest) | No | `false` |
| `--since` | | Migrate versions created on or after this date (RFC3339 or YYYY-MM-DD) | No | |
| `--to` | | Destination [host/]owner[/package-name] | Yes | |
| `--until` | | Migrate versions created on or before this date (RFC3339 or YYYY-MM-DD) | No | |
| `--version` | | Migrate specific version(s) by ID (can be specified multiple times) | No | All versions |

### Version Selection

By default, all versions of the package are migrated.
Version selection flags can be combined to narrow down the set:

- `--version <id>`: Direct version ID specification. Can be repeated to select multiple specific versions.
- `--latest <N>`: Selects the most recent N versions ordered by creation date (descending).
- `--since <date>`: Filters to versions created on or after the given date.
- `--until <date>`: Filters to versions created on or before the given date.

When multiple flags are specified, they are applied as intersection (AND).
For example, `--latest 10 --since 2025-01-01` selects the latest 10 versions that were created on or after 2025-01-01.

## Owner Type Detection

The command determines whether each owner is an organization or a user by calling the GitHub Users API:

```text
GET /users/{owner}
```

- `User.Type == "Organization"` → use Organizations Packages API
- `User.Type == "User"` → use Users Packages API

This is done for both source and destination owners so the correct API endpoints are used automatically.

## Migration Strategy by Package Type

### container / docker

Container images (OCI/Docker) are stored in `ghcr.io` (or `containers.<host>` for GHES).
Migration copies image manifests and blobs between registries.

- **Source**: `ghcr.io/<source-owner>/<package-name>:<tag>` (or `@<digest>` for untagged versions)
- **Destination**: `ghcr.io/<dest-owner>/<dest-package-name>:<tag>` (or `@<digest>`)
- **Method**: `crane copy` ([google/go-containerregistry](https://github.com/google/go-containerregistry))
- **Version mapping**: Each `PackageVersion` corresponds to one or more tags or a digest. Tagged versions are copied per-tag; untagged versions are copied by digest reference.
- **Authentication**: Uses a `ghKeychain` that resolves credentials per container registry host via `gh auth token`. Supports cross-host migration (e.g., github.com ↔ GHES).
- **Note**: Multi-architecture images (manifest lists/OCI index) are handled correctly by `crane copy` — all referenced manifests and blobs are copied.
- **Label rewriting** (`--rewrite-labels`): When enabled, OCI image config labels containing the source owner URL (e.g. `org.opencontainers.image.source`, `org.opencontainers.image.url`) are rewritten to reflect the destination owner/host. This uses `mutate.ConfigFile` from go-containerregistry instead of `crane copy`, which changes the image digest. For multi-arch images, each platform image's labels are rewritten and the index is rebuilt. Label rewriting is only applied to tagged versions; untagged (digest-only) versions are copied as-is.
- **GHES Note**: Container registries on GHES require a classic PAT with `read:packages`/`write:packages` scope. OAuth App tokens may be rejected. Set `GH_ENTERPRISE_TOKEN` in `.env` or environment.

### npm

npm packages are stored in the GitHub npm registry (`npm.pkg.github.com`).

- **Source**: `@<source-owner>/<package-name>` from `https://npm.pkg.github.com`
- **Destination**: `@<dest-owner>/<dest-package-name>` to `https://npm.pkg.github.com`
- **Method**: TBD (candidates: download tarball via npm registry API, re-publish with `npm publish`, or direct registry HTTP API)
- **Version mapping**: Each `PackageVersion.Name` is the semver version string.
- **Considerations**:
  - Scoped package name changes from `@source-owner` to `@dest-owner`
  - `package.json` inside the tarball may need modification (name field)

### maven

Maven packages are stored in the GitHub Maven registry (`maven.pkg.github.com`).

- **Source**: `https://maven.pkg.github.com/<source-owner>/<repo>`
- **Destination**: `https://maven.pkg.github.com/<dest-owner>/<repo>`
- **Method**: TBD (download artifacts via Maven repository API, upload to destination)
- **Version mapping**: Each `PackageVersion.Name` is the Maven version string.
- **Considerations**:
  - Maven packages are tied to a repository. Migration may need a target repository specification.
  - Artifacts include JAR, POM, and potentially sources/javadoc JARs.
  - POM content (groupId, etc.) may need modification.

### rubygems

Ruby gems are stored in the GitHub RubyGems registry (`rubygems.pkg.github.com`).

- **Source**: `https://rubygems.pkg.github.com/<source-owner>`
- **Destination**: `https://rubygems.pkg.github.com/<dest-owner>`
- **Method**: TBD (download .gem file, push to destination)
- **Version mapping**: Each `PackageVersion.Name` is the gem version string.
- **Considerations**:
  - Gem metadata (gemspec) inside the .gem may reference the source owner.

### nuget

NuGet packages are stored in the GitHub NuGet registry (`nuget.pkg.github.com`).

- **Source**: `https://nuget.pkg.github.com/<source-owner>/index.json`
- **Destination**: `https://nuget.pkg.github.com/<dest-owner>/index.json`
- **Method**: TBD (download .nupkg, push to destination)
- **Version mapping**: Each `PackageVersion.Name` is the NuGet version string.
- **Considerations**:
  - .nuspec inside .nupkg may reference source-specific metadata.

## Execution Flow

1. **Parse arguments**: Extract source owner, package name, destination owner, optional destination package name.
2. **Detect owner types**: Call Users API for both source and destination owners to determine org/user type.
3. **List source versions**: Fetch all versions of the source package using the appropriate API (org or user).
4. **Apply version filters**: Filter versions by `--version`, `--latest`, `--since`, `--until` flags.
5. **Dry-run check**: If `--dry-run`, display the list of versions that would be migrated and exit.
6. **Migrate versions**: For each selected version, perform the package-type-specific migration.
7. **Delete source** (optional): If `--delete` is set, delete the migrated versions (or entire package if all versions were migrated) from the source.
8. **Report**: Output a summary of migrated versions (success/failure counts).

## Error Handling

- If the destination package already has a version with the same name/tag, the behavior should be:
  - Skip with a warning (default)
  - TBD: `--force` flag to overwrite
- If any version migration fails, continue with remaining versions and report failures at the end.
- The `--delete` operation only deletes versions that were successfully migrated.

## Directory/File Structure

```text
cmd/
  migrate.go              # Parent 'migrate' command, registers subcommands
  migrate/
    container.go          # migrate container/docker subcommand (shared via newContainerMigrateCmd)
    docker.go             # migrate docker subcommand (delegates to newContainerMigrateCmd)
    npm.go                # migrate npm subcommand (planned)
    maven.go              # migrate maven subcommand (planned)
    rubygems.go           # migrate rubygems subcommand (planned)
    nuget.go              # migrate nuget subcommand (planned)

pkg/
  packages/
    container.go          # Container/docker migration logic (MigrateContainer, auth, version filter)

go-gh-extension/pkg/gh/
  migrate.go              # Common migration wrapper functions (owner type detection, version filtering)

go-gh-extension/pkg/gh/client/
  (existing packages.go)  # Reuse existing package/version listing APIs
```

## Open Questions

- [x] Which external tool to use for container migration? → `crane` (google/go-containerregistry)
- [ ] For npm, is re-publishing with modified `package.json` necessary, or can the tarball be published as-is?
- [ ] For maven, should a `--repo` flag be added to specify the destination repository?
- [ ] Should there be a `--force` flag to overwrite existing versions at the destination?
- [ ] Should `--concurrency` flag be supported for parallel version migration?
