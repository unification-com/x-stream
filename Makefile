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

lint:
	golangci-lint run
	@find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" | xargs gofmt -w -s
	go mod verify

gofmt:
	@find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" | xargs gofmt -w -s
	go mod verify

.PHONY: lint gofmt
