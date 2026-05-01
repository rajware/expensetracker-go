VERSION_MAJOR ?= 1
VERSION_MINOR ?= 0
BUILD_STRING  ?= 0
PATCH_STRING  ?= -alpha1
VERSION_STRING = $(VERSION_MAJOR).$(VERSION_MINOR).$(BUILD_STRING)$(PATCH_STRING)

REGISTRY_USER ?= quay.io/rajware
IMAGE_NAME = $(REGISTRY_USER)/expensetracker-go
IMAGE_TAG = $(IMAGE_NAME):$(VERSION_STRING)
IMAGE_TAG_LATEST = $(IMAGE_NAME):latest

IMAGE_PLATFORMS ?= linux/amd64,linux/arm64,linux/ppc64le,linux/s390x

.PHONY: test-domain
test-domain:
	go test -v ./internal/domain
