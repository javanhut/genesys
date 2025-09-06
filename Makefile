.PHONY: all build build-opt build-prod clean install install-local uninstall test race install-completions uninstall-completions

# Installation directory
PREFIX ?= /usr/local
BINDIR = $(PREFIX)/bin
BASH_COMPLETION_DIR ?= $(PREFIX)/share/bash-completion/completions
ZSH_COMPLETION_DIR ?= $(PREFIX)/share/zsh/site-functions
FISH_COMPLETION_DIR ?= $(PREFIX)/share/fish/vendor_completions.d

# Build flags
GOFLAGS ?= 
LDFLAGS := -s -w
CGO_ENABLED ?= 0

# Shell detection is done dynamically in install target

# Default target
all: build

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

# Install genesys to system
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
	@echo "Genesys installed successfully!"
	@echo "Shell completions have been installed."
	@echo ""
	@DETECTED_SHELL=$$(ps -p $$$$ -o comm= 2>/dev/null | sed 's/^-//' || basename "$$SHELL" 2>/dev/null || echo "unknown"); \
	if [ "$$DETECTED_SHELL" = "bash" ]; then \
		echo "To enable bash completions, add this to your ~/.bashrc:"; \
		echo "  source $(BASH_COMPLETION_DIR)/genesys"; \
	elif [ "$$DETECTED_SHELL" = "zsh" ]; then \
		echo "To enable zsh completions, add this to your ~/.zshrc:"; \
		echo "  autoload -U compinit && compinit"; \
		echo "  compdef _genesys genesys"; \
	elif [ "$$DETECTED_SHELL" = "fish" ]; then \
		echo "Fish completions should be automatically loaded."; \
	else \
		echo "Shell completions installed for bash, zsh, and fish."; \
		echo "Current shell: $$DETECTED_SHELL"; \
		echo ""; \
		echo "To enable completions:"; \
		echo "  Bash: source $(BASH_COMPLETION_DIR)/genesys"; \
		echo "  Zsh:  autoload -U compinit && compinit && compdef _genesys genesys"; \
		echo "  Fish: completions should auto-load"; \
	fi

# Install locally to user's home directory (no sudo required)
install-local: build-opt
	@echo "Installing genesys to user directory..."
	mkdir -p $(HOME)/.local/bin
	install -m 755 genesys $(HOME)/.local/bin/genesys
	rm -f genesys
	@echo ""
	@echo "Genesys installed to $(HOME)/.local/bin/genesys"
	@echo "Make sure $(HOME)/.local/bin is in your PATH"
	@echo ""
	@echo "Shell completions not installed with local install."
	@echo "For completions, run: sudo make install"

# Install shell completions
install-completions:
	@echo "Installing shell completions..."
	# Bash completions
	install -d $(BASH_COMPLETION_DIR)
	install -m 644 autocomplete/genesys-completion.bash $(BASH_COMPLETION_DIR)/genesys
	# Zsh completions
	install -d $(ZSH_COMPLETION_DIR)
	install -m 644 autocomplete/genesys-completion.zsh $(ZSH_COMPLETION_DIR)/_genesys
	# Fish completions
	install -d $(FISH_COMPLETION_DIR)
	install -m 644 autocomplete/genesys-completion.fish $(FISH_COMPLETION_DIR)/genesys.fish

# Uninstall genesys from system
uninstall: uninstall-completions
	rm -f $(BINDIR)/genesys
	@echo "Genesys uninstalled successfully!"

# Uninstall shell completions
uninstall-completions:
	@echo "Removing shell completions..."
	rm -f $(BASH_COMPLETION_DIR)/genesys
	rm -f $(ZSH_COMPLETION_DIR)/_genesys
	rm -f $(FISH_COMPLETION_DIR)/genesys.fish

# Run tests
test:
	go test ./...

# Run with race detection
race:
	go build -race -o genesys ./cmd/genesys