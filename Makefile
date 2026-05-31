.PHONY: deps test test-integration fmt lint lint-warn check lint-install run \
        migrate-up migrate-down migrate-status \
        dev-up dev-down

# ── Config ─────────────────────────────────────────────────────────────────
DATABASE_URL     ?= postgres://opensourcebackup:dev_password@localhost:5432/opensourcebackup?sslmode=disable
MIGRATIONS_PATH  := migrations
CONTROL_PLANE    := ./cmd/control-plane
AGENT            := ./cmd/agent
MIGRATE_BIN      := migrate

# ── Dependencies ────────────────────────────────────────────────────────────
deps:
	go mod download

# ── Tests ───────────────────────────────────────────────────────────────────
test:
	go test ./...

test-integration:
	DATABASE_URL=$(DATABASE_URL) go test -tags=integration ./...

# ── Format ──────────────────────────────────────────────────────────────────
fmt:
	gofmt -w .
	goimports -w -local github.com/cerberus8484/opensourcebackup .

# ── Lint ────────────────────────────────────────────────────────────────────
# Nur eigene Packages — web/node_modules enthält npm-Go-Dateien die wir nicht kontrollieren
GO_PKGS := ./cmd/... ./internal/...

# Schicht 1: blockiert — Verletzung = kein Merge (siehe .golangci.hard.yml)
lint:
	golangci-lint run --config .golangci.hard.yml $(GO_PKGS)

# Schicht 2: Baustellen — blockiert nie (siehe .golangci.warn.yml)
lint-warn:
	golangci-lint run --config .golangci.warn.yml $(GO_PKGS)

# Alles in einem: fmt → lint → test
check: fmt lint test

# golangci-lint v2 installieren (einmalig)
lint-install:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

# ── Run ─────────────────────────────────────────────────────────────────────
run:
	DATABASE_URL=$(DATABASE_URL) go run $(CONTROL_PLANE)

run-agent:
	go run $(AGENT)

# ── Agent Release Builds ─────────────────────────────────────────────────────
VERSION ?= v0.1.0

build-agent-windows:
	GOOS=windows GOARCH=amd64 go build -o dist/agent/$(VERSION)/opensourcebackup-agent-windows-amd64.exe $(AGENT)

build-agent-linux:
	GOOS=linux GOARCH=amd64 go build -o dist/agent/$(VERSION)/opensourcebackup-agent-linux-amd64 $(AGENT)

build-agent-linux-arm64:
	GOOS=linux GOARCH=arm64 go build -o dist/agent/$(VERSION)/opensourcebackup-agent-linux-arm64 $(AGENT)

build-agent-darwin:
	GOOS=darwin GOARCH=arm64 go build -o dist/agent/$(VERSION)/opensourcebackup-agent-darwin-arm64 $(AGENT)

build-agent-all: build-agent-windows build-agent-linux build-agent-linux-arm64 build-agent-darwin

# ── Migrations ──────────────────────────────────────────────────────────────
# Requires: go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
migrate-up:
	$(MIGRATE_BIN) -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" up

migrate-down:
	$(MIGRATE_BIN) -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" down

migrate-status:
	$(MIGRATE_BIN) -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" version

# ── Docker Dev Stack ─────────────────────────────────────────────────────────
dev-up:
	docker compose -f deployments/docker-compose/dev.yml up -d

dev-down:
	docker compose -f deployments/docker-compose/dev.yml down
