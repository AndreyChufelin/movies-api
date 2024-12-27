.PHONY: run
COMPOSE_FILE=deployments/docker-compose.yaml
MIGRATIONS_DIR=migrations
DB_DRIVER=postgres
DB_STRING=postgres://postgres:postgres@db:5432/postgres?sslmode=disable

run:
	docker compose -f ${COMPOSE_FILE} up --build --remove-orphans

migrate:
	goose -dir $(MIGRATIONS_DIR) $(DB_DRIVER) $(DB_STRING) up

migrate-down:
	goose -dir $(MIGRATIONS_DIR) $(DB_DRIVER) $(DB_STRING) down
