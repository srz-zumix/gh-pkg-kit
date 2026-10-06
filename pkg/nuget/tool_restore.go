package nuget

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/srz-zumix/go-gh-extension/pkg/logger"
)

func RunToolRestore(ctx context.Context, configFile, workDir string, args []string) error {
	return runRestore(ctx, []string{"tool", "restore"}, configFile, workDir, args)
}

func RunRestore(ctx context.Context, configFile, workDir string, args []string) error {
	return runRestore(ctx, []string{"restore"}, configFile, workDir, args)
}

func runRestore(ctx context.Context, commandArgs []string, configFile, workDir string, args []string) error {
	dotnetArgs := append([]string(nil), commandArgs...)
	if configFile != "" {
		configPath := ResolveConfigPath(configFile)
		if configPath == "" {
			return fmt.Errorf("config file not found: %s", configFile)
		}
		dotnetArgs = append(dotnetArgs, "--configfile", configPath)
	}
	dotnetArgs = append(dotnetArgs, args...)
	providerPath, cleanup, err := ProvideCredentialProvider(ctx, workDir)
	if err != nil {
		return fmt.Errorf("failed to prepare temporary credential provider: %w", err)
	}
	defer cleanup()

	logger.Info("Running: dotnet ", "args", dotnetArgs)
	dotnetCmd := exec.CommandContext(ctx, "dotnet", dotnetArgs...)
	dotnetCmd.Env = os.Environ()
	for _, name := range []string{"NUGET_PLUGIN_PATHS", "NUGET_NETCORE_PLUGIN_PATHS"} {
		paths := os.Getenv(name)
		if paths != "" {
			paths = strings.TrimRight(paths, ";") + ";"
		}
		dotnetCmd.Env = append(dotnetCmd.Env, name+"="+paths+providerPath)
	}
	dotnetCmd.Stdout = os.Stdout
	dotnetCmd.Stderr = os.Stderr
	dotnetCmd.Stdin = os.Stdin
	if err := dotnetCmd.Run(); err != nil {
		return fmt.Errorf("dotnet %s failed: %w", strings.Join(commandArgs, " "), err)
	}
	return nil
}
