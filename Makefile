# ==== минимальные настройки ====
APP      := order-svc
CMD      := cmd/app
BIN      := bin/$(APP)
DC       := docker compose

# контейнер и доступ к БД из docker-compose
PG_CONT  := l0-postgres-1
PG_USER  := postgres
PG_PASS  := postgres
PG_DB    := order_service_db

.PHONY: up
up:
	$(DC) up -d --build

.PHONY: down
down:
	$(DC) down -v

.PHONY: migrate-up
migrate-up:
	@echo ">> applying db/001_init.sql"
	docker cp db/001_init.sql $(PG_CONT):/tmp/001_init.sql
	docker exec -e PGPASSWORD=$(PG_PASS) $(PG_CONT) \
		psql -U $(PG_USER) -d $(PG_DB) -f /tmp/001_init.sql

.PHONY: migrate-down
migrate-down:
	@echo ">> applying db/001_down.sql"
	docker cp db/001_down.sql $(PG_CONT):/tmp/001_down.sql
	docker exec -e PGPASSWORD=$(PG_PASS) $(PG_CONT) \
		psql -U $(PG_USER) -d $(PG_DB) -f /tmp/001_down.sql

.PHONY: build
build:
	mkdir -p bin
	go build -o $(BIN) ./$(CMD)

.PHONY: run
run: build
	$(BIN)

.PHONY: logs
logs:
	$(DC) logs -f

.PHONY: clean
clean:
	rm -rf bin

# Сгенерируем и отправим случайные заказы в брокер сообщений (Kafka).
.PHONY: produce
produce:
	go run ./cmd/producer -n 5 -interval 1s

# Соберём сервис с включенным кэшем
.PHONY: cache-on-up
cache-on-up:
	@CACHE_ENABLED=1 docker compose up -d --build

# Соберём сервис с выключенным кэшем
.PHONY: cache-off-up
cache-off-up:
	@CACHE_ENABLED=0 docker compose up -d --build

.PHONY: app-logs
app-logs:
	docker compose logs -f app

.PHONY: build-app
build-app:
	docker compose build app
