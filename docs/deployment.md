# Развёртывание на VPS

## Требования

- Linux VPS с Git, Docker Engine и плагином Docker Compose v2.
- Пользователь с правом запуска Docker.
- Для отправки на часы — сетевой маршрут от контейнера до TC002. Домашний IP и `.local`-имя обычно недоступны с VPS; используйте VPN-маршрут в домашнюю сеть или, после реализации MQTT, доступный обеим сторонам брокер.

## Первое развёртывание

На VPS:

```sh
git clone https://github.com/LikiPiki/tc002-apps.git
cd tc002-apps
cp deploy/.env.example deploy/.env
# Отредактируйте deploy/.env: адрес часов и интервал выполнения.
sh deploy/deploy.sh
```

Скрипт проверяет Compose-конфигурацию, собирает образ и запускает воркер в фоне. Перезапуск после перезагрузки обеспечивается `restart: unless-stopped` при работающем Docker daemon.

## Выбор приложений

Go-команды можно собирать независимо:

```sh
make list-apps
make build
make docker-build
```

Сборкой управляет `configs/apps.list`: имена папок из `cmd/`, по одному на строку; пустые строки и комментарии `#` допустимы. Например:

```text
worker
# clockctl
```

Если конфига нет, собираются **все** Go-приложения. Если конфиг есть — только перечисленные и их зависимости; пустой список или неизвестное имя считаются ошибкой. Начните с `cp configs/apps.list.example configs/apps.list`. Этот персональный конфиг не коммитится. Другой путь можно задать через `make build CONFIG=configs/vps.list` (отсутствующий файл также означает сборку всех приложений).

`make check` и локальное dev-окружение проверяют весь проект независимо от списка production-сборки. Существующие файлы в `bin/` и `dist/` автоматически не удаляются: переносите архивы именно выбранных приложений.

Для нового Docker-приложения достаточно папки `cmd/<name>`; для управляемого запуска через Compose также добавьте сервис с тем же именем и image `tc002-apps/<name>:local`.

На VPS укажите, какие сервисы развернуть:

```sh
sh deploy/deploy.sh worker
# Когда появятся другие сервисы: sh deploy/deploy.sh weather monitor
```

Скрипт собирает и обновляет только указанные сервисы, не запускает зависимости и не удаляет другие контейнеры. Без аргументов запускается только `worker`. `clockctl` — одноразовая CLI-утилита: используйте `compose run`, а не постоянный сервис.

## Перенос только готового образа

Чтобы на VPS не собирать Go и не отправлять исходники:

```sh
# На машине сборки:
cp configs/apps.list.example configs/apps.list # пример выбирает worker
make docker-save
scp dist/worker.tar.gz user@vps:/opt/tc002-apps/
# Также передайте deploy/compose.yaml и deploy/deploy.sh в /opt/tc002-apps/deploy/.
```

Создайте на VPS `deploy/.env` по примеру `deploy/.env.example`. Затем в `/opt/tc002-apps`:

```sh
docker load -i worker.tar.gz
sh deploy/deploy.sh --no-build worker
```

Архив содержит только выбранный runtime-образ, без исходников и компилятора. Для запуска нужны Compose-файл, скрипт и `.env`; полный Git checkout не требуется. Архитектура образа должна совпадать с VPS: сборка на ARM Mac по умолчанию даёт ARM64. Для x86-64 VPS собирайте на x86-64 машине или явно выберите платформу:

```sh
DOCKER_DEFAULT_PLATFORM=linux/amd64 make docker-save
```

Кросс-платформенная сборка требует поддержки эмуляции соответствующей архитектуры в Docker. При откате в этом режиме загрузите архив предыдущего образа и повторите запуск с `--no-build`.

Сейчас воркер только пишет `worker tick` в лог. Подключение к часам проверяется отдельно:

```sh
docker compose --env-file deploy/.env -f deploy/compose.yaml run --rm --build clockctl --text 'Hello from VPS'
```

`deploy/.env` исключён из Git и Docker build context. Храните реальные настройки на VPS.

## Логи и остановка

Все команды ниже выполняются из корня репозитория:

```sh
docker compose --env-file deploy/.env -f deploy/compose.yaml logs -f --tail=100 worker
docker compose --env-file deploy/.env -f deploy/compose.yaml ps
docker compose --env-file deploy/.env -f deploy/compose.yaml down
```

Логи ротируются: до трёх файлов по 10 МБ.

## Обновление

```sh
git rev-parse HEAD  # сохраните SHA текущей версии для отката
git pull --ff-only
sh deploy/deploy.sh
```

Исходники собираются непосредственно на VPS; registry и GitHub-токены для образов не нужны. Новые изменения должны быть опубликованы в репозитории до обновления VPS.

## Откат

В чистом checkout на VPS переключитесь на сохранённый SHA и повторите развёртывание:

```sh
git checkout <previous-sha>
sh deploy/deploy.sh
```

Для следующих обновлений вернитесь на рабочую ветку и выполните обычный процесс обновления.

## Одноразовый запуск воркера

```sh
docker compose --env-file deploy/.env -f deploy/compose.yaml run --rm --build worker --once
```

Go на VPS устанавливать не требуется: сборка выполняется в Docker. Каждая новая команда может использовать тот же Dockerfile с другим build arg `APP`.
