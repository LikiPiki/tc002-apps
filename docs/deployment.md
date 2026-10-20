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
