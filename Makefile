.PHONY: deps test test-integration lint lint-warn lint-install run \
        migrate-up migrate-down migrate-status \
        dev-up dev-down

# ── Config ─────────────────────────────────────────────────────────────────
DATABASE_URL     ?= postgres://opensourcebackup:dev_password@localhost:5432/opensourcebackup?sslmode=disable
MIGRATIONS_PATH  := migrations
CONTROL_PLANE    := ./cmd/control-plane
MIGRATE_BIN      := migrate

# ── Dependencies ────────────────────────────────────────────────────────────
deps:
	go mod download

# ── Tests ───────────────────────────────────────────────────────────────────
test:
	go test ./...

test-integration:
	DATABASE_URL=$(DATABASE_URL) go test -tags=integration ./...

# ── Lint ────────────────────────────────────────────────────────────────────
# Schicht 1: blockiert — Verletzung = kein Merge
lint:
	golangci-lint run ./...

# Schicht 2: Baustellen sichtbar machen — blockiert nie
# Linter die hier auftauchen, wandern nach einem Sprint in .golangci.yml Schicht 1
lint-warn:
	golangci-lint run ./... \
	  --enable revive,gocritic,cyclop,funlen,godot,exhaustive,wrapcheck,gomnd \
	  --exit-code 0

# golangci-lint installieren (einmalig)
lint-install:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# ── Run ─────────────────────────────────────────────────────────────────────
run:
	DATABASE_URL=$(DATABASE_URL) go run $(CONTROL_PLANE)

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
