-include .env

run:
	go run ./cmd/api

build:
	go build ./...

test:
	go test ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

check: fmt vet test

DB_URL=postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)

migrate-up:
	@goose -dir migrations postgres "$(DB_URL)" up

migrate-down:
	@goose -dir migrations postgres "$(DB_URL)" down

migrate-status:
	@goose -dir migrations postgres "$(DB_URL)" status
