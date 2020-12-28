PACKAGENAME := $(shell go list -m)
GITVERSION := $(shell cd .. && git describe --always --tags)

USERNAME?=""

# ==============================================================================
# Administration

MIGCMD?=up
ARG1?=""
ARG2?=""
migrate:
	go run github.com/trezcool/masomo/apps/admin migrate $(MIGCMD) $(ARG1) $(ARG2)

EMAIL?=""
ADMIN?=false
adduser:
ifeq ($(ADMIN),true)
	go run github.com/trezcool/masomo/apps/admin adduser -username $(USERNAME) -email $(EMAIL) -admin
else
	go run github.com/trezcool/masomo/apps/admin adduser -username $(USERNAME) -email $(EMAIL)
endif

resetpassword:
	go run github.com/trezcool/masomo/apps/admin resetpassword -username $(USERNAME)

# ==============================================================================
# Tests & Linting

lint:
	go fmt ./...
	golangci-lint run --disable=typecheck

test: lint
	ENV=TEST go test -p 1 -timeout 2m -race -v ./...

# ==============================================================================
# Build

buildapi:
	go build -ldflags "-X ${PACKAGENAME}/core.build=${GITVERSION}" github.com/trezcool/masomo/apps/api

buildadmin:
	go build -ldflags "-X ${PACKAGENAME}/core.build=${GITVERSION}" github.com/trezcool/masomo/apps/admin

# ==============================================================================
# SQLBoiler models generation

boil:
	cd config && sqlboiler psql
