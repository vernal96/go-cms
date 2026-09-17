# Go CMS Start

Минимальный backend-проект на Go, подключающий публичный
`github.com/vernal96/go-cms-kernel`.

В состав входят Core, Admin, PostgreSQL, миграции, seed-данные, JWT и локальные
public/private filesystem-диски. Frontend admin развивается отдельно в
[`github.com/vernal96/go-cms-admin`](https://github.com/vernal96/go-cms-admin).

## Запуск

Требуются Docker и Docker Compose v2.

```sh
make up
```

Или вручную:

```sh
make env
docker compose --env-file .env up -d --build --wait
```

После запуска API доступен по адресу `http://localhost:8080`.

Dev-пользователь:

```text
login:    admin
password: admin-dev-only-2026
```

Пароль и значения `JWT_SIGNING_KEY`, `FILES_PRIVATE_SIGNING_KEY` в `.env` —
только для разработки. Перед deployment замените их и настройте production
storage, logger, event bus и PostgreSQL.

## Архитектура

`backend/cmd/server/main.go` собирает `app.Definition` из публичных kernel
контрактов:

- Core и Admin — обязательные модули профиля;
- Core PostgreSQL adapter — persistence;
- localstorage — public/private disks;
- project seed — сайт `localhost` и admin group membership;
- JWT — `/api/auth/login` и защищённые Admin API;
- `/healthz` — контейнерный healthcheck.

Backend не импортирует frontend source и не знает npm-пакеты. Админка подключается
по HTTP API. Инструкции по установке admin package и host находятся в отдельном
репозитории `go-cms-admin`.

Для локального запуска админки рядом с backend:

```sh
cd ../go-cms-admin
cp .env.example .env
# ADMIN_API_TARGET=http://localhost:8080 задан в .env
npm ci
npm run dev -- --host 0.0.0.0
```

Откройте `http://localhost:5173`. В Docker-варианте admin host запускается
командой `ADMIN_API_TARGET=http://host.docker.internal:8080 docker compose up
--build` из admin-репозитория.

## Добавление модулей

Добавьте модуль в `kernel.Profile.Modules` после Core и объявите зависимости.
Его database adapter подключите в `app.DatabaseDefinition.Adapters`, а
проектные migrations/seeds — через соответствующие публичные kernel providers.
Не используйте импорты из удалённого `internal` текущего проекта.

## Команды

```sh
make ps       # статус контейнеров
make logs     # логи
make check    # Go checks и compose config
make down     # остановить, сохранив volumes
docker compose down -v  # удалить БД и файлы
```

## Проверка независимости

```sh
GOWORK=off go -C backend test ./...
GOWORK=off go -C backend vet ./...
GOWORK=off go -C backend build ./...
docker compose --env-file .env.example config --quiet
```

В итоговых manifests нет `replace` и зависимостей от прежнего module path.
Проверка kernel release требует доступного
remote и опубликованного тега `v0.1.0`.
