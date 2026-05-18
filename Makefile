#!/usr/bin/make -f

DOCKER := $(shell which docker)

include scripts/makefiles/proto.mk
include scripts/makefiles/unittests.mk

all: lint test

###############################################################################
###                                  Build                                  ###
###############################################################################

go.sum: go.mod
	@echo "--> Ensure dependencies have not been modified"
	go mod verify

build:
	go build ./...

clean:
	rm -rf coverage.txt

.PHONY: all build clean

###############################################################################
###                                  Lint                                   ###
###############################################################################

# Read-only checks: pass/fail without modifying the tree. Suitable for CI and
# pre-push hooks. `lint-fix` and `fmt` are the local-write counterparts.
lint: fmt-check
	golangci-lint run --timeout=5m
	go mod verify

lint-fix: fmt
	golangci-lint run --fix --timeout=5m
	go mod verify

# Auto-format every .go file in the tree (write).
fmt:
	@find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./api/*" | xargs gofmt -w -s

# Report files that would change under `gofmt -s` and fail if any. CI gate.
fmt-check:
	@out=$$(find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./api/*" | xargs gofmt -l -s); \
	if [ -n "$$out" ]; then \
		echo "files need gofmt -s:"; echo "$$out"; exit 1; \
	fi

# Backwards-compat alias for the previous target name.
gofmt: fmt

.PHONY: lint lint-fix fmt fmt-check gofmt
