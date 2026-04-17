# gh-pkg-kit

Package-related operations extensions for GitHub CLI.

## Installation

```sh
gh extension install srz-zumix/gh-pkg-kit
```

## Shell Completion

**Workaround Available!** While gh CLI doesn't natively support extension completion, we provide a patch script that enables it.

**Prerequisites:** Before setting up gh-pkg-kit completion, ensure gh CLI completion is configured for your shell. See [gh completion documentation](https://cli.github.com/manual/gh_completion) for setup instructions.

For detailed installation instructions and setup for each shell, see the [Shell Completion Guide](https://github.com/srz-zumix/go-gh-extension/blob/main/docs/shell-completion.md).

## Configuration

This tool automatically loads a `.env` file from the current directory on startup (via [godotenv](https://github.com/joho/godotenv)).
You can set authentication tokens and other environment variables in a `.env` file instead of exporting them in your shell.

See [.env.example](.env.example) for available variables and usage notes.

| Variable | Description |
| ---- | ----------- |
| `GH_TOKEN` | Token for github.com (priority: GH_TOKEN > GITHUB_TOKEN > `gh auth` config) |
| `GH_ENTERPRISE_TOKEN` | Token for GitHub Enterprise Server (priority: GH_ENTERPRISE_TOKEN > GITHUB_ENTERPRISE_TOKEN > `gh auth` config) |

> **Note**: Container registries (ghcr.io / containers.\<host\>) require a classic PAT with `read:packages` / `write:packages` scope. Fine-grained tokens and OAuth App tokens may be rejected.

## Usage

### Global Flags

| Flag | Short | Description | Default |
| ---- | ----- | ----------- | ------- |
| `--help` | `-h` | Help for gh-pkg-kit | |
| `--log-level` | `-L` | Set log level: {debug\|info\|warn\|error} | `info` |
| `--read-only` | | Run in read-only mode (prevent write operations) | |
| `--version` | `-v` | Version for gh-pkg-kit | |

## container

### container pull

```sh
gh pkg-kit container pull <package-name> [--owner <owner>] [--tag <tag>] [--output <path>] [--dry-run]
```

Pulls a container image from the GitHub Container Registry (ghcr.io) and saves it as a Docker-loadable tarball.
The tag defaults to `"latest"` if not specified.
The output file defaults to `<package-name>-<tag>.tar` in the current directory.
The saved tarball can be loaded with: `docker load -i <output-file>`

| Flag | Short | Description | Required | Default |
| ---- | ----- | ----------- | -------- | ------- |
| `--dry-run` | `-n` | Show what would be pulled without performing the pull | No | `false` |
| `--output` | | Output file path | No | `<package-name>-<tag>.tar` |
| `--owner` | `-o` | [host/]owner | No | Current repository owner |
| `--tag` | `-t` | Image tag to pull | No | `"latest"` |

## docker

### docker pull

```sh
gh pkg-kit docker pull <package-name> [--owner <owner/repo>] [--tag <tag>] [--output <path>] [--dry-run]
```

Pulls a docker image from the legacy Docker Package Registry (`docker.pkg.github.com`) and saves it as a Docker-loadable tarball.
The repository name is required in `--owner` because the legacy Docker Package Registry image path includes the owner and repository (`docker.pkg.github.com/OWNER/REPO/PACKAGE`).
The tag defaults to `"latest"` if not specified.
The output file defaults to `<package-name>-<tag>.tar` in the current directory.
The saved tarball can be loaded with: `docker load -i <output-file>`

| Flag | Short | Description | Required | Default |
| ---- | ----- | ----------- | -------- | ------- |
| `--dry-run` | `-n` | Show what would be pulled without performing the pull | No | `false` |
| `--output` | | Output file path | No | `<package-name>-<tag>.tar` |
| `--owner` | `-o` | [host/]owner/repo (repository name is required) | No | Current repository |
| `--tag` | `-t` | Image tag to pull | No | `"latest"` |

## gem

### gem download

```sh
gh pkg-kit gem download <package-name> [--owner <owner>] [--version <version>] [--output <path>]
```

Downloads a .gem file from the GitHub RubyGems registry.
Version defaults to the latest version if not specified.
The owner is resolved from the current repository if `--owner` is not specified.
The output file defaults to `<package-name>-<version>.gem` in the current directory.

| Flag | Short | Description | Required | Default |
| ---- | ----- | ----------- | -------- | ------- |
| `--output` | | Output file path | No | `<package-name>-<version>.gem` |
| `--owner` | `-o` | [HOST/]OWNER | No | Current repository owner |
| `--version` | | Package version to download | No | Latest version |

## maven

### maven download

```sh
gh pkg-kit maven download <package-name> [--repo <owner/repo>] [--version <version>] [--output-dir <path>]
```

Downloads .pom and .jar files from the GitHub Maven registry.
Accepts both colon-separated format (e.g. `com.example:my-artifact`) and the GitHub Packages dot-separated format (e.g. `com.example.my-artifact`).
Version defaults to the latest version if not specified.
The repository defaults to the current repository if `--repo` is not specified; the repository name must be included.
Output files are written to `--output-dir` (default: current directory) as `<artifactId>-<version>.<ext>`.

| Flag | Short | Description | Required | Default |
| ---- | ----- | ----------- | -------- | ------- |
| `--output-dir` | `-o` | Output directory | No | Current directory |
| `--repo` | `-R` | Repository in [host/]owner/repo format | No | Current repository |
| `--version` | | Package version to download | No | Latest version |

## migrate

### migrate container

```sh
gh pkg-kit migrate container <package-name> --dst <dest-owner[/repo]> [--src <source-owner>] [flags]
```

Migrates container packages from one owner to another within GitHub Packages.
Uses the OCI Distribution API to copy image manifests and blobs, including multi-architecture images.
The source owner is resolved from the current repository if `--src` is not specified.
The source and destination owner types (organization or user) are detected automatically.
Package name change is not supported; the source package name is always used at destination.

| Flag | Short | Description | Required | Default |
| ---- | ----- | ----------- | -------- | ------- |
| `--delete` | | Delete source versions after successful migration | No | `false` |
| `--dst` | `-d` | Destination [host/]owner[/repo] | Yes | |
| `--dst-token` | | Access token for destination owner (fallback: `$GH_DST_TOKEN`) | No | |
| `--dry-run` | `-n` | Show what would be migrated without performing the migration | No | `false` |
| `--latest` | `-l` | Migrate latest N versions (by creation date) | No | |
| `--rewrite-labels` | | Rewrite OCI image config labels to reflect destination owner/host (changes image digest) | No | `false` |
| `--since` | | Migrate versions created on or after this date (RFC3339 or YYYY-MM-DD) | No | |
| `--src` | `-s` | Source [host/]owner | No | Current repository owner |
| `--src-token` | | Access token for source owner (fallback: `$GH_SRC_TOKEN`) | No | |
| `--until` | | Migrate versions created on or before this date (RFC3339 or YYYY-MM-DD) | No | |
| `--version` | | Migrate specific version(s) by ID or name (can be specified multiple times) | No | All versions |

### migrate docker

```sh
gh pkg-kit migrate docker <package-name> --dst <dest-owner[/repo]> [--src <source-owner/repo>] [flags]
```

Migrates docker packages from the legacy `docker.pkg.github.com` registry to `ghcr.io` within GitHub Packages.
Uses the OCI Distribution API to copy image manifests and blobs, including multi-architecture images.
The repository name is required in `--src` because the legacy Docker Package Registry image path includes the owner and repository (`docker.pkg.github.com/OWNER/REPO/PACKAGE`).
The source and destination owner types (organization or user) are detected automatically.
Package name change is not supported; the source package name is always used at destination.

| Flag | Short | Description | Required | Default |
| ---- | ----- | ----------- | -------- | ------- |
| `--delete` | | Delete source versions after successful migration | No | `false` |
| `--dst` | `-d` | Destination [host/]owner[/repo] | Yes | |
| `--dst-token` | | Access token for destination owner (fallback: `$GH_DST_TOKEN`) | No | |
| `--dry-run` | `-n` | Show what would be migrated without performing the migration | No | `false` |
| `--latest` | `-l` | Migrate latest N versions (by creation date) | No | |
| `--rewrite-labels` | | Rewrite OCI image config labels to reflect destination owner/host (changes image digest) | No | `false` |
| `--since` | | Migrate versions created on or after this date (RFC3339 or YYYY-MM-DD) | No | |
| `--src` | `-s` | Source [host/]owner/repo (repository name is required) | No | Current repository |
| `--src-token` | | Access token for source owner (fallback: `$GH_SRC_TOKEN`) | No | |
| `--until` | | Migrate versions created on or before this date (RFC3339 or YYYY-MM-DD) | No | |
| `--version` | | Migrate specific version(s) by ID or name (can be specified multiple times) | No | All versions |

### migrate gem

```sh
gh pkg-kit migrate gem <package-name> --dst <dest-owner[/repo]> [--src <source-owner>] [flags]
```

Migrates RubyGems packages from one owner to another within GitHub Packages.
Downloads .gem files from the source RubyGems registry and pushes them to the destination.
The source owner is resolved from the current repository if `--src` is not specified.
The source and destination owner types (organization or user) are detected automatically.

| Flag | Short | Description | Required | Default |
| ---- | ----- | ----------- | -------- | ------- |
| `--delete` | | Delete source versions after successful migration | No | `false` |
| `--dst` | `-d` | Destination [host/]owner[/repo] | Yes | |
| `--dst-token` | | Access token for destination owner (fallback: `$GH_DST_TOKEN`) | No | |
| `--dry-run` | `-n` | Show what would be migrated without performing the migration | No | `false` |
| `--latest` | `-l` | Migrate latest N versions (by creation date) | No | |
| `--since` | | Migrate versions created on or after this date (RFC3339 or YYYY-MM-DD) | No | |
| `--src` | `-s` | Source [host/]owner | No | Current repository owner |
| `--src-token` | | Access token for source owner (fallback: `$GH_SRC_TOKEN`) | No | |
| `--until` | | Migrate versions created on or before this date (RFC3339 or YYYY-MM-DD) | No | |
| `--version` | | Migrate specific version(s) by ID or name (can be specified multiple times) | No | All versions |

### migrate maven

```sh
gh pkg-kit migrate maven <package-name> --dst <dest-owner[/repo]> [--src <source-owner/repo>] [flags]
```

Migrates Maven packages from one owner to another within GitHub Packages.
Downloads .pom and .jar files from the source Maven registry and pushes them to the destination.
Accepts both colon-separated format (e.g. `com.example:my-artifact`) and the GitHub Packages dot-separated format (e.g. `com.example.my-artifact`).
The source repository is resolved from the current repository if `--src` is not specified; the repository name must be included.
The repository name in `--dst` is optional; if omitted, it is inferred from the source package metadata.

| Flag | Short | Description | Required | Default |
| ---- | ----- | ----------- | -------- | ------- |
| `--delete` | | Delete source versions after successful migration | No | `false` |
| `--dst` | `-d` | Destination [host/]owner[/repo] | Yes | |
| `--dst-token` | | Access token for destination owner (fallback: `$GH_DST_TOKEN`) | No | |
| `--dry-run` | `-n` | Show what would be migrated without performing the migration | No | `false` |
| `--latest` | `-l` | Migrate latest N versions (by creation date) | No | |
| `--overwrite` | | Overwrite existing versions at the destination (delete and re-push on 409 conflict) | No | `false` |
| `--since` | | Migrate versions created on or after this date (RFC3339 or YYYY-MM-DD) | No | |
| `--src` | `-s` | Source [host/]owner/repo (repository name is required) | No | Current repository |
| `--src-token` | | Access token for source owner (fallback: `$GH_SRC_TOKEN`) | No | |
| `--until` | | Migrate versions created on or before this date (RFC3339 or YYYY-MM-DD) | No | |
| `--version` | | Migrate specific version(s) by ID or name (can be specified multiple times) | No | All versions |

### migrate npm

```sh
gh pkg-kit migrate npm <package-name> --dst <dest-owner[/repo]> [--src <source-owner>] [flags]
```

Migrates npm packages from one owner to another within GitHub Packages.
Downloads .tgz tarball files from the source npm registry and pushes them to the destination.
The source owner is resolved from the current repository if `--src` is not specified.
The source and destination owner types (organization or user) are detected automatically.
Package name change is not supported; the source package name is always used at destination.
By default, the `repository` field in `package.json` inside the tarball is rewritten to reflect the destination URL. Use `--skip-rewrite-package-json` to disable this behavior.

| Flag | Short | Description | Required | Default |
| ---- | ----- | ----------- | -------- | ------- |
| `--delete` | | Delete source versions after successful migration | No | `false` |
| `--dst` | `-d` | Destination [host/]owner[/repo] | Yes | |
| `--dst-token` | | Access token for destination owner (fallback: `$GH_DST_TOKEN`) | No | |
| `--dry-run` | `-n` | Show what would be migrated without performing the migration | No | `false` |
| `--latest` | `-l` | Migrate latest N versions (by creation date) | No | |
| `--overwrite` | | Overwrite existing versions at destination by deleting them before pushing | No | `false` |
| `--since` | | Migrate versions created on or after this date (RFC3339 or YYYY-MM-DD) | No | |
| `--skip-rewrite-package-json` | | Skip rewriting `package.json` in the tarball | No | `false` |
| `--src` | `-s` | Source [host/]owner | No | Current repository owner |
| `--src-token` | | Access token for source owner (fallback: `$GH_SRC_TOKEN`) | No | |
| `--until` | | Migrate versions created on or before this date (RFC3339 or YYYY-MM-DD) | No | |
| `--version` | | Migrate specific version(s) by ID or name (can be specified multiple times) | No | All versions |

### migrate nuget

```sh
gh pkg-kit migrate nuget <package-name> --dst <dest-owner[/repo]> [--src <source-owner>] [flags]
```

Migrates NuGet packages from one owner to another within GitHub Packages.
Downloads .nupkg files from the source NuGet registry and pushes them to the destination.
The source owner is resolved from the current repository if `--src` is not specified.
The source and destination owner types (organization or user) are detected automatically.
Package name change is not supported; the source package name is always used at destination.
By default, the `<repository>` element in `.nuspec` is rewritten to reflect the destination URL. Use `--skip-rewrite-repository` to disable this behavior.

| Flag | Short | Description | Required | Default |
| ---- | ----- | ----------- | -------- | ------- |
| `--delete` | | Delete source versions after successful migration | No | `false` |
| `--dst` | `-d` | Destination [host/]owner[/repo] | Yes | |
| `--dst-token` | | Access token for destination owner (fallback: `$GH_DST_TOKEN`) | No | |
| `--dry-run` | `-n` | Show what would be migrated without performing the migration | No | `false` |
| `--latest` | `-l` | Migrate latest N versions (by creation date) | No | |
| `--overwrite` | | Overwrite existing versions at the destination (delete and re-push on 409 conflict) | No | `false` |
| `--since` | | Migrate versions created on or after this date (RFC3339 or YYYY-MM-DD) | No | |
| `--skip-rewrite-repository` | | Skip rewriting `<repository>` element in .nuspec | No | `false` |
| `--src` | `-s` | Source [host/]owner | No | Current repository owner |
| `--src-token` | | Access token for source owner (fallback: `$GH_SRC_TOKEN`) | No | |
| `--until` | | Migrate versions created on or before this date (RFC3339 or YYYY-MM-DD) | No | |
| `--version` | | Migrate specific version(s) by ID or name (can be specified multiple times) | No | All versions |

## npm

### npm download

```sh
gh pkg-kit npm download <package-name> [--owner <owner>] [--version <version>] [--output <path>]
```

Downloads a .tgz tarball file from the GitHub npm registry.
Version defaults to the latest version if not specified.
The owner is resolved from the current repository if `--owner` is not specified.
The output file defaults to `<package-name>.<version>.tgz` in the current directory.

| Flag | Short | Description | Required | Default |
| ---- | ----- | ----------- | -------- | ------- |
| `--output` | | Output file path | No | `<package-name>.<version>.tgz` |
| `--owner` | `-o` | [HOST/]OWNER | No | Current repository owner |
| `--version` | | Package version to download | No | Latest version |

## nuget

### nuget download

```sh
gh pkg-kit nuget download <package-name> [--owner <owner>] [--version <version>] [--output <path>]
```

Downloads a .nupkg file from the GitHub NuGet registry.
Version defaults to the latest version if not specified.
The owner is resolved from the current repository if `--owner` is not specified.
The output file defaults to `<package-name>.<version>.nupkg` in the current directory.

| Flag | Short | Description | Required | Default |
| ---- | ----- | ----------- | -------- | ------- |
| `--output` | | Output file path | No | `<package-name>.<version>.nupkg` |
| `--owner` | `-o` | [HOST/]OWNER | No | Current repository owner |
| `--version` | | Package version to download | No | Latest version |

### nuget tool-restore

```sh
gh pkg-kit nuget tool-restore [--configfile <path>] [--work-dir <path>] [--overwrite] [-- dotnet-tool-restore-args...]
```

Runs `dotnet tool restore` after injecting GitHub Packages credentials from the gh auth token into a NuGet.Config file.
By default, a temporary copy of the NuGet.Config is created with credentials injected.
With `--overwrite`, the credentials are written directly into the existing NuGet.Config.
Extra arguments after `--` are passed through to `dotnet tool restore`.

| Flag | Short | Description | Required | Default |
| ---- | ----- | ----------- | -------- | ------- |
| `--configfile` | | Path to NuGet.Config (auto-detected if not specified) | No | Auto-detect |
| `--overwrite` | | Overwrite the existing NuGet.Config with injected credentials instead of using a temporary copy | No | `false` |
| `--work-dir` | | Working directory for temporary files | No | System temp dir (deleted on exit) |

## org

### org delete

```sh
gh pkg-kit org delete <package-name> --type <type> [--owner <owner>]
```

Deletes an entire package in an organization. You cannot delete a public package if any version of the package has more than 5,000 downloads.

| Flag | Short | Description | Required | Default |
| ---- | ----- | ----------- | -------- | ------- |
| `--owner` | `-o` | Owner name | No | Current repository owner |
| `--type` | `-T` | Package type (npm, maven, rubygems, docker, nuget, container) | Yes | |

### org get

```sh
gh pkg-kit org get <package-name> --type <type> [--owner <owner>] [--format <format>] [--jq <expression>] [--template <template>]
```

Gets a specific package in an organization.

| Flag | Short | Description | Required | Default |
| ---- | ----- | ----------- | -------- | ------- |
| `--format` | | Output format: {json} | No | |
| `--jq` | `-q` | Filter JSON output using a jq expression | No | |
| `--owner` | `-o` | Owner name | No | Current repository owner |
| `--template` | `-t` | Format JSON output using a Go template | No | |
| `--type` | `-T` | Package type (npm, maven, rubygems, docker, nuget, container) | Yes | |

### org list

```sh
gh pkg-kit org list [--type <type>] [--owner <owner>] [--visibility <visibility>] [--format <format>] [--jq <expression>] [--template <template>]
```

Lists packages in an organization readable by the user.
If `--type` is not specified, lists packages for all package types.

| Flag | Short | Description | Required | Default |
| ---- | ----- | ----------- | -------- | ------- |
| `--format` | | Output format: {json} | No | |
| `--jq` | `-q` | Filter JSON output using a jq expression | No | |
| `--owner` | `-o` | Owner name | No | Current repository owner |
| `--template` | `-t` | Format JSON output using a Go template | No | |
| `--type` | `-T` | Package type (npm, maven, rubygems, docker, nuget, container) | No | All types |
| `--visibility` | `-V` | Package visibility (public, private, internal) | No | |

### org restore

```sh
gh pkg-kit org restore <package-name> --type <type> [--owner <owner>]
```

Restores an entire package in an organization.
The package must have been deleted within the last 30 days, and the same package namespace and version must still be available.

| Flag | Short | Description | Required | Default |
| ---- | ----- | ----------- | -------- | ------- |
| `--owner` | `-o` | Owner name | No | Current repository owner |
| `--type` | `-T` | Package type (npm, maven, rubygems, docker, nuget, container) | Yes | |

### org versions delete

```sh
gh pkg-kit org versions delete <package-name> <version-id> --type <type> [--owner <owner>]
```

Deletes a specific package version in an organization. If the package is public and the package version has more than 5,000 downloads, you cannot delete the package version.

| Flag | Short | Description | Required | Default |
| ---- | ----- | ----------- | -------- | ------- |
| `--owner` | `-o` | Owner name | No | Current repository owner |
| `--type` | `-T` | Package type (npm, maven, rubygems, docker, nuget, container) | Yes | |

### org versions get

```sh
gh pkg-kit org versions get <package-name> <version-id> --type <type> [--owner <owner>] [--format <format>] [--jq <expression>] [--template <template>]
```

Gets a specific package version in an organization.

| Flag | Short | Description | Required | Default |
| ---- | ----- | ----------- | -------- | ------- |
| `--format` | | Output format: {json} | No | |
| `--jq` | `-q` | Filter JSON output using a jq expression | No | |
| `--owner` | `-o` | Owner name | No | Current repository owner |
| `--template` | `-t` | Format JSON output using a Go template | No | |
| `--type` | `-T` | Package type (npm, maven, rubygems, docker, nuget, container) | Yes | |

### org versions list

```sh
gh pkg-kit org versions list <package-name> --type <type> [--owner <owner>] [--state <state>] [--format <format>] [--jq <expression>] [--template <template>]
```

Lists package versions for a package owned by an organization.

| Flag | Short | Description | Required | Default |
| ---- | ----- | ----------- | -------- | ------- |
| `--format` | | Output format: {json} | No | |
| `--jq` | `-q` | Filter JSON output using a jq expression | No | |
| `--owner` | `-o` | Owner name | No | Current repository owner |
| `--state` | `-s` | Package state (active, deleted). Default: active | No | |
| `--template` | `-t` | Format JSON output using a Go template | No | |
| `--type` | `-T` | Package type (npm, maven, rubygems, docker, nuget, container) | Yes | |

### org versions restore

```sh
gh pkg-kit org versions restore <package-name> <version-id> --type <type> [--owner <owner>]
```

Restores a specific package version in an organization.
The package must have been deleted within the last 30 days, and the same package namespace and version must still be available.

| Flag | Short | Description | Required | Default |
| ---- | ----- | ----------- | -------- | ------- |
| `--owner` | `-o` | Owner name | No | Current repository owner |
| `--type` | `-T` | Package type (npm, maven, rubygems, docker, nuget, container) | Yes | |

## user

### user delete

```sh
gh pkg-kit user delete <package-name> --type <type> [--owner <owner>]
```

Deletes an entire package for a user. You cannot delete a public package if any version of the package has more than 5,000 downloads.

| Flag | Short | Description | Required | Default |
| ---- | ----- | ----------- | -------- | ------- |
| `--owner` | `-o` | Owner name | No | Authenticated user |
| `--type` | `-T` | Package type (npm, maven, rubygems, docker, nuget, container) | Yes | |

### user get

```sh
gh pkg-kit user get <package-name> --type <type> [--owner <owner>] [--format <format>] [--jq <expression>] [--template <template>]
```

Gets a specific package metadata for a package owned by a user.

| Flag | Short | Description | Required | Default |
| ---- | ----- | ----------- | -------- | ------- |
| `--format` | | Output format: {json} | No | |
| `--jq` | `-q` | Filter JSON output using a jq expression | No | |
| `--owner` | `-o` | Owner name | No | Authenticated user |
| `--template` | `-t` | Format JSON output using a Go template | No | |
| `--type` | `-T` | Package type (npm, maven, rubygems, docker, nuget, container) | Yes | |

### user list

```sh
gh pkg-kit user list [--type <type>] [--owner <owner>] [--visibility <visibility>] [--format <format>] [--jq <expression>] [--template <template>]
```

Lists all packages in a user's namespace for which the requesting user has access.
If `--type` is not specified, lists packages for all package types.

| Flag | Short | Description | Required | Default |
| ---- | ----- | ----------- | -------- | ------- |
| `--format` | | Output format: {json} | No | |
| `--jq` | `-q` | Filter JSON output using a jq expression | No | |
| `--owner` | `-o` | Owner name | No | Authenticated user |
| `--template` | `-t` | Format JSON output using a Go template | No | |
| `--type` | `-T` | Package type (npm, maven, rubygems, docker, nuget, container) | No | All types |
| `--visibility` | `-V` | Package visibility (public, private, internal) | No | |

### user restore

```sh
gh pkg-kit user restore <package-name> --type <type> [--owner <owner>]
```

Restores an entire package for a user.
The package must have been deleted within the last 30 days, and the same package namespace and version must still be available.

| Flag | Short | Description | Required | Default |
| ---- | ----- | ----------- | -------- | ------- |
| `--owner` | `-o` | Owner name | No | Authenticated user |
| `--type` | `-T` | Package type (npm, maven, rubygems, docker, nuget, container) | Yes | |

### user versions delete

```sh
gh pkg-kit user versions delete <package-name> <version-id> --type <type> [--owner <owner>]
```

Deletes a specific package version for a user. If the package is public and the package version has more than 5,000 downloads, you cannot delete the package version.

| Flag | Short | Description | Required | Default |
| ---- | ----- | ----------- | -------- | ------- |
| `--owner` | `-o` | Owner name | No | Authenticated user |
| `--type` | `-T` | Package type (npm, maven, rubygems, docker, nuget, container) | Yes | |

### user versions get

```sh
gh pkg-kit user versions get <package-name> <version-id> --type <type> [--owner <owner>] [--format <format>] [--jq <expression>] [--template <template>]
```

Gets a specific package version for a package owned by a user.

| Flag | Short | Description | Required | Default |
| ---- | ----- | ----------- | -------- | ------- |
| `--format` | | Output format: {json} | No | |
| `--jq` | `-q` | Filter JSON output using a jq expression | No | |
| `--owner` | `-o` | Owner name | No | Authenticated user |
| `--template` | `-t` | Format JSON output using a Go template | No | |
| `--type` | `-T` | Package type (npm, maven, rubygems, docker, nuget, container) | Yes | |

### user versions list

```sh
gh pkg-kit user versions list <package-name> --type <type> [--owner <owner>] [--state <state>] [--format <format>] [--jq <expression>] [--template <template>]
```

Lists package versions for a package owned by a user.

| Flag | Short | Description | Required | Default |
| ---- | ----- | ----------- | -------- | ------- |
| `--format` | | Output format: {json} | No | |
| `--jq` | `-q` | Filter JSON output using a jq expression | No | |
| `--owner` | `-o` | Owner name | No | Authenticated user |
| `--state` | `-s` | Package state (active, deleted). Default: active | No | |
| `--template` | `-t` | Format JSON output using a Go template | No | |
| `--type` | `-T` | Package type (npm, maven, rubygems, docker, nuget, container) | Yes | |

### user versions restore

```sh
gh pkg-kit user versions restore <package-name> <version-id> --type <type> [--owner <owner>]
```

Restores a specific package version for a user.
The package must have been deleted within the last 30 days, and the same package namespace and version must still be available.

| Flag | Short | Description | Required | Default |
| ---- | ----- | ----------- | -------- | ------- |
| `--owner` | `-o` | Owner name | No | Authenticated user |
| `--type` | `-T` | Package type (npm, maven, rubygems, docker, nuget, container) | Yes | |
