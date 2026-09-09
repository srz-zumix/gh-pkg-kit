/*
Copyright © 2025 srz_zumix
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-pkg-kit/version"
	"github.com/srz-zumix/go-gh-extension/pkg/actions"
	"github.com/srz-zumix/go-gh-extension/pkg/cmdflags"
)

var rootCmd = &cobra.Command{
	Use:   "gh-pkg-kit",
	Short: "GitHub Packages operations extension for GitHub CLI",
	Long: `Package-related operations extension for GitHub CLI.

Download package assets (container, docker, gem, maven, npm, nuget),
migrate packages between owners or registries (including legacy
docker.pkg.github.com to ghcr.io), and manage packages and their versions
for organizations and users (list/get/delete/restore).`,
	Version: version.Version,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	if actions.IsRunsOn() {
		rootCmd.SetErrPrefix(actions.GetErrorPrefix())
	}
	cmdflags.AddPersistentFlags(rootCmd)
}
