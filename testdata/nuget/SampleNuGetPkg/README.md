# SampleNuGetPkg

A minimal NuGet package used for **gh-pkg-kit integration testing**.

## Build

```sh
dotnet pack -c Release
```

The `.nupkg` is written to `bin/Release/`.

## Publish to GitHub Packages

Set the required environment variables, then run:

```sh
export GITHUB_OWNER=<your-org-or-user>
export GITHUB_TOKEN=<personal-access-token>

dotnet nuget push bin/Release/SampleNuGetPkg.*.nupkg \
  --source "https://nuget.pkg.github.com/${GITHUB_OWNER}/index.json" \
  --api-key "${GITHUB_TOKEN}"
```

Or via the Makefile in the repo root:

```sh
make testdata-nuget-publish GITHUB_OWNER=<owner>
```
