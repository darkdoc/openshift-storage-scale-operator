# SHELL defines bash so all the inline scripts here will work as expected.
SHELL := /usr/bin/env bash

DEFAULT_VERSION := 0.0.1
VERSION ?= $(DEFAULT_VERSION)
export VERSION

# For the default version, use 'latest' image tags.
# Otherwise version prefixed with 'v'
ifeq ($(VERSION), $(DEFAULT_VERSION))
IMAGE_TAG = latest
else
IMAGE_TAG = v$(VERSION)
endif
export IMAGE_TAG

# Override this when building images for dev only!
IMAGE_REGISTRY ?= quay.io/openshift-storage-scale
# Image pullspec to use for all building/pushing image targets
IMG ?= $(IMAGE_REGISTRY)/openshift-fusion-access-operator-console:$(IMAGE_TAG)

POCKER := $(shell command -v docker 2>/dev/null || command -v podman 2>/dev/null)
ifeq ($(POCKER),)
$(error No container runtime found. Please install Docker or Podman)
endif

# Some useful colors for environments that support ANSI colors
ifeq ($(NO_COLOR),)
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
MAGENTA := \033[0;35m
CYAN := \033[0;36m
BOLD := \033[1m
RESET := \033[0m
else
RED :=
GREEN :=
YELLOW :=
MAGENTA :=
CYAN :=
BOLD :=
RESET :=
endif

.DEFAULT_GOAL := install

##@ Targets

.PHONY: help
help: ## Display this help
	@awk \
		'BEGIN \
			{ \
				FS = ":.*##"; \
				printf "\nUsage:\n  make ${CYAN}<target>${RESET}\n" \
			} /^[a-zA-Z_0-9-]+:.*?##/ \
			{ printf "  ${CYAN}%-15s${RESET} %s\n", $$1, $$2 } /^##@/ \
			{ printf "\n${BOLD}%s${RESET}\n", substr($$0, 5) } \
		' $(MAKEFILE_LIST)

##@ Development

.PHONY: install
install: ## Installs dependencies (default)
	npm install

.PHONY: lint
lint: install ## Lints source files
	npm run lint

.PHONY: clean
clean: ## Removes the node_modules directory
	rm -rf node_modules

.PHONY: container-build
container-build: ## Builds the container image
	$(POCKER) build -f docker/Dockerfile -t ${IMG} ..
docker-build: container-build

.PHONY: container-push
container-push: ## Pushes the container image
	$(POCKER) push ${IMG}
docker-push: container-push
