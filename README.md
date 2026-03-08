# gh-pkg-kit

Package-related operations extensions for GitHub CLI.

## Installation

```sh
gh extension install srz-zumix/gh-pkg-kit
```

## Usage

### Global Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--help` | `-h` | Help for gh-pkg-kit | |
| `--log-level` | `-L` | Set log level: {debug\|info\|warn\|error} | `info` |
| `--read-only` | | Run in read-only mode (prevent write operations) | |
| `--version` | `-v` | Version for gh-pkg-kit | |

## org

### org delete

```sh
gh pkg-kit org delete <package-name> --type <type> [--owner <owner>]
```

Deletes an entire package in an organization. You cannot delete a public package if any version of the package has more than 5,000 downloads.

| Flag | Short | Description | Required | Default |
|------|-------|-------------|----------|---------|
| `--owner` | `-o` | Owner name | No | Current repository owner |
| `--type` | `-T` | Package type (npm, maven, rubygems, docker, nuget, container) | Yes | |

### org get

```sh
gh pkg-kit org get <package-name> --type <type> [--owner <owner>] [--format <format>] [--jq <expression>] [--template <template>]
```

Gets a specific package in an organization.

| Flag | Short | Description | Required | Default |
|------|-------|-------------|----------|---------|
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
|------|-------|-------------|----------|---------|
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
|------|-------|-------------|----------|---------|
| `--owner` | `-o` | Owner name | No | Current repository owner |
| `--type` | `-T` | Package type (npm, maven, rubygems, docker, nuget, container) | Yes | |

### org versions delete

```sh
gh pkg-kit org versions delete <package-name> <version-id> --type <type> [--owner <owner>]
```

Deletes a specific package version in an organization. If the package is public and the package version has more than 5,000 downloads, you cannot delete the package version.

| Flag | Short | Description | Required | Default |
|------|-------|-------------|----------|---------|
| `--owner` | `-o` | Owner name | No | Current repository owner |
| `--type` | `-T` | Package type (npm, maven, rubygems, docker, nuget, container) | Yes | |

### org versions get

```sh
gh pkg-kit org versions get <package-name> <version-id> --type <type> [--owner <owner>] [--format <format>] [--jq <expression>] [--template <template>]
```

Gets a specific package version in an organization.

| Flag | Short | Description | Required | Default |
|------|-------|-------------|----------|---------|
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
|------|-------|-------------|----------|---------|
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
|------|-------|-------------|----------|---------|
| `--owner` | `-o` | Owner name | No | Current repository owner |
| `--type` | `-T` | Package type (npm, maven, rubygems, docker, nuget, container) | Yes | |

## user

### user delete

```sh
gh pkg-kit user delete <package-name> --type <type> [--owner <owner>]
```

Deletes an entire package for a user. You cannot delete a public package if any version of the package has more than 5,000 downloads.

| Flag | Short | Description | Required | Default |
|------|-------|-------------|----------|---------|
| `--owner` | `-o` | Owner name | No | Authenticated user |
| `--type` | `-T` | Package type (npm, maven, rubygems, docker, nuget, container) | Yes | |

### user get

```sh
gh pkg-kit user get <package-name> --type <type> [--owner <owner>] [--format <format>] [--jq <expression>] [--template <template>]
```

Gets a specific package metadata for a package owned by a user.

| Flag | Short | Description | Required | Default |
|------|-------|-------------|----------|---------|
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
|------|-------|-------------|----------|---------|
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
|------|-------|-------------|----------|---------|
| `--owner` | `-o` | Owner name | No | Authenticated user |
| `--type` | `-T` | Package type (npm, maven, rubygems, docker, nuget, container) | Yes | |

### user versions delete

```sh
gh pkg-kit user versions delete <package-name> <version-id> --type <type> [--owner <owner>]
```

Deletes a specific package version for a user. If the package is public and the package version has more than 5,000 downloads, you cannot delete the package version.

| Flag | Short | Description | Required | Default |
|------|-------|-------------|----------|---------|
| `--owner` | `-o` | Owner name | No | Authenticated user |
| `--type` | `-T` | Package type (npm, maven, rubygems, docker, nuget, container) | Yes | |

### user versions get

```sh
gh pkg-kit user versions get <package-name> <version-id> --type <type> [--owner <owner>] [--format <format>] [--jq <expression>] [--template <template>]
```

Gets a specific package version for a package owned by a user.

| Flag | Short | Description | Required | Default |
|------|-------|-------------|----------|---------|
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
|------|-------|-------------|----------|---------|
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
|------|-------|-------------|----------|---------|
| `--owner` | `-o` | Owner name | No | Authenticated user |
| `--type` | `-T` | Package type (npm, maven, rubygems, docker, nuget, container) | Yes | |
