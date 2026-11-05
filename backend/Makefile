.PHONY: test vet migrate seed run

test:
	go test ./...

vet:
	go vet ./...

migrate:
	go run ./cmd/migrate

seed:
	go run ./cmd/seed

run:
	go run ./cmd/server
