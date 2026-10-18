SHELL = /bin/bash

MKFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
ROOT := $(dir $(MKFILE_PATH))
GOBIN ?= $(ROOT)/tools/bin
ENV_PATH = PATH=$(GOBIN):$(PATH)
BIN_PATH ?= $(ROOT)/bin

GOPRIVATE = GOPRIVATE=github.com/Mikhalevich/

LINTER_NAME := golangci-lint
LINTER_VERSION := v2.14.0

.PHONY: all build run-web test bench vendor \
install-linter lint fmt tools-update \
web-install web-gen web-dev web-build web-lint \
compose-up compose-down

all: build

build:
	$(GOPRIVATE) go build -mod=vendor -o $(BIN_PATH)/web ./cmd/web

run-web:
	./bin/web -config=./config/config-web.yaml

compose-up:
	docker compose -f script/docker/docker-compose.yml up --build

compose-down:
	docker compose -f script/docker/docker-compose.yml down

test:
	go test ./...

bench:
	go test -bench=. -benchmem ./...

vendor:
	$(GOPRIVATE) go mod tidy
	$(GOPRIVATE) go mod vendor

install-linter:
	if [ ! -f $(GOBIN)/$(LINTER_VERSION)/$(LINTER_NAME) ]; then \
		echo INSTALLING $(GOBIN)/$(LINTER_VERSION)/$(LINTER_NAME) $(LINTER_VERSION) ; \
		curl -sSfL https://golangci-lint.run/install.sh  | sh -s -- -b $(GOBIN)/$(LINTER_VERSION) $(LINTER_VERSION) ; \
		echo DONE ; \
	fi

lint: install-linter
	$(GOBIN)/$(LINTER_VERSION)/$(LINTER_NAME) run --config .golangci.yml

fmt: install-linter
	$(GOBIN)/$(LINTER_VERSION)/$(LINTER_NAME) fmt --config .golangci.yml

tools-update:
	go get tool

web-install:
	cd web && npm ci

web-gen:
	cd web && npm run gen

web-dev:
	cd web && npm run dev

web-build:
	cd web && npm run build

web-lint:
	cd web && npm run lint
