# tc002-apps

Личные приложения для Ulanzi TC002 с AWTRIX NG и Go-сервисы обработки данных для VPS.

## Структура

```text
apps/                Berry-приложения, исполняемые на часах
assets/icons/        Иконки приложений
cmd/worker/          Заглушка периодического обработчика для VPS
cmd/clockctl/        CLI отправки уведомлений на часы
internal/awtrix/     HTTP-клиент AWTRIX NG
deploy/              Docker Compose, пример окружения, скрипт развёртывания
docs/                Архитектура, настройка, развёртывание
```

## Быстрый старт

Требуется Go 1.24+; для контейнеров — Docker с Compose v2.

```sh
make check
make list-apps
make build
make docker-build
go run ./cmd/worker --once
AWTRIX_URL=http://awtrixng-a1b2c3.local go run ./cmd/clockctl --text 'Hello from Go'
```

`worker` пока только пишет отметку выполнения в лог: получение данных и публикация появятся вместе с первым приложением. `clockctl` уже отправляет уведомления через HTTP API AWTRIX NG.

Выбор Go-приложений задаётся в `configs/apps.list` (по имени из `cmd/` на строку). Если файла нет, собираются все. Начните с `cp configs/apps.list.example configs/apps.list` — пример выбирает только `worker`. `make docker-save` создаёт `dist/<app>.tar.gz` для переноса выбранных образов на VPS. Подробности — в [инструкции развёртывания](docs/deployment.md).

## Docker

```sh
cp deploy/.env.example deploy/.env
docker compose --env-file deploy/.env -f deploy/compose.yaml up -d --build
docker compose --env-file deploy/.env -f deploy/compose.yaml logs -f worker
```

## Локальная проверка без часов

```sh
make dev-check
# Откройте http://localhost:8080 — виртуальная матрица TC002 52 × 16.
make dev-notify TEXT="Hello TC002"
make dev-down
```

Dev-окружение собирает настоящий headless AWTRIX NG в Alpine-контейнере. Подробности и ограничения — в [руководстве по локальной разработке](docs/development.md). Все команды: `make help`.

Подробнее: [развёртывание на VPS](docs/deployment.md), [настройка часов](docs/setup.md), [архитектура](docs/architecture.md).
