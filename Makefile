run:
	go run ./cmd/api

build:
	go build ./...

test:
	go test ./...

tidy:
	go mod tidy
