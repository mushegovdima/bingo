ENV ?= dev

up:
	export $$(grep -v '^#' frontend/.env | xargs) && ENV=$(ENV) docker compose up -d --build

dev: up
	ngrok http 3000

down:
	docker compose down

down-volumes:
	docker compose down -v

logs:
	docker compose logs -f

migrate-status:
	docker compose run --rm migrator ./migrator -env $(ENV) status
	docker compose run --rm notifier-migrator ./migrator -env $(ENV) status

migrate-rollback:
	docker compose run --rm migrator ./migrator -env $(ENV) rollback

migrate-rollback-notifier:
	docker compose run --rm notifier-migrator ./migrator -env $(ENV) rollback

COUNT ?= 30
seed:
	docker compose build api
	docker compose run --rm --entrypoint ./seed api -env $(ENV) -count $(COUNT)

proto:
	cd api && PATH="$$(go env GOPATH)/bin:$$PATH" buf generate ../proto
	cd notifier && PATH="$$(go env GOPATH)/bin:$$PATH" buf generate ../proto

test:
	cd api && go test ./... -race
	cd notifier && go test ./... -race

.PHONY: up dev down down-volumes logs migrate-status migrate-rollback migrate-rollback-notifier seed proto test
