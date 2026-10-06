package credentialprovider

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/srz-zumix/gh-pkg-kit/pkg/nuget"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
)

func captureInstallLogs(t *testing.T) func() string {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "install-log-*")
	if err != nil {
		t.Fatal(err)
	}
	stderr := os.Stderr
	os.Stderr = file
	logger.SetLogLevel("info")
	t.Cleanup(func() {
		os.Stderr = stderr
		logger.SetLogLevel("info")
		if err := file.Close(); err != nil {
			t.Error(err)
		}
	})
	return func() string {
		t.Helper()
		data, err := os.ReadFile(file.Name())
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
}

func TestInstallReportsConventionDiscovery(t *testing.T) {
	logs := captureInstallLogs(t)
	dir := t.TempDir()
	t.Setenv("NUGET_PLUGIN_PATHS", "")
	t.Setenv("NUGET_NETCORE_PLUGIN_PATHS", "")
	var output, stderr bytes.Buffer
	cmd := newInstallCmd(func(_ context.Context, root string, force bool) (string, error) {
		if root != dir || force {
			t.Fatal("incorrect install arguments")
		}
		return filepath.Join(root, nuget.ProviderName, nuget.ProviderName+".dll"), nil
	})
	cmd.SetOut(&output)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--dir", dir})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, nuget.ProviderName, nuget.ProviderName+".dll")
	logOutput := logs()
	if !strings.Contains(logOutput, path) || strings.Contains(logOutput, "NUGET_PLUGIN_PATHS") {
		t.Fatalf("incorrect installation guidance: %s", logOutput)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected discovery warning: %s", stderr.String())
	}
}

func TestInstallWarnsAboutDiscoveryOverrides(t *testing.T) {
	logs := captureInstallLogs(t)
	t.Setenv("NUGET_PLUGIN_PATHS", "/other/plugin")
	t.Setenv("NUGET_NETCORE_PLUGIN_PATHS", "/other/netcore/plugin")
	var stderr bytes.Buffer
	cmd := newInstallCmd(func(context.Context, string, bool) (string, error) { return "/provider.dll", nil })
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--dir", t.TempDir(), "--force"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	logOutput := logs()
	for _, name := range []string{"NUGET_PLUGIN_PATHS", "NUGET_NETCORE_PLUGIN_PATHS"} {
		if !strings.Contains(logOutput, name+" overrides") || !strings.Contains(logOutput, "level=WARN") {
			t.Fatal(logOutput)
		}
	}
}
