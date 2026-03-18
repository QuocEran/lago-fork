.PHONY: dev build test lint generate migrate tidy

APP_NAME := lago-api
CMD_API   := ./cmd/api
CMD_WORKER := ./cmd/worker

# ── Development ──────────────────────────────────────────────────────────────

dev:
	go run $(CMD_API)/...

dev-worker:
	go run $(CMD_WORKER)/...

# ── Build ────────────────────────────────────────────────────────────────────

build:
	go build -o bin/$(APP_NAME) $(CMD_API)/...

build-worker:
	go build -o bin/$(APP_NAME)-worker $(CMD_WORKER)/...

# ── Testing ──────────────────────────────────────────────────────────────────

test:
	go test ./... -race -count=1

test-verbose:
	go test ./... -v -race -count=1

test-cover:
	go test ./... -race -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

# ── Code generation (gqlgen) ─────────────────────────────────────────────────

generate:
	go run github.com/99designs/gqlgen generate

# ── Linting ──────────────────────────────────────────────────────────────────

lint:
	golangci-lint run ./...

# ── Dependencies ─────────────────────────────────────────────────────────────

tidy:
	go mod tidy

# ── Database migrations ───────────────────────────────────────────────────────

migrate:
	migrate -path migrations -database "$$DATABASE_URL" up

migrate-down:
	migrate -path migrations -database "$$DATABASE_URL" down 1

migrate-status:
	migrate -path migrations -database "$$DATABASE_URL" version
