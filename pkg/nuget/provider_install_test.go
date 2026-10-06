package nuget

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
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
}
