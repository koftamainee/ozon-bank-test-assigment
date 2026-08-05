.PHONY: dev infra stop logs build up down test migrate migrate-create \
       install-air lint lint-fix db-shell db-reset tidy check sqlc install-sqlc gqlgen install-gqlgen

dev: infra
	if [ -n "$$TMUX" ]; then \
		tmux new-window -n app "export \$$(grep -v '^#' .env 2>/dev/null | xargs); air"; \
		tmux select-window -t shell; \
	else \
		tmux new-window -t forum -n app "export \$$(grep -v '^#' .env 2>/dev/null | xargs); air"; \
		tmux select-window -t forum:shell; \
		tmux attach -t forum; \
	fi

infra:
	if [ -n "$$TMUX" ]; then \
		tmux new-window -n shell; \
		tmux send-keys -t shell "cd $(shell pwd)" C-m; \
		tmux new-window -n infra "docker compose -f docker-compose.dev.yaml up"; \
	else \
		tmux new-session -d -s forum -n shell; \
		tmux send-keys -t forum:shell "cd $(shell pwd)" C-m; \
		tmux new-window -t forum -n infra "docker compose -f docker-compose.dev.yaml up"; \
	fi

stop:
	docker compose -f docker-compose.dev.yaml down 2>/dev/null

logs:
	docker compose -f docker-compose.dev.yaml logs -f

build:
	docker compose build

up:
	docker compose up -d

down:
	docker compose down

db-shell:
	docker compose -f docker-compose.dev.yaml exec db psql -U app_admin -d forum

db-reset:
	docker compose -f docker-compose.dev.yaml down -v
	docker compose -f docker-compose.dev.yaml up -d db
	@echo "waiting for db..."
	@until docker compose -f docker-compose.dev.yaml exec db pg_isready -U app_admin -d forum >/dev/null 2>&1; do sleep 1; done
	$(MAKE) migrate

migrate:
	@export $$(grep -v '^#' .env 2>/dev/null | xargs); \
	if [ -z "$${MIGRATOR_PASSWORD:-}" ]; then echo "error: MIGRATOR_PASSWORD not set in .env"; exit 1; fi; \
	docker compose -f docker-compose.dev.yaml run --rm migrator -path /migrations -database "postgres://migrator:$${MIGRATOR_PASSWORD}@db:5432/forum?sslmode=disable" up

migrate-create:
	@if [ -z "$(NAME)" ]; then echo "usage: make migrate-create NAME=add_users"; exit 1; fi
	@sh -c ' \
		LAST=$$(ls migrations/*.up.sql 2>/dev/null | sed "s|.*/||;s|_.*||" | sort -n | tail -1); \
		NEXT=$${LAST:-0}; \
		NEXT=$$(( NEXT + 1 )); \
		NUM=$$(printf "%06d" $$NEXT); \
		touch "migrations/$${NUM}_$(NAME).up.sql"; \
		touch "migrations/$${NUM}_$(NAME).down.sql"; \
		echo "created migrations/$${NUM}_$(NAME).{up,down}.sql"'

test:
	go test -race ./...
	@echo "all tests passed"

check:
	go vet ./...
	go test -race ./...
	golangci-lint run ./...
	@echo "check passed"

tidy:
	go mod tidy

lint:
	golangci-lint run ./...
	@echo "lint passed"

lint-fix:
	golangci-lint run --fix ./...
	@echo "lint passed"

install-air:
	go install github.com/air-verse/air@latest

install-sqlc:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1

sqlc:
	@printf '%s\n' "-- Code generated from migrations/*.up.sql by 'make sqlc'. DO NOT EDIT." > sqlc/schema.gen.sql
	@cat migrations/*.up.sql >> sqlc/schema.gen.sql
	go run github.com/sqlc-dev/sqlc/cmd/sqlc generate -f sqlc/sqlc.yaml

install-gqlgen:
	go install github.com/99designs/gqlgen@v0.17.94

gqlgen:
	cd internal/graphql && go run github.com/99designs/gqlgen generate
