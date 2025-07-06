# Makefile for building and releasing dflow

BINARY_NAME=dflow
DIST_DIR=dist

.PHONY: clean build release

## clean: remove dist directory
clean:
	rm -rf $(DIST_DIR)

## build: build binaries for all target platforms
build: clean
	@mkdir -p $(DIST_DIR)
	GOOS=linux   GOARCH=amd64   go build -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64
	GOOS=darwin  GOARCH=arm64   go build -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64
	GOOS=windows GOARCH=amd64   go build -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe

## release: build binaries and show GitHub release command
release: build
	@echo "🎉 Binaries ready in ./dist"
	@echo "To create GitHub release:"
	@echo "  gh release create vX.Y.Z ./dist/* --title 'vX.Y.Z' --notes-file CHANGELOG.md"
