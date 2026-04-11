package migrator

import (
	"context"
	"fmt"
	"os"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
)

// ClientsAndRepositories holds the result of setting up source and destination clients and repositories.
type ClientsAndRepositories struct {
	SrcRepo    repository.Repository
	DestRepo   repository.Repository
	SrcClient  *gh.GitHubClient
	DestClient *gh.GitHubClient
}

// newClient creates a GitHub client for the given repository, using the token if provided.
func newClient(repo repository.Repository, token string) (*gh.GitHubClient, error) {
	if token != "" {
		return gh.NewGitHubClientWithToken(repo, token)
	}
	return gh.NewGitHubClientWithRepo(repo)
}

// SetupClients creates GitHub clients for already-resolved repositories,
// applying token fallback from environment variables and defaulting dest host to src host.
func SetupClients(srcRepo, destRepo repository.Repository, srcToken, dstToken string) (*ClientsAndRepositories, error) {
	if srcToken == "" {
		srcToken = os.Getenv("GH_SRC_TOKEN")
	}
	if dstToken == "" {
		dstToken = os.Getenv("GH_DST_TOKEN")
	}

	if destRepo.Host == "" {
		destRepo.Host = srcRepo.Host
	}

	srcClient, err := newClient(srcRepo, srcToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create source client: %w", err)
	}

	destClient, err := newClient(destRepo, dstToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination client: %w", err)
	}

	return &ClientsAndRepositories{
		SrcRepo:    srcRepo,
		DestRepo:   destRepo,
		SrcClient:  srcClient,
		DestClient: destClient,
	}, nil
}

// ResolveDestRepo resolves the destination repository name from the source package when it is not specified.
// If clients.DestRepo.Name is already set, this is a no-op.
// Otherwise, it looks up the source package via the GitHub API and uses its associated repository name.
func ResolveDestRepo(ctx context.Context, clients *ClientsAndRepositories, packageType, packageName string) error {
	if clients.DestRepo.Name != "" {
		return nil
	}
	srcOwnerType, err := gh.DetectOwnerType(ctx, clients.SrcClient, clients.SrcRepo.Owner)
	if err != nil {
		return fmt.Errorf("failed to detect source owner type: %w", err)
	}
	pkg, err := gh.GetPackageByOwnerType(ctx, clients.SrcClient, srcOwnerType, clients.SrcRepo.Owner, packageType, packageName)
	if err != nil {
		return fmt.Errorf("failed to get source package '%s': %w", packageName, err)
	}
	repo := pkg.GetRepository()
	if repo == nil {
		return fmt.Errorf("source package '%s' is not associated with any repository", packageName)
	}
	repoName := repo.GetName()
	if repoName == "" {
		return fmt.Errorf("source package '%s' has an empty repository name", packageName)
	}
	clients.DestRepo.Name = repoName
	logger.Info("Resolved destination repository name from source package", "package", packageName, "repo", repoName)
	return nil
}
