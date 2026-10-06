package nuget

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/srz-zumix/go-gh-extension/pkg/logger"
)

func TestInstallCredentialProvider(t *testing.T) {
	dir := t.TempDir()
	publish := func(_ context.Context, source, output string) error {
		if _, err := os.Stat(filepath.Join(source, ProviderName+".csproj")); err != nil {
			return err
		}
		for _, suffix := range []string{".dll", ".deps.json", ".runtimeconfig.json"} {
			if err := os.WriteFile(filepath.Join(output, ProviderName+suffix), []byte("test output"), 0644); err != nil {
				return err
			}
		}
		return nil
	}
	path, err := installCredentialProvider(context.Background(), dir, false, publish)
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(dir, ProviderName, ProviderName+".dll") {
		t.Fatal(path)
	}
	if _, err := installCredentialProvider(context.Background(), dir, false, publish); err == nil {
		t.Fatal("must not overwrite without force")
	}
	if _, err := installCredentialProvider(context.Background(), dir, true, func(context.Context, string, string) error { return fmt.Errorf("publish failed") }); err == nil {
		t.Fatal("must report publish failure")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("failed publish must preserve existing installation")
	}
	if _, err := installCredentialProvider(context.Background(), dir, true, publish); err != nil {
		t.Fatal(err)
	}
	if err := RemoveCredentialProvider(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("provider not removed")
	}
	if err := RemoveCredentialProvider(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, ProviderName), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := installCredentialProvider(context.Background(), dir, true, publish); err == nil {
		t.Fatal("force must not replace unmanaged directory")
	}
	if err := RemoveCredentialProvider(dir); err == nil {
		t.Fatal("must not remove unmanaged directory")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	path, err = installCredentialProvider(context.Background(), "", false, publish)
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(home, ".nuget", "plugins", "netcore", ProviderName, ProviderName+".dll") {
		t.Fatalf("incorrect default installation path: %s", path)
	}
}

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
	t.Setenv("NUGET_PLUGIN_PATHS", "")
	t.Setenv("NUGET_NETCORE_PLUGIN_PATHS", "")
	path := filepath.Join(t.TempDir(), ProviderName, ProviderName+".dll")
	ReportCredentialProviderInstallation(path)
	logOutput := logs()
	if !strings.Contains(logOutput, path) || strings.Contains(logOutput, "NUGET_PLUGIN_PATHS") || strings.Contains(logOutput, "level=WARN") {
		t.Fatalf("incorrect installation guidance: %s", logOutput)
	}
}

func TestInstallWarnsAboutDiscoveryOverrides(t *testing.T) {
	logs := captureInstallLogs(t)
	t.Setenv("NUGET_PLUGIN_PATHS", "/other/plugin")
	t.Setenv("NUGET_NETCORE_PLUGIN_PATHS", "/other/netcore/plugin")
	ReportCredentialProviderInstallation("/provider.dll")
	logOutput := logs()
	for _, name := range []string{"NUGET_PLUGIN_PATHS", "NUGET_NETCORE_PLUGIN_PATHS"} {
		if !strings.Contains(logOutput, name+" overrides") || !strings.Contains(logOutput, "level=WARN") {
			t.Fatal(logOutput)
		}
	}
}
