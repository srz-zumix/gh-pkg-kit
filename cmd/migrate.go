package cmd

import (
	"github.com/spf13/cobra"
	migrateCmd "github.com/srz-zumix/gh-pkg-kit/cmd/migrate"
)

func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate packages between owners",
		Long:  `Migrate GitHub Packages from one owner (org or user) to another. Each package type has its own subcommand because the migration method differs by type.`,
	}
	cmd.AddCommand(migrateCmd.NewContainerCmd())
	cmd.AddCommand(migrateCmd.NewDockerCmd())
	cmd.AddCommand(migrateCmd.NewNuGetCmd())
	return cmd
}

func init() {
	rootCmd.AddCommand(newMigrateCmd())
}
