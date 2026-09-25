# Какие пресетные элементы можно удалить из кода

В этом starter пресеты находятся в профиле, проектной инфраструктуре и dev seed. Сначала удалите ненужную декларацию и все её ссылки, затем соответствующую настройку окружения и Compose, если она больше нигде не используется. Не удаляйте обязательные модули `core` и `admin`.

## Модули профиля

Сейчас профиль `starter` объявлен в [`backend/internal/profile/starter.go`](../backend/internal/profile/starter.go): он включает `core` и `admin`, а также привязывает кэш aliases Core (`core.durable` и `core.hot`) к хранилищу `shared`.

Удалять из состава профиля можно только дополнительные, необязательные модули, которые проект добавит позже. При удалении также уберите их зависимости, адаптеры, миграции/seeds и frontend/API ожидания. Core должен оставаться первым, Admin обязателен. Не удаляйте используемые Core cache bindings, пока не проверили, что Core может работать без них.

## Dev seed

Файлы [`backend/cmd/server/seeds/000001_starter.up.sql`](../backend/cmd/server/seeds/000001_starter.up.sql) и `000001_starter.down.sql` задают локальный сайт `localhost` и демо-пользователя `admin` с членством в защищённой группе `admin`. Seed включается только при `CMS_DEV_SEED=true`.

Если демо-данные не нужны, можно удалить эти SQL файлы или убрать ненужные записи из них. При этом проверьте `//go:embed seeds/*.sql` и передачу `SeedFiles` из `backend/cmd/server/main.go`, а также `SeedPlans` в `backend/internal/settings/settings.go`: проектный seed регистрируется только в режиме dev seed. Если удаляется весь seed, удалите связанную регистрацию/передачу файлов и описание dev-seed из README и deployment-документации. После изменения уже применённого seed на локальной базе пересоздайте dev-базу; этот seed не является механизмом обновления существующих данных.

## Инфраструктура starter

[`backend/internal/infrastructure/infrastructure.go`](../backend/internal/infrastructure/infrastructure.go) объявляет Kafka event bus, PostgreSQL и Core adapter, публичный и приватный localstorage, Redis cache и Argon2id password hasher. Это не произвольный список: соответствующие функции нужны текущей композиции приложения. Их можно убирать только вместе с потребителями и связанными декларациями:

- **Kafka/event bus** — после удаления событий и потребителей из подключённых модулей.
- **Redis/cache** — после изменения cache bindings профиля и всех использующих cache компонентов.
- **Public/private диски** — после проверки требований модулей к файлам, аватарам и загрузкам; настройки также заданы в `.env.example` и `compose.yaml`.
- **PostgreSQL/Core adapter** — не удалять из текущего starter: это основная база и хранилище обязательного Core.
- **Argon2id** — не удалять, пока Core отвечает за учётные записи с паролями.

Для каждого удалённого connector проверьте конфигурацию `backend/internal/settings`, переменные `.env.example`, сервисы и healthchecks в `compose.yaml`, Makefile, deployment docs и CI. Не оставляйте неиспользуемые переменные окружения и сервисы Compose.

