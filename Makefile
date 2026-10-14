SHELL = /bin/bash

MKFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
ROOT := $(dir $(MKFILE_PATH))
GOBIN ?= $(ROOT)/tools/bin
ENV_PATH = PATH=$(GOBIN):$(PATH)
BIN_PATH ?= $(ROOT)/bin

GOPRIVATE = GOPRIVATE=github.com/Mikhalevich/

LINTER_NAME := golangci-lint
LINTER_VERSION := v2.14.0

.PHONY: all build test bench compose-up compose-down load-test-data vendor install-linter \
lint fmt tools-update generate load-assets \
minikube-load-images minikube-decrypt-secrets minikube-encrypt-secrets minikube-helm-install minikube-helm-uninstall \
install-helm-secrets generate-helm-secrets \
do-helm-load-images do-helm-encrypt-secrets do-helm-install do-helm-uninstall

all: build

build:
	$(GOPRIVATE) go build -mod=vendor -o $(BIN_PATH)/web ./cmd/web/main.go

test:
	go test ./...

bench:
	go test -bench=. -benchmem ./...

compose-up:
	docker compose -f ./script/docker/docker-compose.yml up --build

compose-down:
	docker compose -f ./script/docker/docker-compose.yml down

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

generate:
	$(ENV_PATH) go generate ./...
