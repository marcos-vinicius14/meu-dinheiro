ifneq (,$(wildcard .env.local))
    include .env.local
    export
else ifneq (,$(wildcard .env))
    include .env
    export
endif

.PHONY: test test-api test-bot run-api run-bot run-web build-all up down up-prd down-prd release

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

release:
	@if [ -z "$(TAG)" ]; then \
		echo "Uso: make release TAG=v0.1.0 [MSG=\"Release v0.1.0\"]"; \
		exit 1; \
	fi
	@if ! echo "$(TAG)" | grep -Eq '^v[0-9]+'; then \
		echo "Erro: A tag deve começar com 'v' (ex: TAG=v0.1.0) para acionar o workflow de deploy no Coolify."; \
		exit 1; \
	fi
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "Aviso: Há alterações não commitadas na árvore de trabalho. A tag $(TAG) apontará para o commit HEAD atual."; \
	fi
	@echo "Criando tag anotada $(TAG)..."
	git tag -a $(TAG) -m "$(if $(MSG),$(MSG),Release $(TAG))"
	@echo "Publicando tag $(TAG) no Git para disparar o deploy no Coolify..."
	git push origin $(TAG)
	@echo "Tag $(TAG) publicada com sucesso! O workflow de deploy foi iniciado."
