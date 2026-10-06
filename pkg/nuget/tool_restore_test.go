package nuget

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestToolRestoreTemporaryProvider(t *testing.T) {
	testRestoreTemporaryProvider(t, RunToolRestore, []string{"tool", "restore"}, nil)
}

func TestRestoreTemporaryProvider(t *testing.T) {
	testRestoreTemporaryProvider(t, RunRestore, []string{"restore"}, []string{"Sample Project.csproj"})
}

func testRestoreTemporaryProvider(t *testing.T, run func(context.Context, string, string, []string) error, commandArgs, extraArgs []string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake dotnet uses a shell script")
	}
	for _, outcome := range []string{"success", "restore-failure", "publish-failure", "default-discovery"} {
		t.Run(outcome, func(t *testing.T) {
			dir := t.TempDir()
			config := filepath.Join(dir, "NuGet.Config")
			original := `<configuration><packageSources><clear/></packageSources></configuration>`
			if err := os.WriteFile(config, []byte(original), 0600); err != nil {
				t.Fatal(err)
			}
			script := `#!/bin/sh
if [ "$1" = publish ]; then
  [ "$TEST_OUTCOME" != publish-failure ] || exit 1
  while [ "$1" != --output ]; do shift; done
  shift
  for suffix in .dll .deps.json .runtimeconfig.json; do
    printf test > "$1/CredentialProvider.GhPkgKit$suffix"
  done
  exit 0
fi
printf '%s\n' "$@" > "$TEST_ARGS"
printf '%s\n' "$NUGET_PLUGIN_PATHS" "$NUGET_NETCORE_PLUGIN_PATHS" > "$TEST_ENV"
provider="${NUGET_NETCORE_PLUGIN_PATHS##*;}"
[ -f "$provider" ] || exit 2
[ "$TEST_OUTCOME" != restore-failure ] || exit 3
`
			if err := os.WriteFile(filepath.Join(dir, "dotnet"), []byte(script), 0755); err != nil {
				t.Fatal(err)
			}
			argsFile := filepath.Join(dir, "args")
			envFile := filepath.Join(dir, "env")
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("TEST_OUTCOME", outcome)
			t.Setenv("TEST_ARGS", argsFile)
			t.Setenv("TEST_ENV", envFile)
			t.Setenv("NUGET_PLUGIN_PATHS", "/existing/first.dll;/existing/second.dll")
			t.Setenv("NUGET_NETCORE_PLUGIN_PATHS", "/existing/netcore.dll")
			configArgument := config
			restoreArgs := append(append([]string(nil), extraArgs...), "--verbosity", "minimal")
			expectedArgs := strings.Join(commandArgs, "\n") + "\n--configfile\n" + config + "\n" + strings.Join(restoreArgs, "\n") + "\n"
			if outcome == "default-discovery" {
				configArgument = ""
				expectedArgs = strings.Join(commandArgs, "\n") + "\n" + strings.Join(restoreArgs, "\n") + "\n"
				t.Setenv("NUGET_PLUGIN_PATHS", "")
				t.Setenv("NUGET_NETCORE_PLUGIN_PATHS", "")
			}
			originalPluginPaths := os.Getenv("NUGET_PLUGIN_PATHS")
			originalNetcorePaths := os.Getenv("NUGET_NETCORE_PLUGIN_PATHS")
			workDir := filepath.Join(dir, "work")
			err := run(context.Background(), configArgument, workDir, restoreArgs)
			expectFailure := outcome == "restore-failure" || outcome == "publish-failure"
			if (err != nil) != expectFailure {
				t.Fatalf("unexpected result for %s: %v", outcome, err)
			}
			content, err := os.ReadFile(config)
			if err != nil || string(content) != original {
				t.Fatalf("NuGet.Config changed: %v", err)
			}
			entries, err := os.ReadDir(workDir)
			if err != nil || len(entries) != 0 {
				t.Fatalf("temporary provider not cleaned up: %v, %v", entries, err)
			}
			if outcome == "publish-failure" {
				if _, err := os.Stat(argsFile); !os.IsNotExist(err) {
					t.Fatal("restore must not run after publish failure")
				}
				return
			}
			arguments, err := os.ReadFile(argsFile)
			if err != nil || string(arguments) != expectedArgs {
				t.Fatalf("unexpected restore arguments: %s, %v", arguments, err)
			}
			environment, err := os.ReadFile(envFile)
			if err != nil {
				t.Fatal(err)
			}
			paths := strings.Split(strings.TrimSpace(string(environment)), "\n")
			if len(paths) != 2 {
				t.Fatalf("unexpected plugin environment: %s", environment)
			}
			providerPath := ""
			for index, originalPaths := range []string{originalPluginPaths, originalNetcorePaths} {
				prefix := originalPaths
				if prefix != "" {
					prefix += ";"
				}
				if !strings.HasPrefix(paths[index], prefix) {
					t.Fatalf("existing plugin paths not preserved: %s", environment)
				}
				path := strings.TrimPrefix(paths[index], prefix)
				if index == 0 {
					providerPath = path
				}
				if path != providerPath || !strings.HasSuffix(path, ProviderName+".dll") {
					t.Fatalf("incorrect temporary provider path: %s", path)
				}
			}
			if os.Getenv("NUGET_PLUGIN_PATHS") != originalPluginPaths || os.Getenv("NUGET_NETCORE_PLUGIN_PATHS") != originalNetcorePaths {
				t.Fatal("parent environment changed")
			}
		})
	}
}

func TestToolRestoreRejectsMissingConfigBeforePreparingProvider(t *testing.T) {
	testRestoreRejectsMissingConfigBeforePreparingProvider(t, RunToolRestore)
}

func TestRestoreRejectsMissingConfigBeforePreparingProvider(t *testing.T) {
	testRestoreRejectsMissingConfigBeforePreparingProvider(t, RunRestore)
}

func testRestoreRejectsMissingConfigBeforePreparingProvider(t *testing.T, run func(context.Context, string, string, []string) error) {
	t.Helper()
	dir := t.TempDir()
	config := filepath.Join(dir, "missing.config")
	workDir := filepath.Join(dir, "work")
	err := run(context.Background(), config, workDir, nil)
	if err == nil || !strings.Contains(err.Error(), "config file not found: "+config) {
		t.Fatalf("expected missing config error: %v", err)
	}
	if _, err := os.Stat(workDir); !os.IsNotExist(err) {
		t.Fatal("must validate config before preparing a provider")
	}
}
