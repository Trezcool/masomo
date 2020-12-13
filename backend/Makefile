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
	ENV=TEST go test -v -race ./...

# ==============================================================================
# SQLBoiler models generation

boil:
	cd config && sqlboiler psql
