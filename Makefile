EXTENSION_NAME=pkg-kit
NUGET_TESTDATA_DIR=testdata/nuget/SampleNuGetPkg
NPM_TESTDATA_DIR=testdata/npm/sample-npm-pkg
RUBYGEMS_TESTDATA_DIR=testdata/rubygems/sample-rubygems-pkg
MAVEN_TESTDATA_DIR=testdata/maven/sample-maven-pkg
GITHUB_OWNER?=$(shell gh api user --jq .login)
GITHUB_ACTOR?=$(shell gh api user --jq .login)
GITHUB_TOKEN?=$(shell gh auth token)
GITHUB_REPO?=gh-pkg-kit

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

testdata-npm-publish: ## Publish the sample npm package to GitHub Packages
	cd $(NPM_TESTDATA_DIR) && \
		PACKAGE_NAME=$$(node -p "require('./package.json').name") && \
		PACKAGE_SCOPE=$$(printf '%s\n' "$$PACKAGE_NAME" | sed -n 's/^@\([^/]*\)\/.*$$/\1/p') && \
		if [ -n "$$PACKAGE_SCOPE" ] && [ "$$PACKAGE_SCOPE" != "$(GITHUB_OWNER)" ]; then \
			printf '%s\n' "Error: package scope '$$PACKAGE_SCOPE' does not match GITHUB_OWNER '$(GITHUB_OWNER)'" >&2; \
			exit 1; \
		fi && \
		GITHUB_OWNER=$(GITHUB_OWNER) GITHUB_TOKEN=$(GITHUB_TOKEN) npm publish

testdata-rubygems-pack: ## Build the sample RubyGems package (.gem)
	cd $(RUBYGEMS_TESTDATA_DIR) && gem build sample-rubygems-pkg.gemspec

testdata-rubygems-publish: testdata-rubygems-pack ## Build and publish the sample RubyGems package to GitHub Packages
	@TMP_HOME=$$(mktemp -d); \
		trap 'rm -rf "$$TMP_HOME"' EXIT INT TERM; \
		mkdir -p "$$TMP_HOME/.local/share/gem"; \
		printf -- "---\n:github: Bearer $(GITHUB_TOKEN)\n" > "$$TMP_HOME/.local/share/gem/credentials"; \
		chmod 0600 "$$TMP_HOME/.local/share/gem/credentials"; \
		cd $(RUBYGEMS_TESTDATA_DIR) && HOME="$$TMP_HOME" gem push \
			--key github \
			--host https://rubygems.pkg.github.com/$(GITHUB_OWNER) \
			sample-rubygems-pkg-*.gem

testdata-maven-pack: ## Build the sample Maven package (.jar)
	cd $(MAVEN_TESTDATA_DIR) && mvn package -s settings.xml

testdata-maven-publish: testdata-maven-pack ## Build and publish the sample Maven package to GitHub Packages
	cd $(MAVEN_TESTDATA_DIR) && \
		GITHUB_ACTOR=$(GITHUB_ACTOR) GITHUB_TOKEN=$(GITHUB_TOKEN) \
		mvn deploy -s settings.xml \
			-DGITHUB_OWNER=$(GITHUB_OWNER) \
			-DGITHUB_REPO=$(GITHUB_REPO)
