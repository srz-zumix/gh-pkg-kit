package migrate

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-pkg-kit/pkg/packages"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

// NewContainerCmd creates a command to migrate container packages between owners
func NewContainerCmd() *cobra.Command {
	return newContainerMigrateCmd("container")
}

// newContainerMigrateCmd creates a migrate command for the given container-based package type.
func newContainerMigrateCmd(packageType string) *cobra.Command {
	var (
		from       string
		to         string
		deleteFlag bool
		dryRun     bool
		versionIDs []int64
		latest     int
		since      string
		until      string
	)

	cmd := &cobra.Command{
		Use:   packageType + " <package-name>",
		Short: "Migrate " + packageType + " packages between owners",
		Long: fmt.Sprintf(`Migrates %s packages from one owner to another within GitHub Packages.
Uses the OCI Distribution API to copy image manifests and blobs, including multi-architecture images.
The source owner is resolved from the current repository if --from is not specified.
The source and destination owner types (organization or user) are detected automatically.`, packageType),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			srcPackage := args[0]

			repo, err := parser.Repository(parser.RepositoryOwnerWithHost(from))
			if err != nil {
				return fmt.Errorf("failed to resolve source owner: %w", err)
			}
			srcOwner := repo.Owner

			dest, err := parser.ParsePackageRef(to, srcPackage)
			if err != nil {
				return err
			}

			// Default dest host to source host
			destHost := dest.Host
			if destHost == "" {
				destHost = repo.Host
			}

			srcClient, destClient, err := gh.NewGitHubClientWith2Hosts(repo.Host, destHost)
			if err != nil {
				return err
			}
			ctx := cmd.Context()

			return packages.MigrateContainer(ctx, srcClient, destClient, packages.ContainerOptions{
				PackageType: packageType,
				SrcHost:     repo.Host,
				DestHost:    destHost,
				SrcOwner:    srcOwner,
				SrcPackage:  srcPackage,
				DestOwner:   dest.Owner,
				DestPackage: dest.Package,
				DeleteFlag:  deleteFlag,
				DryRun:      dryRun,
				VersionIDs:  versionIDs,
				Latest:      latest,
				Since:       since,
				Until:       until,
			})
		},
	}

	f := cmd.Flags()
	f.StringVar(&from, "from", "", "Source [host/]owner (default: current repository owner)")
	f.StringVar(&to, "to", "", "Destination [host/]owner[/package-name] (host detected by '.' in first segment)")
	_ = cmd.MarkFlagRequired("to")
	f.BoolVar(&deleteFlag, "delete", false, "Delete source versions after successful migration")
	f.BoolVar(&dryRun, "dry-run", false, "Show what would be migrated without performing the migration")
	f.Int64SliceVar(&versionIDs, "version", nil, "Migrate specific version(s) by ID (can be specified multiple times)")
	f.IntVarP(&latest, "latest", "n", 0, "Migrate latest N versions (by creation date)")
	f.StringVar(&since, "since", "", "Migrate versions created on or after this date (RFC3339 or YYYY-MM-DD)")
	f.StringVar(&until, "until", "", "Migrate versions created on or before this date (RFC3339 or YYYY-MM-DD)")

	return cmd
}
