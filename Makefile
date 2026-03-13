# Copyright (c) 2026 Michael Lechner
# Licensed under the MIT License. See LICENSE file in the project root for full license information.

VERSION=1.1.0
LDFLAGS=-s -w -X main.version=$(VERSION)

.PHONY: build run run-verbose test tidy clean release help

# Build all binaries (standard)
build:
	@mkdir -p bin
	go build -o bin/tui ./cmd/tui

# Run the TUI application
run:
	go run ./cmd/tui $(ARGS)

# Run the TUI application with all details enabled
run-verbose:
	go run ./cmd/tui -v --all $(ARGS)

# Run all tests
test:
	go test -v ./...

# Tidy Go modules
tidy:
	go mod tidy

# Clean binaries and dist folder
clean:
	rm -rf bin dist

# Prepare a release package in the dist folder (Optimized build)
release: clean
	@mkdir -p bin
	@echo "Building optimized binary for version $(VERSION)..."
	go build -ldflags="$(LDFLAGS)" -trimpath -o bin/tui ./cmd/tui
	@rm -rf dist
	@mkdir -p dist/man
	@cp bin/tui dist/gitpulse
	@cp docs/gitpulse.1 dist/man/
	@cp README.md README-de.md LICENSE dist/
	@mkdir -p dist/config && cp config/repos.ini.example dist/config/
	@mkdir -p dist/docs && cp docs/json_schema.md docs/schema.json dist/docs/
	@echo "Release prepared in ./dist folder"

# Show help
help:
	@echo "GitPulseMLC Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make build         Build all binaries"
	@echo "  make run           Run the TUI application (use ARGS=\"--flag\" for options)"
	@echo "  make run-verbose   Run the TUI with -v and --all enabled"
	@echo "  make test          Run all tests"
	@echo "  make tidy          Tidy Go modules"
	@echo "  make clean         Remove binaries and dist folder"
	@echo "  make release       Prepare a release package in ./dist (Optimized)"
