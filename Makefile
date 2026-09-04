.PHONY: db-up db-down migrate-up migrate-down build run test lint

db-up:
	docker compose up -d

db-down:
	docker compose down

migrate-up:
	docker exec -i outpost-db psql -U postgres -d outpost < migrations/001_create_applications_table.up.sql
	docker exec -i outpost-db psql -U postgres -d outpost < migrations/002_create_endpoints_table.up.sql

migrate-down:
	docker exec -i outpost-db psql -U postgres -d outpost < migrations/002_create_endpoints_table.down.sql
	docker exec -i outpost-db psql -U postgres -d outpost < migrations/001_create_applications_table.down.sql

build:
	go build -o bin/api cmd/api/main.go

run:
	go run cmd/api/main.go

test:
	go test -v ./...

lint:
	golangci-lint run ./...
