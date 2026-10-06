---
name: gh-pkg-kit
description: gh-pkg-kit is a GitHub CLI extension for working with GitHub Packages. Use it to download package assets (container, docker, gem, maven, npm, nuget), configure a NuGet credential provider backed by gh auth, migrate packages between owners/registries (including legacy docker.pkg.github.com → ghcr.io), and manage packages/versions for organizations and users (list/get/delete/restore) — all directly from the command line via `gh pkg-kit`.
---

# gh-pkg-kit

A GitHub CLI extension (`gh pkg-kit`) for package-related operations on GitHub Packages.
It supports downloading package assets, migrating packages between owners or registries, and managing packages and their versions for organizations and users.

## Prerequisites

### Installation

```sh
gh extension install srz-zumix/gh-pkg-kit
```

### Authentication

`gh pkg-kit` uses the `gh` CLI's authentication. Ensure you are authenticated before using the extension:

```sh
gh auth login
gh auth status
```

Container registries (`ghcr.io` / `containers.<host>`) require a classic PAT with `read:packages` / `write:packages` scope. Fine-grained tokens and OAuth App tokens may be rejected.

Tokens can also be provided via environment variables or a `.env` file in the current directory (loaded automatically):

| Variable | Description |
| -------- | ----------- |
| `GH_TOKEN` | Token for github.com (priority: `GH_TOKEN` > `GITHUB_TOKEN` > `gh auth` config) |
| `GH_ENTERPRISE_TOKEN` | Token for GitHub Enterprise Server |
| `GH_SRC_TOKEN` | Source token for `migrate` commands (fallback for `--src-token`) |
| `GH_DST_TOKEN` | Destination token for `migrate` commands (fallback for `--dst-token`) |

## CLI Structure

```
gh pkg-kit                         # Root command
├── container                      # GitHub Container Registry (ghcr.io)
│   └── pull                       # Pull container image as Docker tarball
├── docker                         # Legacy docker.pkg.github.com
│   └── pull                       # Pull docker image as Docker tarball
├── gem                            # RubyGems registry
│   └── download                   # Download .gem file
├── maven                          # Maven registry
│   └── download                   # Download .pom / .jar files
├── npm                            # npm registry
│   └── download                   # Download .tgz tarball
├── nuget                          # NuGet registry
│   ├── credential-provider        # NuGet authentication using gh auth tokens
│   │   ├── install                # Publish the .NET provider into the netcore plugin root
│   │   ├── run                    # Run the installed .NET provider
│   │   └── uninstall              # Remove the managed provider directory
│   ├── download                   # Download .nupkg file
│   └── tool-restore               # Run `dotnet tool restore` with a temporary credential provider
├── migrate                        # Migrate packages between owners/registries
│   ├── container                  # Migrate container packages
│   ├── docker                     # Migrate legacy docker → ghcr.io
│   ├── gem                        # Migrate RubyGems packages
│   ├── maven                      # Migrate Maven packages
│   ├── npm                        # Migrate npm packages
│   └── nuget                      # Migrate NuGet packages
├── org                            # Organization packages
│   ├── list                       # List packages for an organization
│   ├── get                        # Get a package in an organization
│   ├── delete                     # Delete a package in an organization
│   ├── restore                    # Restore a deleted package in an organization
│   └── versions                   # Package versions for an organization
│       ├── list
│       ├── get
│       ├── delete
│       └── restore
└── user                           # User packages
    ├── list                       # List packages for a user
    ├── get                        # Get a package for a user
    ├── delete                     # Delete a package for a user
    ├── restore                    # Restore a deleted package for a user
    └── versions                   # Package versions for a user
        ├── list
        ├── get
        ├── delete
        └── restore
```

## Global Flags

| Flag | Short | Default | Description |
| ---- | ----- | ------- | ----------- |
| `--help` | `-h` | | Help for gh-pkg-kit |
| `--log-level` | `-L` | `info` | Set log level: {debug\|info\|warn\|error} |
| `--read-only` | | `false` | Run in read-only mode (prevent write operations) |
| `--version` | `-v` | | Version for gh-pkg-kit |

## Container (gh pkg-kit container)

### Pull container image (gh pkg-kit container pull)

```sh
gh pkg-kit container pull <package-name> [--owner <owner>] [--tag <tag>] [--output <path>] [--dryrun] [--load] [--rm]
```

Pull a container image from `ghcr.io` and save it as a Docker-loadable tarball.
Load the saved file with `docker load -i <output-file>`.

```sh
# Pull the latest image for the current repository's owner
gh pkg-kit container pull my-image

# Pull a specific tag
gh pkg-kit container pull my-image --tag v1.2.3

# Specify owner and output file
gh pkg-kit container pull my-image --owner my-org --output ./my-image.tar

# Dry-run (show what would be pulled)
gh pkg-kit container pull my-image --dryrun

# Pull and load directly into the local Docker daemon
gh pkg-kit container pull my-image --load

# Pull, load into Docker daemon, and remove the local tarball afterwards
gh pkg-kit container pull my-image --load --rm
```

**Flags:**

| Flag | Short | Default | Description |
| ---- | ----- | ------- | ----------- |
| `--dryrun` | `-n` | `false` | Show what would be pulled without performing the pull |
| `--load` | | `false` | Load the pulled image into the local Docker daemon after saving |
| `--output` | | `<package-name>-<tag>.tar` | Output file path |
| `--owner` | `-o` | Current repository owner | `[host/]owner` |
| `--rm` | | `false` | Remove the local tarball after loading into Docker daemon (requires `--load`) |
| `--tag` | `-t` | `latest` | Image tag to pull |

## Docker (gh pkg-kit docker)

### Pull docker image (gh pkg-kit docker pull)

```sh
gh pkg-kit docker pull <package-name> [--owner <owner/repo>] [--tag <tag>] [--output <path>] [--dryrun] [--load] [--rm]
```

Pull a docker image from the legacy `docker.pkg.github.com` registry and save it as a Docker-loadable tarball.
`--owner` must include the repository name because the legacy path is `docker.pkg.github.com/OWNER/REPO/PACKAGE`.

```sh
# Pull from a specific owner/repo
gh pkg-kit docker pull my-image --owner my-org/my-repo

# Pull a specific tag
gh pkg-kit docker pull my-image --owner my-org/my-repo --tag v1.2.3

# Dry-run
gh pkg-kit docker pull my-image --owner my-org/my-repo --dryrun

# Pull and load directly into the local Docker daemon
gh pkg-kit docker pull my-image --owner my-org/my-repo --load

# Pull, load into Docker daemon, and remove the local tarball afterwards
gh pkg-kit docker pull my-image --owner my-org/my-repo --load --rm
```

**Flags:**

| Flag | Short | Default | Description |
| ---- | ----- | ------- | ----------- |
| `--dryrun` | `-n` | `false` | Show what would be pulled without performing the pull |
| `--load` | | `false` | Load the pulled image into the local Docker daemon after saving |
| `--output` | | `<package-name>-<tag>.tar` | Output file path |
| `--owner` | `-o` | Current repository | `[host/]owner/repo` (repository name is required) |
| `--rm` | | `false` | Remove the local tarball after loading into Docker daemon (requires `--load`) |
| `--tag` | `-t` | `latest` | Image tag to pull |

## Gem (gh pkg-kit gem)

### Download gem (gh pkg-kit gem download)

```sh
gh pkg-kit gem download <package-name> [flags]
```

Download a `.gem` file from the GitHub RubyGems registry.

```sh
# Download the latest version
gh pkg-kit gem download my-gem

# Download a specific version
gh pkg-kit gem download my-gem --version 1.2.3

# Specify owner and output file
gh pkg-kit gem download my-gem --owner my-org --output ./my-gem.gem
```

**Flags:**

| Flag | Short | Default | Description |
| ---- | ----- | ------- | ----------- |
| `--output` | | `<package-name>-<version>.gem` | Output file path |
| `--owner` | `-o` | Current repository owner | `[HOST/]OWNER` |
| `--version` | | Latest version | Package version to download |

## Maven (gh pkg-kit maven)

### Download maven artifact (gh pkg-kit maven download)

```sh
gh pkg-kit maven download <package-name> [flags]
```

Download `.pom` and `.jar` files from the GitHub Maven registry.
Accepts both colon-separated (`com.example:my-artifact`) and dot-separated (`com.example.my-artifact`) formats.
`--repo` must include the repository name.

```sh
# Download the latest version using the current repository
gh pkg-kit maven download com.example:my-artifact

# Specify repository and version
gh pkg-kit maven download com.example:my-artifact --repo my-org/my-repo --version 1.2.3

# Specify output directory
gh pkg-kit maven download com.example:my-artifact --output-dir ./artifacts
```

**Flags:**

| Flag | Short | Default | Description |
| ---- | ----- | ------- | ----------- |
| `--output-dir` | `-o` | Current directory | Output directory |
| `--repo` | `-R` | Current repository | Repository in `[host/]owner/repo` format |
| `--version` | | Latest version | Package version to download |

## npm (gh pkg-kit npm)

### Download npm tarball (gh pkg-kit npm download)

```sh
gh pkg-kit npm download <package-name> [flags]
```

Download a `.tgz` tarball from the GitHub npm registry.

```sh
# Download the latest version
gh pkg-kit npm download my-pkg

# Download a specific version
gh pkg-kit npm download my-pkg --version 1.2.3 --owner my-org
```

**Flags:**

| Flag | Short | Default | Description |
| ---- | ----- | ------- | ----------- |
| `--output` | | `<package-name>.<version>.tgz` | Output file path |
| `--owner` | `-o` | Current repository owner | `[HOST/]OWNER` |
| `--version` | | Latest version | Package version to download |

## NuGet (gh pkg-kit nuget)

### Install credential provider (gh pkg-kit nuget credential-provider install)

```sh
gh pkg-kit nuget credential-provider install [--dir <path>] [--force]
```

Publishes the embedded .NET source into `~/.nuget/plugins/netcore/CredentialProvider.GhPkgKit`.
Install requires .NET SDK 8+ and access to nuget.org; runtime use requires .NET 8+ and `gh` on PATH.
The provider rolls forward to newer .NET runtimes and uses `NuGet.Protocol` for protocol v2 and cancellation.
NuGet discovers the DLL by convention for project and tool restore. No PATH shim or token-bearing NuGet.Config is generated.
`--force` replaces only an existing managed installation. Unmanaged directories and symbolic links are preserved.

```sh
gh auth login --hostname github.com
gh pkg-kit nuget credential-provider install
dotnet tool restore
```

Installed entry point: `~/.nuget/plugins/netcore/CredentialProvider.GhPkgKit/CredentialProvider.GhPkgKit.dll`.
`NUGET_PLUGIN_PATHS` and `NUGET_NETCORE_PLUGIN_PATHS` override convention discovery.
When migrating from the old shim, remove its plugin-path override from the environment and shell startup files,
or replace the old path with the new DLL path while preserving other required plugins.
The installer leaves the obsolete `~/.dotnet/tools/nuget-plugin-gh-pkg-kit` shim and shell settings unchanged.
For a custom plugin root, explicitly configure the DLL path (this Unix example preserves explicit netcore plugins):

```sh
gh pkg-kit nuget credential-provider install --dir "$HOME/custom-nuget-plugins"
export NUGET_NETCORE_PLUGIN_PATHS="${NUGET_NETCORE_PLUGIN_PATHS:+$NUGET_NETCORE_PLUGIN_PATHS;}$HOME/custom-nuget-plugins/CredentialProvider.GhPkgKit/CredentialProvider.GhPkgKit.dll"
```

Include other conventionally discovered plugins when using an override. The default installation needs no override.
The provider calls `gh auth token --hostname <host>`; no interactive login occurs and `--interactive` is not required.
Only the netcore (.NET tooling) plugin location is supported, not netfx.

Credentials are provided only for HTTPS GitHub Packages NuGet URLs.
GHES hosts must be registered with `gh auth login --hostname <host>` or set explicitly via `GH_HOST`.
Tokens are read from gh configuration, secure storage, or the usual gh token environment variables.
Use a classic PAT with `read:packages` and access to the package; OAuth and fine-grained tokens may be rejected.
The provider does not perform interactive login or obtain a different token on authentication retries.

| Flag | Required | Default | Description |
| ---- | -------- | ------- | ----------- |
| `--dir` | No | `~/.nuget/plugins/netcore` | Plugin root; the provider subdirectory is created beneath it |
| `--force` | No | `false` | Replace an existing managed installation |

### Run credential provider (gh pkg-kit nuget credential-provider run)

```sh
gh pkg-kit nuget credential-provider run [-Plugin]
```

Runs the installed .NET provider from the default netcore root and forwards protocol streams with `-Plugin`.
NuGet normally launches the DLL directly.
Without the optional argument, prints an installation hint and exits. `--help` displays command help.
Protocol responses contain credentials; do not log them or run the protocol for manual token inspection.

### Uninstall credential provider (gh pkg-kit nuget credential-provider uninstall)

```sh
gh pkg-kit nuget credential-provider uninstall [--dir <path>]
```

Removes the managed `CredentialProvider.GhPkgKit` subdirectory from the default netcore root or optional `--dir` root.
An absent provider is ignored; directories without the generation marker and symbolic links are preserved.

| Flag | Required | Default | Description |
| ---- | -------- | ------- | ----------- |
| `--dir` | No | `~/.nuget/plugins/netcore` | Netcore plugin root |

### Download nuget package (gh pkg-kit nuget download)

```sh
gh pkg-kit nuget download <package-name> [flags]
```

Download a `.nupkg` file from the GitHub NuGet registry.

```sh
# Download the latest version
gh pkg-kit nuget download MyPackage

# Download a specific version
gh pkg-kit nuget download MyPackage --version 1.2.3 --owner my-org
```

**Flags:**

| Flag | Short | Default | Description |
| ---- | ----- | ------- | ----------- |
| `--output` | | `<package-name>.<version>.nupkg` | Output file path |
| `--owner` | `-o` | Current repository owner | `[HOST/]OWNER` |
| `--version` | | Latest version | Package version to download |

### Restore dotnet tools with a temporary credential provider (gh pkg-kit nuget tool-restore)

```sh
gh pkg-kit nuget tool-restore [--configfile <path>] [--work-dir <path>] [-- dotnet-tool-restore-args...]
```

Run `dotnet tool restore` with a temporary GitHub Packages credential provider backed by `gh auth`.
No permanent provider installation is required, and `NuGet.Config` is not modified.
Requires .NET SDK 8+ and access to nuget.org to build the provider on each invocation.
The provider is appended to the restore subprocess's `NUGET_PLUGIN_PATHS` and `NUGET_NETCORE_PLUGIN_PATHS`, preserving existing explicit paths and leaving the parent environment unchanged.
Temporary provider files are deleted on exit, including on failure and when `--work-dir` is specified.
Arguments after `--` are optional and passed through to `dotnet tool restore`. The former `--overwrite` flag is no longer supported.

```sh
# Restore tools using dotnet's default config discovery
gh pkg-kit nuget tool-restore

# Use a specific NuGet.Config file
gh pkg-kit nuget tool-restore --configfile ./NuGet.Config

# Build the temporary provider under a specific parent directory
gh pkg-kit nuget tool-restore --work-dir ./tmp

# Pass extra args through to `dotnet tool restore`
gh pkg-kit nuget tool-restore -- --verbosity minimal
```

**Flags:**

| Flag | Short | Default | Description |
| ---- | ----- | ------- | ----------- |
| `--configfile` | | dotnet's default config discovery | Optional path to the existing `NuGet.Config`, passed through unchanged |
| `--work-dir` | | System temp dir | Optional parent directory for temporary provider files (unique subdirectory deleted on exit) |

## Migrate (gh pkg-kit migrate)

The `migrate` subcommands copy package versions from a source owner to a destination owner within GitHub Packages.
`container` and `docker` use the OCI Distribution API to copy manifests/blobs (including multi-arch images).
`gem`, `maven`, `npm`, and `nuget` download source assets and push them to the destination.

Common flags across most migrate subcommands:

| Flag | Short | Default | Description |
| ---- | ----- | ------- | ----------- |
| `--delete` | | `false` | Delete source versions after successful migration |
| `--dst` | `-d` | | Destination `[host/]owner[/repo]` (required) |
| `--dst-token` | | `$GH_DST_TOKEN` | Access token for destination owner |
| `--dryrun` | `-n` | `false` | Show what would be migrated without performing the migration |
| `--latest` | `-l` | | Migrate latest N versions (by creation date) |
| `--since` | | | Migrate versions created on or after this date (RFC3339 or YYYY-MM-DD) |
| `--src` | `-s` | Current repository / owner | Source `[host/]owner[/repo]` |
| `--src-token` | | `$GH_SRC_TOKEN` | Access token for source owner |
| `--until` | | | Migrate versions created on or before this date (RFC3339 or YYYY-MM-DD) |
| `--version` | | All versions | Migrate specific version(s) by ID or name (repeatable) |

### Migrate container packages (gh pkg-kit migrate container)

```sh
gh pkg-kit migrate container <package-name> --dst <dest-owner[/repo]> [flags]
```

Copy container images between owners on `ghcr.io`. Multi-architecture images are preserved.
Package name change is not supported; the source package name is always used at the destination.
By default, OCI image config labels and manifest annotations (e.g. `org.opencontainers.image.source`) are rewritten to reflect the destination owner or host.
Rewriting labels or annotations changes the resulting image digest; use `--no-rewrite-labels` to preserve the original digest only when you are not linking the package to a destination repository.
Specify `/repo` in `--dst` to also set the source label and link the package to a specific destination repository.
Linking to a destination repository requires rewriting labels or annotations, so `--dst <owner/repo>` is incompatible with `--no-rewrite-labels` and will change the resulting digest.

```sh
# Migrate all versions to another org
gh pkg-kit migrate container my-image --dst dst-org

# Migrate only the latest 5 versions with dry-run
gh pkg-kit migrate container my-image --dst dst-org --latest 5 --dryrun

# Migrate and link the package to a specific destination repository
gh pkg-kit migrate container my-image --dst dst-org/dst-repo

# Delete source versions after successful migration
gh pkg-kit migrate container my-image --dst dst-org --delete

# Skip rewriting OCI labels
gh pkg-kit migrate container my-image --dst dst-org --no-rewrite-labels
```

Additional flag:

| Flag | Default | Description |
| ---- | ------- | ----------- |
| `--no-rewrite-labels` | `false` | Skip rewriting OCI image config labels and manifest annotations (for example `org.opencontainers.image.source`); when rewriting is enabled, those metadata updates also change the resulting image digest |

### Migrate legacy docker → ghcr.io (gh pkg-kit migrate docker)

```sh
gh pkg-kit migrate docker <package-name> --dst <dest-owner[/repo]> [flags]
```

Copy images from `docker.pkg.github.com/OWNER/REPO/PACKAGE` to `ghcr.io`.
`--src` must include the repository name.
By default, OCI image config labels and manifest annotations (e.g. `org.opencontainers.image.source`) are rewritten to reflect the destination owner/host.
Specify `/repo` in `--dst` to also set the source label to link the package to a specific destination repository.

```sh
# Migrate from a legacy docker registry to ghcr.io under a new org
gh pkg-kit migrate docker my-image --src old-org/old-repo --dst dst-org

# Restrict to versions created since a date
gh pkg-kit migrate docker my-image --src old-org/old-repo --dst dst-org --since 2024-01-01

# Migrate and link the package to a specific destination repository
gh pkg-kit migrate docker my-image --src old-org/old-repo --dst dst-org/dst-repo

# Skip rewriting OCI labels
gh pkg-kit migrate docker my-image --src old-org/old-repo --dst dst-org --no-rewrite-labels
```

Additional flag:

| Flag | Default | Description |
| ---- | ------- | ----------- |
| `--no-rewrite-labels` | `false` | Skip rewriting OCI image config labels and manifest annotations (e.g. `org.opencontainers.image.source`) |

### Migrate RubyGems packages (gh pkg-kit migrate gem)

```sh
gh pkg-kit migrate gem <package-name> --dst <dest-owner[/repo]> [flags]
```

```sh
gh pkg-kit migrate gem my-gem --dst dst-org
gh pkg-kit migrate gem my-gem --dst dst-org --latest 3 --dryrun
```

### Migrate Maven packages (gh pkg-kit migrate maven)

```sh
gh pkg-kit migrate maven <package-name> --dst <dest-owner[/repo]> [flags]
```

`--src` must include the repository name. The repository name in `--dst` is optional; if omitted, it is inferred from the source package metadata.

```sh
# Basic migration
gh pkg-kit migrate maven com.example:my-artifact --src old-org/old-repo --dst dst-org

# Overwrite existing destination versions on 409 conflict
gh pkg-kit migrate maven com.example:my-artifact --src old-org/old-repo --dst dst-org/dst-repo --overwrite
```

Additional flag:

| Flag | Default | Description |
| ---- | ------- | ----------- |
| `--overwrite` | `false` | Overwrite existing versions at the destination (delete and re-push on 409 conflict) |

### Migrate npm packages (gh pkg-kit migrate npm)

```sh
gh pkg-kit migrate npm <package-name> --dst <dest-owner[/repo]> [flags]
```

By default, the `repository` field in `package.json` inside each tarball is rewritten to reflect the destination URL. Use `--skip-rewrite-package-json` to disable this behavior.

```sh
# Migrate to another org (rewriting package.json by default)
gh pkg-kit migrate npm my-pkg --dst dst-org

# Skip rewriting package.json
gh pkg-kit migrate npm my-pkg --dst dst-org --skip-rewrite-package-json
```

Additional flags:

| Flag | Default | Description |
| ---- | ------- | ----------- |
| `--overwrite` | `false` | Overwrite existing versions at destination by deleting them before pushing |
| `--skip-rewrite-package-json` | `false` | Skip rewriting `package.json` in the tarball |

### Migrate NuGet packages (gh pkg-kit migrate nuget)

```sh
gh pkg-kit migrate nuget <package-name> --dst <dest-owner[/repo]> [flags]
```

By default, the `<repository>` element in `.nuspec` is rewritten to reflect the destination URL. Use `--skip-rewrite-repository` to disable this behavior.

```sh
# Migrate to another org
gh pkg-kit migrate nuget MyPackage --dst dst-org

# Overwrite destination on conflict
gh pkg-kit migrate nuget MyPackage --dst dst-org --overwrite
```

Additional flags:

| Flag | Default | Description |
| ---- | ------- | ----------- |
| `--overwrite` | `false` | Overwrite existing versions at the destination (delete and re-push on 409 conflict) |
| `--skip-rewrite-repository` | `false` | Skip rewriting `<repository>` element in `.nuspec` |

## Organization packages (gh pkg-kit org)

Manage packages owned by an organization. `--type` selects the package type: `npm`, `maven`, `rubygems`, `docker`, `nuget`, or `container`.
`--owner` defaults to the current repository's owner when omitted.

### List organization packages (gh pkg-kit org list)

```sh
gh pkg-kit org list [flags]
```

Lists packages readable by the user. If `--type` is not specified, lists packages for all types.

```sh
# List all packages across all types
gh pkg-kit org list --owner my-org

# List only container packages
gh pkg-kit org list --owner my-org --type container

# Filter by visibility
gh pkg-kit org list --owner my-org --visibility private

# JSON output filtered with jq
gh pkg-kit org list --owner my-org --format json --jq '.[].name'
```

**Flags:**

| Flag | Short | Default | Description |
| ---- | ----- | ------- | ----------- |
| `--format` | | | Output format: {json} |
| `--jq` | `-q` | | Filter JSON output using a jq expression |
| `--owner` | `-o` | Current repository owner | Owner name |
| `--template` | `-t` | | Format JSON output using a Go template |
| `--type` | `-T` | All types | Package type |
| `--visibility` | `-V` | | Package visibility (public, private, internal) |

### Get, delete, restore an organization package

```sh
gh pkg-kit org get <package-name> --type <type> [--owner <owner>] [--format json] [--jq <expr>] [--template <tmpl>]
gh pkg-kit org delete <package-name> --type <type> [--owner <owner>]
gh pkg-kit org restore <package-name> --type <type> [--owner <owner>]
```

- `delete`: Deletes an entire package. Public packages with any version exceeding 5,000 downloads cannot be deleted.
- `restore`: Restores a deleted package. The package must have been deleted within the last 30 days and the namespace must still be available.

```sh
gh pkg-kit org get my-image --type container --owner my-org
gh pkg-kit org delete my-image --type container --owner my-org
gh pkg-kit org restore my-image --type container --owner my-org
```

### Organization package versions (gh pkg-kit org versions)

```sh
gh pkg-kit org versions list <package-name> --type <type> [--owner <owner>] [--state <state>]
gh pkg-kit org versions get <package-name> <version-id> --type <type> [--owner <owner>]
gh pkg-kit org versions delete <package-name> <version-id> --type <type> [--owner <owner>]
gh pkg-kit org versions restore <package-name> <version-id> --type <type> [--owner <owner>]
```

`--state` filters by `active` or `deleted` (default: `active`).

```sh
# List active versions
gh pkg-kit org versions list my-image --type container --owner my-org

# List deleted versions
gh pkg-kit org versions list my-image --type container --owner my-org --state deleted

# Delete a specific version by ID
gh pkg-kit org versions delete my-image 1234567 --type container --owner my-org

# Restore a specific version by ID (within 30 days of deletion)
gh pkg-kit org versions restore my-image 1234567 --type container --owner my-org
```

## User packages (gh pkg-kit user)

Manage packages owned by a user. `--owner` defaults to the authenticated user when omitted.
The command set mirrors `gh pkg-kit org`:

```sh
gh pkg-kit user list [--type <type>] [--owner <owner>] [--visibility <visibility>] [--format json]
gh pkg-kit user get <package-name> --type <type> [--owner <owner>]
gh pkg-kit user delete <package-name> --type <type> [--owner <owner>]
gh pkg-kit user restore <package-name> --type <type> [--owner <owner>]
gh pkg-kit user versions list <package-name> --type <type> [--owner <owner>] [--state <state>]
gh pkg-kit user versions get <package-name> <version-id> --type <type> [--owner <owner>]
gh pkg-kit user versions delete <package-name> <version-id> --type <type> [--owner <owner>]
gh pkg-kit user versions restore <package-name> <version-id> --type <type> [--owner <owner>]
```

```sh
# List all packages for the authenticated user
gh pkg-kit user list

# List npm packages for a specific user
gh pkg-kit user list --owner octocat --type npm

# Delete a specific version
gh pkg-kit user versions delete my-pkg 1234567 --type npm

# Restore a deleted package (within 30 days)
gh pkg-kit user restore my-pkg --type npm
```

## Common Workflows

### Migrate all container images to a new organization

```sh
# Dry-run first
gh pkg-kit migrate container my-image --src old-org --dst new-org --dryrun

# Migrate and delete source on success
gh pkg-kit migrate container my-image --src old-org --dst new-org --delete
```

### Move packages off the legacy Docker registry to ghcr.io

```sh
gh pkg-kit migrate docker my-image --src old-org/old-repo --dst new-org
```

### Mirror a package locally

```sh
# Save a container image as a tarball for offline use
gh pkg-kit container pull my-image --owner my-org --tag v1.0.0 --output ./my-image-v1.0.0.tar
docker load -i ./my-image-v1.0.0.tar
```

### Clean up old package versions

```sh
# List deleted versions still within the 30-day restore window
gh pkg-kit org versions list my-image --type container --owner my-org --state deleted

# Permanently delete specific active versions (requires write:packages scope)
gh pkg-kit org versions delete my-image 1234567 --type container --owner my-org
```

## Output Formatting

`org`, `user`, and their `versions` subcommands support JSON output with optional `jq` filtering or Go templates:

```sh
# Raw JSON
gh pkg-kit org list --owner my-org --format json

# jq filter
gh pkg-kit org list --owner my-org --format json --jq '.[] | {name, visibility}'

# Go template
gh pkg-kit org list --owner my-org --format json --template '{{range .}}{{.name}}{{"\n"}}{{end}}'
```

## Best Practices

1. **Use `--dryrun` for migrations.** `migrate` commands can move large amounts of data; always verify with `--dryrun` before running.
2. **Scope your tokens appropriately.** Container operations require a classic PAT with `read:packages` / `write:packages`.
3. **Use `--read-only` for inspection sessions.** The global `--read-only` flag blocks any write operation, which is useful when scripting against production owners.
4. **Filter by date or count.** Use `--since`, `--until`, and `--latest` to migrate packages incrementally instead of all at once.
5. **Keep source tokens and destination tokens separate.** Prefer `GH_SRC_TOKEN` and `GH_DST_TOKEN` environment variables (or `.env`) over passing secrets on the command line.

## References

- Repository: <https://github.com/srz-zumix/gh-pkg-kit>
- GitHub Packages docs: <https://docs.github.com/en/packages>
- GitHub Container Registry: <https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry>
- Shell completion: <https://github.com/srz-zumix/go-gh-extension/blob/main/docs/shell-completion.md>
