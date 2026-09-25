# dflow Makefile

# Variables
BINARY_NAME = dflow
BIN_DIR = bin

# 🏷️  Channel marker only. A tag is injected when HEAD is exactly on it; every
# other revision keeps the `dev` marker, so a development tree never claims a
# released version. The installed commit is deliberately NOT injected here: the
# binary self-reports it from its own VCS stamp (buildvcs stays on), which keeps
# one source of truth for the revision.
VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null || echo dev)

# Default target
.PHONY: all
all: build

# 👷 Build local con version inyectada vía ldflags
.PHONY: build
build:
	@echo "🔨 Building $(BINARY_NAME) (version: $(VERSION))..."
	go build -ldflags="-X main.version=$(VERSION)" -o $(BIN_DIR)/$(BINARY_NAME) .

# 📦 Cross-platform build (for testing outside GoReleaser)
.PHONY: build-all
build-all:
	@echo "🧪 Building cross-platform binaries..."
	GOOS=linux   GOARCH=amd64 go build -ldflags="-X main.version=$(VERSION)" -o $(BIN_DIR)/$(BINARY_NAME)-linux .
	GOOS=darwin  GOARCH=amd64 go build -ldflags="-X main.version=$(VERSION)" -o $(BIN_DIR)/$(BINARY_NAME)-darwin .
	GOOS=windows GOARCH=amd64 go build -ldflags="-X main.version=$(VERSION)" -o $(BIN_DIR)/$(BINARY_NAME).exe .

# 🚀 Release with GoReleaser + .env token. The GitHub release body is only the
# latest version's section of CHANGELOG.md, extracted to RELEASE_NOTES.md (the
# dflow update --check digest renders the release body, so old sections do not
# belong in it). The notes step fails fast, so an unpublishable release body
# stops the release before GoReleaser runs. The release step then preflights
# .env/GITHUB_TOKEN and refuses to run off-tag or when the newest CHANGELOG.md
# section does not belong to the tag being published, so a stale changelog can
# never overwrite an already correct release body.
.PHONY: release release-notes
release: release-notes
	@set -eu; \
	fail() { echo "❌  $$1" >&2; exit 1; }; \
	[ -f .env ] \
		|| fail ".env is missing: GoReleaser loads GITHUB_TOKEN from it, so create it from .env.example before releasing."; \
	[ -r .env ] \
		|| fail ".env exists but cannot be read, so it cannot be sourced: check its permissions before releasing."; \
	set -a; . ./.env || fail "Could not source .env: fix its shell syntax before releasing."; set +a; \
	[ -n "$${GITHUB_TOKEN:-}" ] \
		|| fail ".env defines no GITHUB_TOKEN: GoReleaser needs it to publish the GitHub release."; \
	[ "$(VERSION)" != dev ] \
		|| fail "make release needs a tag, but HEAD is not on an exact tag (VERSION=dev): check out the release commit and create its tag first."; \
	notes_version=$$(head -n 1 RELEASE_NOTES.md | awk '{print $$3}'); \
	[ "$$notes_version" = "$(VERSION)" ] \
		|| fail "RELEASE_NOTES.md carries the $$notes_version section but the tag being released is $(VERSION): curate CHANGELOG.md so its newest '## 📦' section matches the tag."; \
	echo "🚀 Running GoReleaser with .env"; \
	goreleaser release --clean --release-notes=RELEASE_NOTES.md

# 📝 Extract the latest version section from CHANGELOG.md into RELEASE_NOTES.md
# (generated, gitignored, never committed). The extraction is validated before
# the release body is written: an empty or heading-only result would publish a
# GitHub release with no body, so it goes to a temporary file and only lands as
# RELEASE_NOTES.md once it holds a version section with content. A rejected run
# removes both the temporary file and any pre-existing RELEASE_NOTES.md, so a
# failure never leaves a release body that a bare GoReleaser invocation could
# publish without re-validating it.
release-notes:
	@set -eu; \
	tmp=RELEASE_NOTES.md.tmp; \
	fail() { rm -f "$$tmp" RELEASE_NOTES.md; echo "❌  $$1" >&2; exit 1; }; \
	awk '/^## 📦/{if (found) exit; found=1} found' CHANGELOG.md > "$$tmp" \
		|| fail "Could not extract the latest section from CHANGELOG.md."; \
	head -n 1 "$$tmp" | grep -q '^## 📦' \
		|| fail "CHANGELOG.md carries no '## 📦' version section; there is nothing to publish as the release body."; \
	awk 'NR > 1 && NF { found = 1 } END { exit !found }' "$$tmp" \
		|| fail "The latest '## 📦' version section in CHANGELOG.md has no content below its heading."; \
	mv "$$tmp" RELEASE_NOTES.md; \
	echo "📝 RELEASE_NOTES.md written from the latest CHANGELOG.md section (generated, never committed)."

# 📥 Install local build to $GOPATH/bin with version injected
.PHONY: install
install:
	@INSTALL_PATH=$$(go env GOPATH)/bin; \
	echo "📥 Installing $(BINARY_NAME) to $$INSTALL_PATH (version: $(VERSION))..."; \
	go build -ldflags="-X main.version=$(VERSION)" -o $$INSTALL_PATH/$(BINARY_NAME) .

# 🧪 Tests
.PHONY: test
test:
	go test ./...

# 🔎 Linter (requires golangci-lint)
.PHONY: lint
lint:
	GOCACHE=/tmp/dflow-gocache GOLANGCI_LINT_CACHE=/tmp/dflow-golangci-lint-cache golangci-lint run ./...

# 📝 Generate the changelog draft from Conventional Commits with git-cliff.
# The draft is curated by hand into CHANGELOG.md (see RELEASING.md, step 3)
# and never committed.
.PHONY: changelog
changelog:
	@command -v git-cliff >/dev/null 2>&1 || { \
		echo "❌  git-cliff is not installed. Install it first: https://git-cliff.org/docs/installation"; \
		exit 1; \
	}
	@echo "📝 Generating CHANGELOG.draft.md from Conventional Commits..."
	@git cliff --unreleased --config cliff.toml -o CHANGELOG.draft.md
	@echo "✅ Draft ready. Curate it into CHANGELOG.md, then discard it: it is never committed."

# 🧹 Clean compiled binaries
.PHONY: clean
clean:
	@echo "🧹 Cleaning binaries..."
	rm -rf $(BIN_DIR)
