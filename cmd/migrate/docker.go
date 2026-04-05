package migrate

import (
	"github.com/spf13/cobra"
)

// NewDockerCmd creates a command to migrate docker packages between owners
func NewDockerCmd() *cobra.Command {
	return newContainerMigrateCmd("docker", true)
}
