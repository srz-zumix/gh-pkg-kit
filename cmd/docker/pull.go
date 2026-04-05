package docker

import (
	"github.com/spf13/cobra"
	containerCmd "github.com/srz-zumix/gh-pkg-kit/cmd/container"
)

// NewPullCmd creates a command to pull a docker image from the legacy Docker Package Registry to a local tarball.
// --owner requires [host/]owner/repo because the legacy registry image path includes the repository name.
func NewPullCmd() *cobra.Command {
	return containerCmd.NewPullCmdFor("docker", true)
}
