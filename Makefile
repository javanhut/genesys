.PHONY: all build build-opt build-prod clean help install install-local uninstall uninstall-local test race install-completions install-local-completions uninstall-completions uninstall-local-completions

# Installation directory
PREFIX ?= /usr/local
BINDIR = $(PREFIX)/bin
BASH_COMPLETION_DIR ?= $(PREFIX)/share/bash-completion/completions
ZSH_COMPLETION_DIR ?= $(PREFIX)/share/zsh/site-functions
FISH_COMPLETION_DIR ?= $(PREFIX)/share/fish/vendor_completions.d

# Local installation directory
LOCAL_BINDIR = $(HOME)/.local/bin
LOCAL_BASH_COMPLETION_DIR = $(HOME)/.local/share/bash-completion/completions
LOCAL_ZSH_COMPLETION_DIR = $(HOME)/.local/share/zsh/site-functions
LOCAL_FISH_COMPLETION_DIR = $(HOME)/.local/share/fish/vendor_completions.d

# Build flags
GOFLAGS ?= 
LDFLAGS := -s -w
CGO_ENABLED ?= 0

# Default target
all: build

# Show help information
help:
	@echo "Genesys Build and Installation"
	@echo ""
	@echo "Build targets:"
	@echo "  build      - Fast development build (default)"
	@echo "  build-opt  - Optimized build"
	@echo "  build-prod - Production build with static linking"
	@echo "  clean      - Remove built binaries"
	@echo ""
	@echo "Installation targets:"
	@echo "  install        - Install system-wide (requires sudo)"
	@echo "  install-local  - Install to user directory (no sudo)"
	@echo "  uninstall      - Remove system installation (requires sudo)"
	@echo "  uninstall-local - Remove user installation"
	@echo ""
	@echo "Testing targets:"
	@echo "  test       - Run tests"
	@echo "  race       - Build with race detection"
	@echo ""
	@echo "Quick start:"
	@echo "  make install-local  # Install to ~/.local/bin (recommended)"
	@echo "  sudo make install   # Install system-wide"

# Fast build for development (default - no optimizations)
build:
	go build -o genesys ./cmd/genesys

# Standard build with optimizations
build-opt:
	CGO_ENABLED=$(CGO_ENABLED) go build -ldflags="$(LDFLAGS)" -trimpath -o genesys ./cmd/genesys

# Production build with maximum optimizations
build-prod:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="$(LDFLAGS) -extldflags '-static'" -trimpath -o genesys ./cmd/genesys

# Clean up binaries
clean:
	rm -f genesys

# Install genesys to system (requires sudo)
install: build-opt
	@if [ "$$(id -u)" -ne 0 ]; then \
		echo "Installing to system directories requires root privileges."; \
		echo "Run: sudo make install"; \
		exit 1; \
	fi
	$(MAKE) install-completions
	install -d $(BINDIR)
	install -m 755 genesys $(BINDIR)/genesys
	rm -f genesys
	@echo ""
	@echo "Genesys installed successfully to $(BINDIR)/genesys!"
	@echo "Shell completions installed system-wide."
	@echo "Completions should be automatically available in new shell sessions."

# Install locally to user's home directory (no sudo required)
install-local: build-opt
	@echo "Installing genesys to user directory..."
	mkdir -p $(LOCAL_BINDIR)
	install -m 755 genesys $(LOCAL_BINDIR)/genesys
	rm -f genesys
	$(MAKE) install-local-completions
	@echo ""
	@echo "Genesys installed to $(LOCAL_BINDIR)/genesys!"
	@echo ""
	@if ! echo "$$PATH" | grep -q "$(LOCAL_BINDIR)"; then \
		echo "WARNING: $(LOCAL_BINDIR) is not in your PATH."; \
		echo "Add the following to your shell configuration file:"; \
		echo "  export PATH=\"$(LOCAL_BINDIR):\$$PATH\""; \
		echo ""; \
	fi
	@echo "Shell completions installed to user directories."
	@echo "You may need to restart your shell for completions to work."

# Install shell completions system-wide (requires sudo)
install-completions:
	@echo "Installing system-wide shell completions..."
	# Bash completions
	install -d $(BASH_COMPLETION_DIR)
	install -m 644 autocomplete/genesys-completion.bash $(BASH_COMPLETION_DIR)/genesys
	# Zsh completions
	install -d $(ZSH_COMPLETION_DIR)
	install -m 644 autocomplete/genesys-completion.zsh $(ZSH_COMPLETION_DIR)/_genesys
	# Fish completions
	install -d $(FISH_COMPLETION_DIR)
	install -m 644 autocomplete/genesys-completion.fish $(FISH_COMPLETION_DIR)/genesys.fish

# Install shell completions locally (no sudo required)
install-local-completions:
	@echo "Installing local shell completions..."
	# Bash completions
	mkdir -p $(LOCAL_BASH_COMPLETION_DIR)
	install -m 644 autocomplete/genesys-completion.bash $(LOCAL_BASH_COMPLETION_DIR)/genesys
	# Zsh completions
	mkdir -p $(LOCAL_ZSH_COMPLETION_DIR)
	install -m 644 autocomplete/genesys-completion.zsh $(LOCAL_ZSH_COMPLETION_DIR)/_genesys
	# Fish completions
	mkdir -p $(LOCAL_FISH_COMPLETION_DIR)
	install -m 644 autocomplete/genesys-completion.fish $(LOCAL_FISH_COMPLETION_DIR)/genesys.fish

# Uninstall genesys from system
uninstall: uninstall-completions
	@if [ "$$(id -u)" -ne 0 ]; then \
		echo "Uninstalling from system directories requires root privileges."; \
		echo "Run: sudo make uninstall"; \
		exit 1; \
	fi
	rm -f $(BINDIR)/genesys
	@echo "Genesys uninstalled successfully from system!"

# Uninstall genesys from local user directory
uninstall-local: uninstall-local-completions
	rm -f $(LOCAL_BINDIR)/genesys
	@echo "Genesys uninstalled successfully from $(LOCAL_BINDIR)!"

# Uninstall shell completions from system
uninstall-completions:
	@echo "Removing system-wide shell completions..."
	rm -f $(BASH_COMPLETION_DIR)/genesys
	rm -f $(ZSH_COMPLETION_DIR)/_genesys
	rm -f $(FISH_COMPLETION_DIR)/genesys.fish

# Uninstall shell completions from local user directory
uninstall-local-completions:
	@echo "Removing local shell completions..."
	rm -f $(LOCAL_BASH_COMPLETION_DIR)/genesys
	rm -f $(LOCAL_ZSH_COMPLETION_DIR)/_genesys
	rm -f $(LOCAL_FISH_COMPLETION_DIR)/genesys.fish

# Run tests
test:
	go test ./...

# Run with race detection
race:
	go build -race -o genesys ./cmd/genesys
