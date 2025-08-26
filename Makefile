.PHONY: all build build-fast build-prod clean install uninstall test race

# Installation directory
PREFIX ?= /usr/local
BINDIR = $(PREFIX)/bin

# Build flags
GOFLAGS ?= 
LDFLAGS := -s -w
CGO_ENABLED ?= 0

# Default target
all: build

# Standard build with optimizations
build:
	CGO_ENABLED=$(CGO_ENABLED) go build -ldflags="$(LDFLAGS)" -trimpath -o genesys ./cmd/genesys

# Fast build for development (no optimizations)
build-fast:
	go build -o genesys ./cmd/genesys

# Production build with maximum optimizations
build-prod:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="$(LDFLAGS) -extldflags '-static'" -trimpath -o genesys ./cmd/genesys

# Clean up binaries
clean:
	rm -f genesys

# Install genesys to system
install: build
	install -d $(BINDIR)
	install -m 755 genesys $(BINDIR)/genesys

# Uninstall genesys from system
uninstall:
	rm -f $(BINDIR)/genesys

# Run tests
test:
	go test ./...

# Run with race detection
race:
	go build -race -o genesys ./cmd/genesys