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

COMPOSE_POSTGRESTEST = deploy/compose/postgrestest.yaml

out/tracker-web: cmd/tracker-web/* internal/*/* internal/*/*/* internal/ui/spa/static/* internal/ui/spa/static/*/*
	CGO_ENABLED=0 go build -o $@ -ldflags "-X main.version=${VERSION_STRING}" ./cmd/tracker-web

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

.PHONY: compose-up-postgrestest
compose-up-postgrestest:
	docker compose -p test -f $(COMPOSE_POSTGRESTEST) up -d

.PHONY: compose-down-postgrestest
compose-down-postgrestest:
	docker compose -p test -f $(COMPOSE_POSTGRESTEST) down

.PHONY: compose-down-volumes-postgrestest
compose-down-volumes-postgrestest:
	docker compose -p test -f $(COMPOSE_POSTGRESTEST) down --volumes

.PHONY: clean
clean: clean-out clean-data

.PHONY: clean-out
clean-out:
	rm -rf out

.PHONY: clean-data
clean-data:
	rm -rf data
