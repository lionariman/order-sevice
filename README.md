# Order Service

Простой сервис «Kafka → PostgreSQL → in-memory cache → HTTP API». Consumer считывает заказы из Kafka, валидирует их, апсертит агрегированную структуру в базу и при необходимости прогревает кэш; HTTP‑слой отдаёт заказ по `GET /order/{id}` и сообщает источники данных в заголовках `X-Source` / `X-Duration-ms`.

## Архитектура и технологии
- **Producer** (`cmd/producer`) генерирует данные через gofakeit и публикует их в Kafka.
- **Consumer** (`internal/consumer.go`) использует `OrderRepository`/`OrderCache` интерфейсы, валидацию на `go-playground/validator` и клиент Sarama.
- **Repository** (`internal/repo.go`) реализует работу с PostgreSQL (pgx/v5) и прогрев кэша.
- **Cache** (`internal/cache.go`) хранит до `MaxCacheLimit` заказов, умеет Warm/Delete/DeleteAllItems.
- **HTTP** (`internal/http.go`) построен на httprouter.
- Docker Compose поднимает PostgreSQL, Kafka+Zookeeper, Kafka UI и само приложение.

## Структура проекта
```
cmd/app          – запуск основного сервиса
cmd/producer     – генератор тестовых заказов
internal         – модели, интерфейсы, бизнес-логика, HTTP, кэш
internal/mocks   – mockgen для OrderCache/OrderRepository
db               – миграции (001_init.sql, 001_down.sql)
tests            – unit-тесты HTTP, consumer’а и кэша
web              – простая HTML-страница
Dockerfile, docker-compose.yaml, Makefile
```

## Конфигурация
Все параметры читаются из окружения (см. `internal/config.go`). Пример значений — в `.env.example`.
- `HTTP_ADDR` — адрес HTTP-сервера.
- `PG_URL` — строка подключения к PostgreSQL.
- `KAFKA_BROKERS`, `KAFKA_TOPIC`, `KAFKA_GROUP_ID` — настройки Kafka.
- `CACHE_ENABLED` — включает/выключает использование кэша.

## Запуск
1. `cp .env.example .env` (при необходимости обновите значения).
2. `make up` — поднять PostgreSQL, Kafka+Zookeeper, Kafka UI и приложение; `make up-nocache` запускает сервис без кэша.
3. `make migrate-up` / `make migrate-down` — применить или откатить миграции (`db/001_init.sql`, `db/001_down.sql`).
4. Сгенерировать заказы: `make produce` или `go run ./cmd/producer -n 10 -interval 500ms`.
5. Смотреть сообщения Kafka: откройте Kafka UI на `http://localhost:8080`, выберите топик `orders` и пролистайте события.
6. Web/API:
   - `curl http://localhost:8081/order/<order_uid>` — получить заказ (добавьте `?nocache=1`, чтобы обойти кэш).
   - Перейдите на `http://localhost:8081/`, введите `order_uid` в форме и получите JSON прямо в браузере.

## Тесты
- Моки находятся в `internal/mocks` и генерируются `mockgen` (см. директиву `//go:generate` в `internal/contracts.go`).
- Тесты запускаются командой `go test ./...` и покрывают HTTP‑хендлер, Kafka consumer и in-memory кэш.
