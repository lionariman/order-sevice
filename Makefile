# ==== минимальные настройки ====
APP      := order-svc
CMD      := cmd/app
BIN      := bin/$(APP)
DC       := docker compose

# контейнер и доступ к БД из docker-compose
PG_CONT  := l0-postgres-1
PG_USER  := postgres
PG_PASS  := postgres
PG_DB    := orders

# ==== цели ====
.PHONY: up down migrate build run logs clean produce cache-on-up cache-off-up

up:
	$(DC) up -d --build

down:
	$(DC) down -v

migrate:
	@echo ">> applying db/001_init.sql"
	docker cp db/001_init.sql $(PG_CONT):/tmp/001_init.sql
	docker exec -e PGPASSWORD=$(PG_PASS) $(PG_CONT) \
		psql -U $(PG_USER) -d $(PG_DB) -f /tmp/001_init.sql

build:
	mkdir -p bin
	go build -o $(BIN) ./$(CMD)

run: build
	$(BIN)

logs:
	$(DC) logs -f

clean:
	rm -rf bin

# Сгенерируем и отправим случайные заказы в брокер сообщений (Kafka).
produce:
	go run ./cmd/producer -n 5 -interval 1s

# Соберём сервис с включенным кэшем
cache-on-up:
	@CACHE_ENABLED=1 docker compose up -d --build

# Соберём сервис с выключенным кэшем
cache-off-up:
	@CACHE_ENABLED=0 docker compose up -d --build

app-logs:
	docker compose logs -f app
