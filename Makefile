EXTENSION_NAME=pkg-kit
NUGET_TESTDATA_DIR=testdata/nuget/SampleNuGetPkg
GITHUB_OWNER?=$(shell gh api user --jq .login)
GITHUB_ACTOR?=$(shell gh api user --jq .login)
GITHUB_TOKEN?=$(shell gh auth token)

help: ## Display this help screen
	@grep -E '^[a-zA-Z][a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sed -e 's/^GNUmakefile://' | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

install: ## install gh extension
	gh extension remove "srz-zumix/gh-${EXTENSION_NAME}" || :
	gh extension remove "${EXTENSION_NAME}" || :
	gh extension install .


install-released:
	gh extension remove "${EXTENSION_NAME}" || :
	gh extension install "srz-zumix/gh-${EXTENSION_NAME}"

build:
	go build -o gh-${EXTENSION_NAME}

test: ## run tests
	go test -v ./...

clean:
	rm -f go.work

go-work:
	# (cd .. && gh repo clone srz-zumix/go-gh-extension)
	ln -snf ../go-gh-extension go-gh-extension
	# go work init
	go work use .
	go work use ./go-gh-extension
	go work sync

testdata-nuget-pack: ## Build the sample NuGet package (.nupkg)
	dotnet pack $(NUGET_TESTDATA_DIR) -c Release

testdata-nuget-publish: testdata-nuget-pack ## Build and publish the sample NuGet package to GitHub Packages
	@dotnet nuget push $(NUGET_TESTDATA_DIR)/bin/Release/SampleNuGetPkg.*.nupkg \
		--source "https://nuget.pkg.github.com/$(GITHUB_OWNER)/index.json" \
		--api-key "$(GITHUB_TOKEN)"
