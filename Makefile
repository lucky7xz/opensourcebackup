.PHONY: deps test test-integration lint run \
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
lint:
	golangci-lint run ./...

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
