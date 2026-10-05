package credentialprovider

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/srz-zumix/gh-pkg-kit/pkg/nuget"
)

func TestInstallReportsConventionDiscovery(t *testing.T) {
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
	if !strings.Contains(output.String(), path) || strings.Contains(output.String(), "NUGET_PLUGIN_PATHS") {
		t.Fatalf("incorrect installation guidance: %s", output.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected discovery warning: %s", stderr.String())
	}
}

func TestInstallWarnsAboutDiscoveryOverrides(t *testing.T) {
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
	for _, name := range []string{"NUGET_PLUGIN_PATHS", "NUGET_NETCORE_PLUGIN_PATHS"} {
		if !strings.Contains(stderr.String(), name+" overrides") {
			t.Fatal(stderr.String())
		}
	}
}
