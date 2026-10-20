.PHONY: check build

check:
	@test -z "$$(gofmt -l cmd internal)" || (gofmt -l cmd internal; exit 1)
	go vet ./...
	go test ./...

build:
	go build -o bin/worker ./cmd/worker
	go build -o bin/clockctl ./cmd/clockctl
