VERSION_MAJOR ?= 1
VERSION_MINOR ?= 0
BUILD_STRING  ?= 0
PATCH_STRING  ?=
VERSION_STRING = $(VERSION_MAJOR).$(VERSION_MINOR).$(BUILD_STRING)$(PATCH_STRING)

REGISTRY_USER ?= quay.io/rajware
IMAGE_NAME = $(REGISTRY_USER)/expensetracker-go
IMAGE_TAG = $(IMAGE_NAME):$(VERSION_STRING)
IMAGE_TAG_LATEST = $(IMAGE_NAME):latest
TAG_LATEST ?= -t $(IMAGE_TAG_LATEST)

IMAGE_PLATFORMS ?= linux/amd64,linux/arm64,linux/ppc64le,linux/s390x

SRC_FILES = cmd/tracker-web/* internal/*/* internal/*/*/* internal/ui/spa/static/* internal/ui/spa/static/*/*

COMPOSE_POSTGRESTEST = deploy/compose/postgrestest.yaml

RELEASE_TARGETS = linux_amd64 linux_arm64 darwin_amd64 darwin_arm64 windows_amd64 windows_arm64

# Macro for dynamically creating target-specific rules
define RELEASERULE
out/tracker-web_$O_$A$(if $(filter windows,$O),.exe): $$(SRC_FILES)
	CGO_ENABLED=0 GOOS=$O GOARCH=$A go build -o $$@ -ldflags "-X main.version=${VERSION_STRING}" ./cmd/tracker-web

.PHONY: $O_$A
$O_$A: out/tracker-web_$O_$A$(if $(filter windows,$O),.exe)
endef

# Targets
# The default target is an executable on the current OS and ARCH.
# This is used in the Dockerfile
out/tracker-web: $(SRC_FILES)
	CGO_ENABLED=0 go build -o $@ -ldflags "-X main.version=${VERSION_STRING}" ./cmd/tracker-web

.PHONY: default
default: out/tracker-web

# Apply the RELEASERULE macro to each release target. This will
# create rules like the following:
#
# out/tracker-web_linux_amd64: $(SRC_FILES)
#	CGO_ENABLED=0 GOOS=$O GOARCH=$A go build -o $@ -ldflags "-X main.version=${VERSION_STRING}" ./cmd/tracker-web
#
# .PHONY: linux_amd64
# linux_amd64: out/tracker-web_linux_amd64
$(foreach target,$(RELEASE_TARGETS), \
  $(eval O=$(word 1,$(subst _, ,$(target)))) \
  $(eval A=$(word 2,$(subst _, ,$(target)))) \
  $(eval $(RELEASERULE)) \
)

out/tracker-sqlite.k8s.yaml: deploy/kubernetes/tracker-sqlite/*.yaml deploy/kubernetes/tracker-sqlite/doc.txt
	./scripts/build-k8smanifest.sh "tracker-sqlite" "$(VERSION_STRING)" $@

out/tracker-postgres.k8s.yaml: deploy/kubernetes/tracker-postgres/*.yaml deploy/kubernetes/tracker-postgres/doc.txt
	./scripts/build-k8smanifest.sh "tracker-postgres" "$(VERSION_STRING)" $@

out/tracker-sqlite.compose.yaml: deploy/compose/tracker-sqlite.yaml
	./scripts/build-composemanifest.sh "tracker-sqlite" "$(VERSION_STRING)" $@

out/tracker-postgres.compose.yaml: deploy/compose/tracker-postgres.yaml
	./scripts/build-composemanifest.sh "tracker-postgres" "$(VERSION_STRING)" $@

.PHONY: release
release: $(RELEASE_TARGETS) out/tracker-sqlite.k8s.yaml out/tracker-postgres.k8s.yaml out/tracker-sqlite.compose.yaml out/tracker-postgres.compose.yaml

# Tests
.PHONY: test
test: test-domain test-auth-cookie test-rest-api test-repo-sqlite test-repo-postgres

.PHONY: test-domain
test-domain:
	go test -v ./internal/domain

.PHONY: test-repo-sqlite
test-repo-sqlite:
	go test -v ./internal/repository/sqlite

.PHONY: test-auth-cookie
test-auth-cookie:
	go test -v ./internal/auth/cookie

.PHONY: test-rest-api
test-rest-api:
	go test -v ./internal/api/rest

.PHONY: test-repo-postgres
test-repo-postgres: compose-up-postgrestest
	go test -v ./internal/repository/postgres

# Docker Compose
.PHONY: compose-up-postgrestest
compose-up-postgrestest:
	docker compose -p test -f $(COMPOSE_POSTGRESTEST) up -d

.PHONY: compose-down-postgrestest
compose-down-postgrestest:
	docker compose -p test -f $(COMPOSE_POSTGRESTEST) down

.PHONY: compose-down-volumes-postgrestest
compose-down-volumes-postgrestest:
	docker compose -p test -f $(COMPOSE_POSTGRESTEST) down --volumes

# Docker
.PHONY: local-image
local-image:
	docker buildx build --load \
	                    -f package/docker/Dockerfile \
						--build-arg VERSION_STRING=${VERSION_STRING} \
						-t $(IMAGE_TAG_LATEST) \
						.

.PHONY: final-image
final-image:
	docker buildx build --push \
						--platform $(IMAGE_PLATFORMS) \
						-f package/docker/Dockerfile \
						--build-arg VERSION_STRING=${VERSION_STRING} \
						-t $(IMAGE_TAG) \
						$(TAG_LATEST) \
						.
# Clean
.PHONY: clean
clean: clean-out clean-data

.PHONY: clean-out
clean-out:
	rm -rf out

.PHONY: clean-data
clean-data:
	rm -rf data

.PHONY: clean-local-image
clean-local-image:
	docker image rm $(IMAGE_TAG_LATEST)
