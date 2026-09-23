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

# 🚀 Release with GoReleaser + .env token
.PHONY: release
release:
	@echo "🚀 Running GoReleaser with .env"
	@set -a; . ./.env; set +a; goreleaser release --clean --release-notes=CHANGELOG.md

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

# 📝 Generate changelog from last tag
.PHONY: changelog
changelog:
	@echo "⚠️  CHANGELOG.md is maintained manually for each release."
	@echo "📝 Generating CHANGELOG.draft.md from commits..."
	@echo "# Changelog Draft\n" > CHANGELOG.draft.md
	@git log $$(git describe --tags --abbrev=0)..HEAD --pretty=format:"- %s" >> CHANGELOG.draft.md
	@echo "\n✅ Done. Review CHANGELOG.draft.md and copy only the notes you need into CHANGELOG.md"

# 🧹 Clean compiled binaries
.PHONY: clean
clean:
	@echo "🧹 Cleaning binaries..."
	rm -rf $(BIN_DIR)
