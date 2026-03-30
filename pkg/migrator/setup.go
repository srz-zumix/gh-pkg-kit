package migrator

import (
	"fmt"
	"os"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
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

