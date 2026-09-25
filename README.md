# Go CMS Start

Документация проекта: [развертывание с нуля](docs/deployment.md),
[удаление пресетных элементов](docs/removing-presets.md),
[работа с профилями](docs/profiles.md). Полный список — в [навигации по документации](docs/README.md).

Минимальный backend на Go с отдельной зависимостью
[`go-cms-kernel v0.2.0`](https://github.com/vernal96/go-cms-kernel).
Включает Core, Admin, PostgreSQL, Redis, Kafka, миграции, опциональный dev seed, JWT и public/private диски.
[Админка](https://github.com/vernal96/go-cms-admin) — отдельное приложение,
подключаемое по HTTP.

## Запуск backend с нуля

Нужны Git, Python 3, GNU Make, Docker и Docker Compose v2 с поддержкой `--wait`.
Локальные Go и Node для Docker-сценария не требуются.

```sh
git clone https://github.com/vernal96/go-cms.git
cd go-cms
make up
```

`make up` создаёт `.env` из `.env.example`, генерирует уникальные секреты, собирает backend, запускает PostgreSQL, Redis и Kafka
и дожидается healthcheck. На новой базе автоматически применяются миграции и seed.
Ядро скачивается по версии из `backend/go.mod`; соседняя папка с ядром,
`replace` и `go.work` не нужны.

- API: `http://localhost:8080`.
- Проверка запуска: `http://localhost:8080/healthz` → HTTP 200.
- Dev-пользователь `admin` / `admin-dev-only-2026` создаётся только при явном `CMS_DEV_SEED=true`.
- Для обычного запуска dev-seed выключен. `make env` создаёт уникальные секреты.
- Первого администратора создайте командой `docker compose --env-file .env run --rm -e CMS_ADMIN_PASSWORD server bootstrap-admin`, предварительно задав `CMS_ADMIN_PASSWORD` в окружении. Логин и email можно задать через `-e CMS_ADMIN_LOGIN -e CMS_ADMIN_EMAIL`. Существующие аккаунты команда не перезаписывает.

Backend отдаёт API, а не HTML админки или готовую главную страницу сайта.
Ответ 404 на `/` не означает сбой запуска: starter не создаёт главную страницу.

Ручной эквивалент:

```sh
make env
docker compose --env-file .env up -d --build --wait
```

## Запуск админки

В соседней директории:

```sh
cd ..
git clone https://github.com/vernal96/go-cms-admin.git
cd go-cms-admin
ADMIN_API_TARGET=http://host.docker.internal:8080 docker compose up -d --build --wait
```

Откройте `http://localhost:5173` и войдите созданным администратором. Команда явно передаёт
адрес backend, доступный из контейнера, даже если ранее был создан локальный `.env`.

Для запуска админки без Docker нужен **Node.js >=24** и npm:

```sh
cd ../go-cms-admin
cp .env.example .env
npm ci
npm run dev
```

В этом режиме `.env` содержит `ADMIN_API_TARGET=http://localhost:8080`.
Настройки `.env` загружаются Vite, переменные процесса имеют приоритет.

## Порты и независимые копии

Compose берёт имя project из имени директории. Для независимых копий, особенно
если их директории одинаково называются, задавайте разные `COMPOSE_PROJECT_NAME`
и свободные порты в `.env` или через переменные процесса. Имена контейнеров,
образов, network и volumes зависят от project:

```sh
COMPOSE_PROJECT_NAME=second-cms SERVER_PORT=18080 make up
BASE_URL=http://localhost:18080 make smoke
```

Для второй админки:

```sh
ADMIN_PORT=15173 ADMIN_API_TARGET=http://host.docker.internal:18080 docker compose -p second-admin up -d --build --wait
```

`SERVER_PORT` — опубликованный порт backend на хосте; внутри контейнера используется
8080. `ADMIN_PORT` — порт админки на хосте; внутри её контейнера используется 5173.
Для локального `npm run dev` значение `ADMIN_PORT` из `.env` задаёт порт Vite.

Если `FILES_PUBLIC_BASE_URL` / `FILES_PRIVATE_BASE_URL` оставлены пустыми, Compose
использует `http://localhost:<SERVER_PORT>`. Для другого домена или HTTPS явно
укажите оба URL в `.env` и пересоздайте backend командой `make up`.

## Публичный сайт и dev-данные

При `CMS_DEV_SEED=true` seed создаёт сайт `localhost` и пользователя в защищённой группе `admin`.
Пароль пользователя задан в seed, **не в `.env`**. Изменить его можно в админке
в разделе «Пользователи». Обычный повторный запуск не применяет seed заново
и не сбрасывает изменённый пароль.

Гостевые права по умолчанию отсутствуют. Даже у сайта с `is_public=true` созданная
страница для гостя вернёт 403 до настройки прав чтения. Для проекта с публичным
контентом добавьте в проектный seed перед первым запуском:

```sql
INSERT INTO core.guest_permissions (permission_code)
VALUES ('core.site.read'), ('core.resource.read')
ON CONFLICT DO NOTHING;
```

Это разрешает гостям читать опубликованные ресурсы публичных сайтов;
приватные сайты и management API остаются защищены. Не выдавайте гостям
`admin.panel.read` или права управления файлами. При изменении уже применённого
seed на dev-стенде нужно пересоздать его базу; команда `docker compose down -v`
удаляет данные и файлы **этого Compose project**.

Значения секретов в `.env.example` предназначены только для примеров; `make env`
генерирует уникальные значения. Kafka в Compose работает с одним брокером;
для отказоустойчивого развёртывания настройте репликацию инфраструктуры. Docker-админка запускает Vite
для разработки; production frontend собирается отдельно через `npm run build`.

## Проверки

Кэш подключён через модульные aliases `core.durable` и `core.hot` к одному Redis.
Compose использует внутренний адрес `redis:6379`; Redis не публикуется на хосте.
При запуске backend через Go задайте `REDIS_ADDR` и при необходимости
`REDIS_PASSWORD`. Данные Redis являются расходным кэшем; для них не создаётся
постоянный volume.

Версия ядра фиксируется в `backend/go.mod`. Ядро v0.2.0 включает раздельные
кэши URL, ресурсов, конфигурации виджетов и опциональных результатов виджетов.

Для smoke-проверок нужны Python 3; для `make check` — Go версии из `backend/go.mod`
(1.26.1 или новее) либо рабочая автоматическая загрузка Go toolchain.

```sh
CMS_DEV_SEED=true make up # изолированное локальное демо для smoke
make smoke             # health, вход, API, отрицательные проверки доступа
make check             # Go test/vet/build и Compose config
make test-deployment   # чистая копия, новая БД/диски, запуск и проверка сохранности
```

`make test-deployment` копирует текущие tracked и неигнорируемые новые файлы в
временную директорию, использует отдельный Compose project и порт 18080,
создаёт ресурс и файлы на обоих дисках, пересоздаёт контейнеры с сохранением
volumes и проверяет записанные данные. После проверки удаляет только свой
временный project, volumes и директорию. Порт можно изменить:

```sh
DEPLOYMENT_TEST_PORT=28080 make test-deployment
```

В CI выполняются `make check` и `make test-deployment`. Это проверяет реальную
зависимость v0.2.0 без локальных подмен.

## Логи и остановка

```sh
make ps        # статус контейнеров
make logs      # Docker-логи, включая фатальные ошибки backend
make app-logs  # JSON-лог работающего приложения
make down      # остановить, сохранив БД и файлы
```

Полный лог приложения хранится в `LOGGER_FILE_PATH` (по умолчанию
`/app/var/log/cms.log` внутри контейнера). Если контейнер завершился, сохраните лог:

```sh
docker compose cp server:/app/var/log/cms.log ./cms.log
```

Если сборка не может получить `go-cms-kernel@v0.2.0`, проверьте доступ к GitHub,
`proxy.golang.org` и `sum.golang.org`. Ошибки `Repository not found` / HTTP 404
означают, что исходники или тег недоступны; наличие локального ядра не заменяет
проверку опубликованной зависимости.

## Архитектура и расширение

`backend/cmd/server/main.go` собирает `app.Definition` через публичные контракты
ядра: Core/Admin, PostgreSQL adapter, localstorage, JWT и проектные seeds.
`backend/internal` содержит только проектную конфигурацию/адаптеры.
Frontend не встраивается в Go binary и взаимодействует с backend через HTTP.

Добавляйте модули в `kernel.Profile.Modules` после Core, явно указывая зависимости.
Database adapters подключаются в `app.DatabaseDefinition.Adapters`, проектные
migrations/seeds — через публичные kernel providers. Не импортируйте `internal`
другого репозитория.

## Версия 0.2.0

[Исправления аудита и результаты проверок](docs/audit-fixes-0.2.0.md).

Этот выпуск меняет схему dev-БД и контракт авторизации. Для старого dev-стенда
пересоздайте данные; сохранение старых схем в pre-production не поддерживается.
Обновляйте backend и admin вместе. Ранее выданные JWT больше не являются сессиями.

- `CMS_DEV_SEED=false` по умолчанию; для локального демо: `CMS_DEV_SEED=true make up`.
- Kafka 4.3.1 входит в Compose и хранит данные в `kafka_data`; брокер доступен
  только во внутренней сети. Один брокер предназначен для разработки; для HA
  задайте внешний кластер через конфигурацию проекта.
- JWT живёт 30 минут по умолчанию. `POST /api/auth/logout` отзывает текущую сессию;
  смена пароля отзывает все. Сессии хранятся в PostgreSQL.
- Каждая реплика сверяет версию сайта перед использованием runtime. Пока локальная
  версия устарела, запрос получает 503; фоновая синхронизация запускается раз в секунду.
- Redis ограничен 256 MiB с eviction. Исчезновение поколения инвалидирует запись,
  а число ключей поколений ограничено 65 536 на физическое хранилище.
- Вход ограничен двумя параллельными проверками, 30 попытками на IP в минуту и
  10 попытками на пару IP/логин. Используется RemoteAddr; не подставляйте
  недоверенные forwarded-заголовки. При reverse proxy учитывайте лимиты на его адрес.
- Обработка изображений ограничена двумя параллельными операциями на приложение.

`make test-deployment` явно включает dev-seed только на изолированном тестовом стенде.
