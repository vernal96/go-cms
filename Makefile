SHELL := /bin/sh

.DEFAULT_GOAL := up
.NOTPARALLEL:

DOCKER_COMPOSE ?= docker compose
COMPOSE := $(DOCKER_COMPOSE) --env-file .env
WAIT_TIMEOUT ?= 180
INFRA_SERVICES := postgres kafka rabbitmq redis loki grafana

.PHONY: up restart env doctor config build down logs ps help

up: build
	@printf '\nRemoving application containers before database initialization...\n'
	$(COMPOSE) rm --stop --force admin server
	@printf '\nInstalling admin dependencies...\n'
	$(COMPOSE) run --rm --no-deps admin npm ci
	@printf '\nStarting infrastructure services...\n'
	$(COMPOSE) up --detach --wait --wait-timeout $(WAIT_TIMEOUT) $(INFRA_SERVICES)
	@printf '\nApplying database migrations...\n'
	$(COMPOSE) run --rm --no-deps server /usr/local/bin/console migrations up
	@printf '\nApplying development seeds...\n'
	$(COMPOSE) run --rm --no-deps server /usr/local/bin/console seeds up -tags=dev
	@printf '\nStarting application services...\n'
	$(COMPOSE) up --detach --wait --wait-timeout $(WAIT_TIMEOUT)
	@server_port=$$(awk -F= '$$1 == "SERVER_PORT" { print $$2 }' .env | tail -n 1); \
	admin_port=$$(awk -F= '$$1 == "ADMIN_PORT" { print $$2 }' .env | tail -n 1); \
	server_port=$${server_port:-8080}; \
	admin_port=$${admin_port:-5173}; \
	printf '\nGo CMS is ready.\n'; \
	printf '  Admin:   http://localhost:%s\n' "$$admin_port"; \
	printf '  API:     http://localhost:%s\n' "$$server_port"; \
	printf '  Grafana: http://localhost:3000\n'; \
	printf '  Login:   admin\n'; \
		printf '  Password: admin-dev-only-2026\n\n'

restart: config
	$(COMPOSE) up --detach --build --force-recreate --wait --wait-timeout $(WAIT_TIMEOUT) server admin
	@printf '\nBackend and admin frontend restarted.\n'

env:
	@if [ -f .env ]; then \
		printf 'Using existing .env\n'; \
	else \
		cp .env.example .env; \
		chmod 600 .env; \
		printf 'Created .env from .env.example\n'; \
	fi

doctor:
	@command -v docker >/dev/null 2>&1 || { printf 'Docker is required but was not found.\n' >&2; exit 1; }
	@docker info >/dev/null 2>&1 || { printf 'Docker daemon is not available.\n' >&2; exit 1; }
	@docker compose version >/dev/null 2>&1 || { printf 'Docker Compose is required but was not found.\n' >&2; exit 1; }

config: env doctor
	$(COMPOSE) config --quiet

build: config
	$(COMPOSE) build server admin

down: config
	$(COMPOSE) down --remove-orphans

logs: config
	$(COMPOSE) logs --follow --tail=100

ps: config
	$(COMPOSE) ps --all

help:
	@printf 'Go CMS development commands:\n'
	@printf '  make, make up  Build, initialize, and start the complete project\n'
	@printf '  make restart   Rebuild and restart backend and admin frontend\n'
	@printf '  make env       Create .env from .env.example when it is missing\n'
	@printf '  make build     Build the server and admin images\n'
	@printf '  make down      Stop containers without deleting persistent volumes\n'
	@printf '  make logs      Follow logs from all services\n'
	@printf '  make ps        Show service status\n'
	@printf '  make help      Show this help\n'
