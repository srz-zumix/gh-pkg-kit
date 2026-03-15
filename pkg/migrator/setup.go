package migrator

import (
	"fmt"
	"os"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

// ClientsAndRepositories holds the result of setting up source and destination clients and repositories.
type ClientsAndRepositories struct {
	SrcRepo    repository.Repository
	DestRepo   repository.Repository
	SrcClient  *gh.GitHubClient
	DestClient *gh.GitHubClient
}

// SetupClientsAndRepositories initializes source and destination repositories and GitHub clients.
// It handles:
// - Resolving source repository as [host/]owner
// - Resolving destination repository as [host/]owner[/repo]
// - Defaulting destination host to source host
// - Creating GitHub clients with token fallback from environment variables
func SetupClientsAndRepositories(src, dst, srcToken, dstToken string) (*ClientsAndRepositories, error) {
	// Get tokens from environment if not provided
	if srcToken == "" {
		srcToken = os.Getenv("GH_SRC_TOKEN")
	}
	if dstToken == "" {
		dstToken = os.Getenv("GH_DST_TOKEN")
	}

	// Parse source repository as [HOST/]OWNER
	srcRepo, err := parser.Repository(parser.RepositoryOwnerWithHost(src))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve source owner: %w", err)
	}

	// Parse destination repository as [HOST/]OWNER[/REPO]
	destRepo, err := parser.Repository(parser.RepositoryOwnerOrRepo(dst))
	if err != nil {
		return nil, fmt.Errorf("failed to parse destination repository: %w", err)
	}

	// Default dest host to source host
	if destRepo.Host == "" {
		destRepo.Host = srcRepo.Host
	}

	// Create source client
	var srcClient *gh.GitHubClient
	if srcToken != "" {
		srcClient, err = gh.NewGitHubClientWithToken(srcRepo, srcToken)
		if err != nil {
			return nil, fmt.Errorf("failed to create source client: %w", err)
		}
	} else {
		srcClient, err = gh.NewGitHubClientWithRepo(srcRepo)
		if err != nil {
			return nil, fmt.Errorf("failed to create source client: %w", err)
		}
	}

	// Create destination client
	var destClient *gh.GitHubClient
	if dstToken != "" {
		destClient, err = gh.NewGitHubClientWithToken(destRepo, dstToken)
		if err != nil {
			return nil, fmt.Errorf("failed to create destination client: %w", err)
		}
	} else {
		destClient, err = gh.NewGitHubClientWithRepo(destRepo)
		if err != nil {
			return nil, fmt.Errorf("failed to create destination client: %w", err)
		}
	}

	return &ClientsAndRepositories{
		SrcRepo:    srcRepo,
		DestRepo:   destRepo,
		SrcClient:  srcClient,
		DestClient: destClient,
	}, nil
}
