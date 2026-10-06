package nuget

import (
	"context"
	"embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/srz-zumix/go-gh-extension/pkg/logger"
)

const ProviderName = "CredentialProvider.GhPkgKit"
const providerMarker = ".gh-pkg-kit-provider"
const providerMarkerContent = "gh-pkg-kit NuGet .NET credential provider v1\n"

//go:embed provider/*.cs provider/*.csproj
var providerSources embed.FS

func DefaultProviderDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".nuget", "plugins", "netcore"), nil
}

func ProvideCredentialProvider(ctx context.Context, workDir string) (string, func(), error) {
	if workDir != "" {
		if err := os.MkdirAll(workDir, 0700); err != nil {
			return "", nil, err
		}
	}
	dir, err := os.MkdirTemp(workDir, "gh-pkg-kit-nuget-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	path, err := InstallCredentialProvider(ctx, dir, false)
	if err != nil {
		cleanup()
		return "", nil, err
	}
	return path, cleanup, nil
}

func InstallCredentialProvider(ctx context.Context, dir string, force bool) (string, error) {
	return installCredentialProvider(ctx, dir, force, func(ctx context.Context, sourceDir, outputDir string) error {
		command := exec.CommandContext(ctx, "dotnet", "publish", filepath.Join(sourceDir, ProviderName+".csproj"),
			"--configuration", "Release", "--output", outputDir, "--verbosity", "minimal",
			"--property:RestoreConfigFile="+filepath.Join(sourceDir, "NuGet.Config"))
		command.Dir = sourceDir
		output, err := command.CombinedOutput()
		if err != nil {
			return fmt.Errorf("dotnet publish failed: %w: %s", err, strings.TrimSpace(string(output)))
		}
		return nil
	})
}

func installCredentialProvider(ctx context.Context, dir string, force bool, publish func(context.Context, string, string) error) (string, error) {
	if dir == "" {
		var err error
		dir, err = DefaultProviderDir()
		if err != nil {
			return "", fmt.Errorf("failed to resolve NuGet plugin directory: %w", err)
		}
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, ProviderName)
	existing := false
	if _, err := os.Lstat(path); err == nil {
		if err := validateProviderDirectory(path); err != nil {
			return "", err
		}
		if !force {
			return "", fmt.Errorf("credential provider already installed: %s (use --force to replace)", path)
		}
		existing = true
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	sourceDir, err := os.MkdirTemp("", "gh-pkg-kit-provider-source-*")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(sourceDir) }()
	entries, err := providerSources.ReadDir("provider")
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		data, err := providerSources.ReadFile("provider/" + entry.Name())
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(filepath.Join(sourceDir, entry.Name()), data, 0600); err != nil {
			return "", err
		}
	}
	config := `<configuration><packageSources><clear/><add key="nuget.org" value="https://api.nuget.org/v3/index.json" /></packageSources></configuration>`
	if err := os.WriteFile(filepath.Join(sourceDir, "NuGet.Config"), []byte(config), 0600); err != nil {
		return "", err
	}
	staging, err := os.MkdirTemp(dir, ".gh-pkg-kit-provider-*")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(staging) }()
	if err := publish(ctx, sourceDir, staging); err != nil {
		return "", err
	}
	for _, suffix := range []string{".dll", ".deps.json", ".runtimeconfig.json"} {
		info, err := os.Lstat(filepath.Join(staging, ProviderName+suffix))
		if err != nil {
			return "", fmt.Errorf("missing published provider file %s: %w", suffix, err)
		}
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("invalid published provider file: %s", info.Name())
		}
	}
	if err := os.WriteFile(filepath.Join(staging, providerMarker), []byte(providerMarkerContent), 0600); err != nil {
		return "", err
	}
	if err := os.Chmod(staging, 0755); err != nil {
		return "", err
	}
	var backup string
	if existing {
		if err := validateProviderDirectory(path); err != nil {
			return "", err
		}
		backup, err = os.MkdirTemp(dir, ".gh-pkg-kit-backup-*")
		if err != nil {
			return "", err
		}
		if err := os.Remove(backup); err != nil {
			return "", err
		}
		if err := os.Rename(path, backup); err != nil {
			return "", err
		}
	} else if _, err := os.Lstat(path); err == nil || !os.IsNotExist(err) {
		return "", fmt.Errorf("provider installation path is no longer available: %s", path)
	}
	if err := os.Rename(staging, path); err != nil {
		if backup != "" {
			if restoreErr := os.Rename(backup, path); restoreErr != nil {
				return "", fmt.Errorf("failed to install provider: %w (previous installation remains at %s)", err, backup)
			}
		}
		return "", err
	}
	if backup != "" {
		if err := os.RemoveAll(backup); err != nil {
			return "", fmt.Errorf("provider installed but failed to remove backup %s: %w", backup, err)
		}
	}
	return filepath.Join(path, ProviderName+".dll"), nil
}

func ReportCredentialProviderInstallation(path string) {
	logger.Info("Installed NuGet credential provider", "path", path)
	logger.Info("Requires .NET 8 or later and gh on PATH at runtime")
	for _, name := range []string{"NUGET_PLUGIN_PATHS", "NUGET_NETCORE_PLUGIN_PATHS"} {
		if os.Getenv(name) != "" {
			logger.Warn(name+" overrides convention-based plugin discovery; unset it or include the installed DLL path", "path", path)
		}
	}
}

func validateProviderDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to modify non-directory provider path: %s", path)
	}
	markerPath := filepath.Join(path, providerMarker)
	markerInfo, err := os.Lstat(markerPath)
	if err != nil || !markerInfo.Mode().IsRegular() {
		return fmt.Errorf("refusing to modify provider directory without a generation marker: %s", path)
	}
	marker, err := os.ReadFile(markerPath)
	if err != nil {
		return err
	}
	if string(marker) != providerMarkerContent {
		return fmt.Errorf("refusing to modify provider directory not generated by gh-pkg-kit: %s", path)
	}
	return nil
}

func RemoveCredentialProvider(dir string) error {
	path := filepath.Join(dir, ProviderName)
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	if err := validateProviderDirectory(path); err != nil {
		return err
	}
	return os.RemoveAll(path)
}

func RunInstalledCredentialProvider(ctx context.Context, input io.Reader, output, stderr io.Writer) error {
	dir, err := DefaultProviderDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, ProviderName, ProviderName+".dll")
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("credential provider is not installed; run gh pkg-kit nuget credential-provider install: %w", err)
	}
	command := exec.CommandContext(ctx, "dotnet", path, "-Plugin")
	command.Stdin, command.Stdout, command.Stderr = input, output, stderr
	return command.Run()
}
