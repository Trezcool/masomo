# ==============================================================================
# Administration

USERNAME?=""

resetpassword:
	go run github.com/trezcool/masomo/backend/apps/admin resetpassword -username $(USERNAME)

# ==============================================================================
# Tests & Linting

lint:
	go fmt ./...
	golangci-lint run --disable=typecheck

test: lint
	ENV=TEST go test -v -race ./...
