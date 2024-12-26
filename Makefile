.PHONY: run
COMPOSE_FILE=deployments/docker-compose.yaml

run:
	docker compose -f ${COMPOSE_FILE} up --build --remove-orphans
