SHELL := /bin/sh

.DEFAULT_GOAL := up

DOCKER_COMPOSE ?= docker compose
COMPOSE := $(DOCKER_COMPOSE) --env-file .env
WAIT_TIMEOUT ?= 180

.PHONY: up env doctor config build down logs ps test vet check smoke help

up: build
	$(COMPOSE) up --detach --wait --wait-timeout $(WAIT_TIMEOUT)
	@server_port=$$(sed -n 's/^SERVER_PORT=//p' .env | tail -n 1); \
	server_port=$${server_port:-8080}; \
	printf '\nGo CMS Start is ready.\n  API: http://localhost:%s\n' "$$server_port"

env:
	@if [ -f .env ]; then \
		printf 'Using existing .env\n'; \
	else \
		cp .env.example .env; chmod 600 .env; \
		printf 'Created .env from .env.example\n'; \
	fi

doctor:
	@command -v docker >/dev/null 2>&1 || { printf 'Docker is required but was not found.\n' >&2; exit 1; }
	@docker info >/dev/null 2>&1 || { printf 'Docker daemon is unavailable.\n' >&2; exit 1; }
	@docker compose version >/dev/null 2>&1 || { printf 'Docker Compose v2 is required.\n' >&2; exit 1; }

config: env doctor
	$(COMPOSE) config --quiet

build: config
	$(COMPOSE) build server

down: config
	$(COMPOSE) down --remove-orphans

logs: config
	$(COMPOSE) logs --follow --tail=100

ps: config
	$(COMPOSE) ps --all

test:
	GOWORK=off go -C backend test ./...

vet:
	GOWORK=off go -C backend vet ./...

check: test vet
	GOWORK=off go -C backend build ./...
	$(DOCKER_COMPOSE) --env-file .env.example config --quiet

smoke:
	./scripts/smoke.sh

help:
	@printf 'Go CMS Start commands:\n'
	@printf '  make, make up  Build and start PostgreSQL and backend\n'
	@printf '  make env       Create .env if it is missing\n'
	@printf '  make build     Build the backend image\n'
	@printf '  make check     Run Go and Compose checks\n'
	@printf '  make smoke     Check a running API with dev credentials\n'
	@printf '  make down      Stop containers and preserve volumes\n'
	@printf '  make logs      Follow backend and database logs\n'
	@printf '  make ps        Show service status\n'
