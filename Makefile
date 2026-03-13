BINARY  := shdw
DIST    := dist
PREFIX  ?= /usr/local/bin
MODULE  := github.com/leihog/shdw

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags="-s -w -X $(MODULE)/cmd.version=$(VERSION)"

.PHONY: build
build:
	@mkdir -p $(DIST)
	go build $(LDFLAGS) -o $(DIST)/$(BINARY) .

.PHONY: build-linux
build-linux:
	@mkdir -p $(DIST)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-linux-amd64 .

.PHONY: build-linux-arm
build-linux-arm:
	@mkdir -p $(DIST)
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-linux-arm64 .

.PHONY: build-all
build-all: build build-linux build-linux-arm

# Install to PREFIX (default: /usr/local/bin)
# Override with: make install PREFIX=~/bin
.PHONY: install
install: build
	@mkdir -p $(PREFIX)
	cp $(DIST)/$(BINARY) $(PREFIX)/$(BINARY)
	@echo "Installed $(VERSION) to $(PREFIX)/$(BINARY)"

.PHONY: clean
clean:
	rm -rf $(DIST)
