BINARY  := shdw
PREFIX  ?= /usr/local/bin
MODULE  := github.com/leihog/shdw

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags="-s -w -X $(MODULE)/cmd.version=$(VERSION)"

# Default build for current platform
.PHONY: build
build:
	go build $(LDFLAGS) -o $(BINARY) .

# Build for Linux amd64 (cross-compile from macOS)
.PHONY: build-linux
build-linux:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY)-linux-amd64 .

# Build for Linux arm64 (e.g. Raspberry Pi, AWS Graviton)
.PHONY: build-linux-arm
build-linux-arm:
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY)-linux-arm64 .

# Build all targets
.PHONY: build-all
build-all: build build-linux build-linux-arm

# Install to PREFIX (default: /usr/local/bin)
# Override with: make install PREFIX=~/bin
.PHONY: install
install: build
	@mkdir -p $(PREFIX)
	cp $(BINARY) $(PREFIX)/$(BINARY)
	@echo "Installed $(VERSION) to $(PREFIX)/$(BINARY)"

.PHONY: clean
clean:
	rm -f $(BINARY) $(BINARY)-linux-amd64 $(BINARY)-linux-arm64
