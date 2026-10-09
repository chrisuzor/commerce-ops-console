.PHONY: install test lint build run dev-api dev-web clean

install:
	cd web && npm ci

test:
	go test ./cmd/... ./internal/...
	cd web && npm test -- --run

lint:
	go vet ./cmd/... ./internal/...
	cd web && npm run lint

build:
	cd web && npm run build
	go build -o bin/commerce-ops ./cmd/api

run:
	cd web && npm run build
	go run ./cmd/api

dev-api:
	STATIC_DIR=- go run ./cmd/api

dev-web:
	cd web && npm run dev

clean:
	rm -rf bin web/dist
