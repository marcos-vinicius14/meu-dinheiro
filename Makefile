ifneq (,$(wildcard .env.local))
    include .env.local
    export
else ifneq (,$(wildcard .env))
    include .env
    export
endif

.PHONY: test test-api test-bot run-api run-bot run-web build-all up down up-prd down-prd

test: test-api test-bot

test-api:
	cd apps/api && go test -v ./...

test-bot:
	cd apps/bot && go test -v ./...

run-api:
	cd apps/api && go run ./cmd/api

run-bot:
	cd apps/bot && go run ./cmd/bot

run-web:
	cd apps/web && npm run dev

build-all:
	cd apps/api && go build -o ../../bin/api ./cmd/api
	cd apps/bot && go build -o ../../bin/bot ./cmd/bot
	cd apps/web && npm run build

up:
	docker compose up -d postgres

down:
	docker compose down

up-prd:
	docker compose -f compose.prd.yaml up -d

down-prd:
	docker compose -f compose.prd.yaml down
