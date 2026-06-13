# Revv Makefile
# Supporting build, test, clean targets

BINARY_NAME := revv
BINARY_DIR  := bin
MODULE      := github.com/vipinsingh/revv

VERSION := $(shell git describe --tags --always 2>/dev/null || echo "dev")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

LDFLAGS := -ldflags "-X $(MODULE)/internal/cli.version=$(VERSION) -X $(MODULE)/internal/cli.commit=$(COMMIT) -X $(MODULE)/internal/cli.date=$(DATE)"

.PHONY: build test clean all

all: test build

build:
	@mkdir -p $(BINARY_DIR)
	go build $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME) ./cmd/revv

test:
	go test ./...

clean:
	rm -rf $(BINARY_DIR)
