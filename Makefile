TOOLS_DIR := .tools
GO_VERSION := 1.27.1
# PaperMod v8.0 was released alongside Hugo 0.134.x.
HUGO_VERSION := 0.134.3
GO_ROOT := $(TOOLS_DIR)/go
GO_BIN := $(GO_ROOT)/bin/go
HUGO ?= $(TOOLS_DIR)/bin/hugo
HUGO_VERSION_FILE := $(TOOLS_DIR)/hugo-version
THEME_DIR := themes/PaperMod
PAPERMOD_VERSION := v8.0
PAPERMOD_REPOSITORY := https://github.com/adityatelange/hugo-PaperMod.git
GOOS := $(shell uname -s | tr '[:upper:]' '[:lower:]')
GOARCH := $(shell uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/')
GO_ARCHIVE := go$(GO_VERSION).$(GOOS)-$(GOARCH).tar.gz
GO_DOWNLOAD_URL := https://go.dev/dl/$(GO_ARCHIVE)
ifeq ($(GOOS),darwin)
HUGO_PLATFORM := darwin-universal
else
HUGO_PLATFORM := $(GOOS)-$(GOARCH)
endif
HUGO_ARCHIVE := hugo_extended_$(HUGO_VERSION)_$(HUGO_PLATFORM).tar.gz
HUGO_DOWNLOAD_URL := https://github.com/gohugoio/hugo/releases/download/v$(HUGO_VERSION)/$(HUGO_ARCHIVE)

.PHONY: setup theme check-hugo serve build clean

# Downloads tools into .tools/; it never changes the system-wide Go installation.
setup:
	@set -eu; \
	command -v curl >/dev/null || { echo "curl is required for setup."; exit 1; }; \
	command -v tar >/dev/null || { echo "tar is required for setup."; exit 1; }; \
	case "$(GOOS)/$(GOARCH)" in \
		linux/amd64|linux/arm64|darwin/amd64|darwin/arm64) ;; \
		*) echo "Unsupported platform: $(GOOS)/$(GOARCH). Install Hugo Extended manually and run make HUGO=/path/to/hugo serve."; exit 1 ;; \
	esac; \
	mkdir -p "$(TOOLS_DIR)/bin" "$(TOOLS_DIR)/downloads"; \
	if [ ! -x "$(GO_BIN)" ]; then \
		echo "Downloading Go $(GO_VERSION) for $(GOOS)/$(GOARCH)..."; \
		curl --fail --location --retry 3 --output "$(TOOLS_DIR)/downloads/$(GO_ARCHIVE)" "$(GO_DOWNLOAD_URL)"; \
		rm -rf "$(GO_ROOT)"; \
		tar -C "$(TOOLS_DIR)" -xzf "$(TOOLS_DIR)/downloads/$(GO_ARCHIVE)"; \
	fi
	@set -eu; \
	if [ ! -x "$(HUGO)" ] || [ ! -f "$(HUGO_VERSION_FILE)" ] || [ "$$(cat "$(HUGO_VERSION_FILE)")" != "$(HUGO_VERSION)" ]; then \
		echo "Downloading Hugo Extended $(HUGO_VERSION) for $(HUGO_PLATFORM)..."; \
		curl --fail --location --retry 3 --output "$(TOOLS_DIR)/downloads/$(HUGO_ARCHIVE)" "$(HUGO_DOWNLOAD_URL)"; \
		rm -f "$(HUGO)"; \
		tar -C "$(TOOLS_DIR)/bin" -xzf "$(TOOLS_DIR)/downloads/$(HUGO_ARCHIVE)" hugo; \
		printf '%s\n' "$(HUGO_VERSION)" > "$(HUGO_VERSION_FILE)"; \
	fi
	@$(MAKE) theme
	@"$(HUGO)" version

theme:
	@test -d "$(THEME_DIR)/.git" || git clone --depth 1 --branch "$(PAPERMOD_VERSION)" "$(PAPERMOD_REPOSITORY)" "$(THEME_DIR)"
	@git -C "$(THEME_DIR)" fetch --depth 1 origin "$(PAPERMOD_VERSION)"
	@git -C "$(THEME_DIR)" checkout --detach "$(PAPERMOD_VERSION)"

check-hugo:
	@test -x "$(HUGO)" || { echo "Hugo Extended is not installed. Run 'make setup' first."; exit 1; }

serve: check-hugo theme
	$(HUGO) server --buildDrafts --disableFastRender

build: check-hugo theme
	$(HUGO) --gc --minify

clean:
	rm -rf public resources
