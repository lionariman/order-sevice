# Order Service

Мини‑сервис, получающий заказы из Kafka, сохраняющий их в PostgreSQL, прогревающий/обновляющий кэш в памяти и отдающий данные по HTTP. Используется как учебный пример построения конвейера Kafka → PostgreSQL → Cache → HTTP.

## Архитектура
- **Producer** (`cmd/producer`) публикует случайные заказы в Kafka (topic `orders`).
- **Consumer** (`internal/consumer.go`) читает сообщения из Kafka, валидирует JSON (минимально) и апсертит заказ в PostgreSQL через `Repo`.
- **Repository** (`internal/repo.go`) хранит агрегированную структуру `Order` в нескольких таблицах (`orders`, `deliveries`, `payments`, `items`). При чтении собирает её обратно.
- **Cache** (`internal/cache.go`) хранит последние заказы в памяти, прогревается при старте из БД (`LoadRecent`) и обновляется из consumer'а.
- **HTTP API** (`internal/http.go`) отдаёт заказ по `GET /order/{id}`. Заголовки `X-Source` и `X-Duration-ms` показывают источник данных (кэш или БД) и время обработки. Статический HTML (`web/index.html`) доступен по `/`.
- **Инфраструктура** описана в `docker-compose.yaml`: Kafka + Zookeeper, PostgreSQL с автоматическим применением миграции `db/001_init.sql`, Kafka UI и само приложение.

## Структура проекта
- `cmd/app/main.go` — точка входа основного сервиса.
- `cmd/producer/main.go` — CLI-утилита для генерации и отправки заказов в Kafka.
- `internal/` — доменные сущности, бизнес-логика, доступ к БД, HTTP-слой и кэш.
- `db/` — SQL-модели и миграции (сейчас только `001_init.sql` для инициализации схемы).
- `web/` — статические файлы и простая HTML-страница для тестирования.
- `Dockerfile`, `docker-compose.yaml` — сборка и запуск инфраструктуры.
- `Makefile` — удобные команды для сборки, запуска и вспомогательных действий.

## Используемые технологии
- Go 1.24.4
- Kafka (Confluent 7.6.x) + Zookeeper
- PostgreSQL 16 + pgx/v5
- Sarama (Kafka client), httprouter
- Docker + Docker Compose для локального окружения

## Запуск локально
### Предварительные требования
- Docker и Docker Compose
- Make (опционально, но упрощает запуск)
- Go (если планируется запуск без контейнеров)

### Основные команды
- `make up` — поднять весь стек (Kafka, PostgreSQL, приложение) в Docker.
- `make cache-off-up` — запустить приложение с выключенным кэшем (`CACHE_ENABLED=0`).
- `make cache-on-up` — аналогично `make up`, но явно включает кэш.
- `make logs` или `docker compose logs -f` — посмотреть логи сервисов.
- `make producer` — опубликовать 5 тестовых заказов в Kafka (можно настроить флагами `-n` и `-interval`).
- `make down` — остановить и удалить контейнеры и volume'ы.

После запуска `GET http://localhost:8081/order/<order_uid>` вернёт JSON заказа (при наличии в БД/кэше). Параметр `?nocache=1` принудительно ходит в БД.

### Ручной запуск без Docker
1. Запустите Kafka/PostgreSQL любым удобным способом, примените `db/001_init.sql`.
2. Настройте переменные окружения (см. раздел «Конфигурация»).
3. Соберите приложение: `go build -o bin/order-svc ./cmd/app`.
4. Запустите бинарь `bin/order-svc`.

## Конфигурация
Все параметры читаются из переменных окружения (см. `internal/config.go`):
- `HTTP_ADDR` (default `:8081`) — адрес HTTP-сервера.
- `PG_URL` (default `postgres://postgres:postgres@localhost:5432/orders?sslmode=disable`) — строка подключения к PostgreSQL.
- `KAFKA_BROKERS` (default `localhost:29092`) — список брокеров Kafka через запятую.
- `KAFKA_TOPIC` (default `orders`) — топик, который слушает consumer и куда пишет producer.
- `KAFKA_GROUP_ID` (default `order-svc`) — group id consumer'а.
- `CACHE_ENABLED` (default `true`) — включает/выключает использование in-memory кэша.

## База данных и миграции
- При запуске через Docker Compose файл `db/001_init.sql` автоматически применяется контейнером PostgreSQL.
- Команда `make migrate` перекинет SQL-файл в контейнер и выполнит его.
- Структура данных: `orders` (шапка), `deliveries`, `payments`, `items` (товары заказа).

## Kafka Producer
CLI-утилита `cmd/producer` генерирует псевдослучайные заказы и отправляет их в Kafka.
Пример запуска:
```bash
go run ./cmd/producer -n 10 -interval 500ms -brokers localhost:29092 -topic orders
```

## HTTP API
- `GET /order/{id}` — получить заказ. Возвращает `404`, если заказа нет.
- `GET /static/*` и `GET /` — отдача статических файлов из каталога `web/`.

## Дальнейшие улучшения
В ближайших задачах планируется:
- добавить README-разделы про миграции down и автоматический откат;
- подготовить `.env.example`;
- обработать ошибки, заменить deprecated API, внедрить валидацию, интерфейсы, тесты, DLQ, линтеры, трассировку и метрики.
