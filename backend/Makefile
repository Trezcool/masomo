MODULE := $(shell go list -m)
VCSREF := $(cd .. && git rev-parse --short HEAD)
BUILD_DATE := $(date -u +”%Y-%m-%dT%H:%M:%SZ”)

USERNAME?=""

# ==============================================================================
# Administration

createdb:
	go run ${MODULE}/apps/admin createdb

MIGCMD?=up
ARG1?=""
ARG2?=""
migrate:
	go run ${MODULE}/apps/admin migrate $(MIGCMD) $(ARG1) $(ARG2)

EMAIL?=""
ADMIN?=false
adduser:
ifeq ($(ADMIN),true)
	go run ${MODULE}/apps/admin adduser -username $(USERNAME) -email $(EMAIL) -admin
else
	go run ${MODULE}/apps/admin adduser -username $(USERNAME) -email $(EMAIL)
endif

resetpassword:
	go run ${MODULE}/apps/admin resetpassword -username $(USERNAME)

# ==============================================================================
# Tests & Linting

lint:
	go fmt ./...
	golangci-lint run --disable=typecheck

test: lint
	ENV=TEST go test -p 1 -timeout 2m -race -v ./...

# ==============================================================================
# Build

api:
	go build -ldflags "-X ${MODULE}/core.build=${VCSREF}" ${MODULE}/apps/api

admin:
	go build -ldflags "-X ${MODULE}/core.build=${VCSREF}" ${MODULE}/apps/admin

# ==============================================================================
# SQLBoiler models generation

boil:
	sqlboiler psql

# ==============================================================================
# Modules support
tidy:
	go mod tidy
	go mod vendor

deps-upgrade:
	go get -u -t -d -v ./...
	go mod tidy
	go mod vendor

deps-cleancache:
	go clean -modcache

# ==============================================================================
# Running from within docker compose

up: tidy
	MODULE=${MODULE} VCSREF=${VCSREF} BUILD_DATE=${BUILD_DATE} docker-compose -f docker-compose.yml -p masomo build
#	docker-compose -f docker-compose.yml -p masomo up --remove-orphans # fixme: SIGTERM not sent to container
	docker-compose -f docker-compose.yml -p masomo run --rm --service-ports web

down:
	docker-compose -f docker-compose.yml -p masomo down --remove-orphans
#	docker-compose -f docker-compose.yml -p masomo down --remove-orphans --volumes --rmi local  # cleanup

logs:
	docker-compose -f docker-compose.yml -p masomo logs -f

# ==============================================================================
# Deploy with Ansible

ENV?="qa"
DEPLOY_USER?=""
DEPLOY_SSH_KEY?=""
deploy:
	cd deploy/ansible
	. venv/bin/activate
	@ansible-playbook \
		-i inventories/$(ENV) -e host=$(ENV) \
		--vault-password-file .vault_pass
		-u $(DEPLOY_USER) -b --private-key $(DEPLOY_SSH_KEY) play.yml

.PHONY: api admin logs test deploy
