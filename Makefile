.PHONY: db-up db-down migrate-up migrate-down run

db-up:
	docker compose up -d

db-down:
	docker compose down

migrate-up:
	docker exec -i outpost-db psql -U postgres -d outpost < migrations/001_create_applications_table.up.sql

migrate-down:
	docker exec -i outpost-db psql -U postgres -d outpost < migrations/001_create_applications_table.down.sql

run:
	go run cmd/api/main.go
