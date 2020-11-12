# Makefile

PWD := $(shell pwd)
BASE_DIR := $(shell basename $(PWD))
# Keep an existing GOPATH, make a private one if it is undefined
GOPATH_DEFAULT := $(PWD)/.go
export GOPATH ?= $(GOPATH_DEFAULT)
GOBIN_DEFAULT := $(GOPATH)/bin
export GOBIN ?= $(GOBIN_DEFAULT)
export GO111MODULE := on
TESTARGS_DEFAULT := -v -race
TESTARGS ?= $(TESTARGS_DEFAULT)
HAS_GOLANGCI := $(shell command -v golangci-lint;)
HAS_GOIMPORTS := $(shell command -v goimports;)
DIST_DIRS	= find * -type d -exec
TEMP_DIR	:=$(shell mktemp -d)
GOOS		?= $(shell go env GOOS)
VERSION		?= $(shell git describe --tags 2> /dev/null || \
			   git describe --match=$(git rev-parse --short=8 HEAD) --always --dirty --abbrev=8)
GOARCH		:= amd64
LDFLAGS		:= "-w -s -X 'main.Version=${VERSION}'"
CMD_PACKAGE := ./cmd
BINARY 		?= ./proxy

$(GOBIN):
	echo "create gobin"
	mkdir -p $(GOBIN)

work: $(GOBIN)

build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
	-ldflags $(LDFLAGS) \
	-o $(BINARY) \
	$(CMD_PACKAGE)

install: clean check test
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go install \
	-ldflags $(LDFLAGS) \
	$(CMD_PACKAGE)

test: unit

check: work fmt vet goimports golangci
unit: work
	go test -tags=unit $(TESTARGS) ./...

fmt:
	go fmt ./...

goimports:
ifndef HAS_GOIMPORTS
	echo "installing goimports"
	GO111MODULE=off go get golang.org/x/tools/cmd/goimports
endif
	goimports -d $(shell find . -iname "*.go")

vet:
	go vet ./...

golangci:
ifndef HAS_GOLANGCI
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.32.2
endif
	golangci-lint run ./...

cover: work
	go test $(TESTARGS) -tags=unit -cover -coverpkg=./ ./...

shell:
	$(SHELL) -i

clean: work
	rm -rf $(BINARY)

version:
	@echo ${VERSION}

.PHONY: install build cover work fmt test version clean check
