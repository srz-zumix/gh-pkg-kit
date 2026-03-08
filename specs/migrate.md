# migrate command specification

## Overview

The `migrate` command copies GitHub Packages from one owner (org or user) to another within GitHub Packages.
Each package type has its own subcommand under `migrate`, because the migration method differs by package type.

The source/destination owner type (organization or user) is determined internally via the GitHub Users API (`User.Type`).

## Command Structure

```sh
gh pkg-kit migrate <package-type> <source-owner>/<package-name> --to <dest-owner>[/<dest-package-name>] [flags]
```

- `<package-type>`: One of `container`, `npm`, `maven`, `rubygems`, `nuget`, `docker`
- `<source-owner>/<package-name>`: Source package identifier
- `--to`: Destination owner (and optionally a different package name). If package name is omitted, the source package name is used.

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
| `--to` | | Destination owner[/package-name] | Yes | |
| `--delete` | | Delete source package/versions after successful migration | No | `false` |
| `--dry-run` | | Show what would be migrated without actually performing the migration | No | `false` |
| `--version` | | Migrate specific version(s) by ID (can be specified multiple times) | No | All versions |
| `--latest` | `-n` | Migrate latest N versions (by creation date) | No | |
| `--since` | | Migrate versions created on or after this date (RFC3339 or YYYY-MM-DD) | No | |
| `--until` | | Migrate versions created on or before this date (RFC3339 or YYYY-MM-DD) | No | |

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

Container images (OCI/Docker) are stored in `ghcr.io`.
Migration copies image manifests and blobs between registries.

- **Source**: `ghcr.io/<source-owner>/<package-name>:<tag>`
- **Destination**: `ghcr.io/<dest-owner>/<dest-package-name>:<tag>`
- **Method**: TBD (candidates: `crane copy`, `oras copy`, `skopeo copy`, or direct OCI Distribution API calls)
- **Version mapping**: Each `PackageVersion` corresponds to a tag or digest. The version's `Name` field is used as the tag.
- **Authentication**: Uses the GitHub token (`gh auth token`) for both source and destination `ghcr.io` registries.
- **Note**: Multi-architecture images (manifest lists/OCI index) must be handled correctly — all referenced manifests and blobs must be copied.

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

## Directory/File Structure (planned)

```text
cmd/
  migrate.go              # Parent 'migrate' command, registers subcommands
  migrate/
    container.go          # migrate container subcommand
    npm.go                # migrate npm subcommand
    maven.go              # migrate maven subcommand
    rubygems.go           # migrate rubygems subcommand
    nuget.go              # migrate nuget subcommand
    docker.go             # migrate docker subcommand

go-gh-extension/pkg/gh/
  migrate.go              # Common migration wrapper functions (owner type detection, version filtering)

go-gh-extension/pkg/gh/client/
  (existing packages.go)  # Reuse existing package/version listing APIs
```

## Open Questions

- [ ] Which external tool to use for container migration? (`crane` / `oras` / `skopeo` / direct API)
- [ ] For npm, is re-publishing with modified `package.json` necessary, or can the tarball be published as-is?
- [ ] For maven, should a `--repo` flag be added to specify the destination repository?
- [ ] Should there be a `--force` flag to overwrite existing versions at the destination?
- [ ] Should `--concurrency` flag be supported for parallel version migration?
