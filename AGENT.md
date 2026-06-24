Ты работаешь в репозитории `vernal96/go-cms`.

Нужно реализовать следующий этап GO CMS: подключить реальные инфраструктурные адаптеры, зарегистрировать их в проектной инфраструктуре, затем начать внедрение ресурсов CMS с миграциями и репозиториями.

Важные архитектурные правила проекта:

1. `core` не должен зависеть от PostgreSQL, Redis, Kafka, S3, локальной файловой системы и других конкретных технологий.
2. Конкретные реализации должны лежать в `adapters/...`.
3. `App` является composition root и хранит инфраструктуру через интерфейсы.
4. Модули получают инфраструктуру через `App` / `ModuleContext`, но не создают её сами.
5. Сайт определяется записью в БД + `SiteProfile` в коде.
6. Каждый сайт получает отдельный `SiteRuntime`.
7. `Registry` должен быть site-scoped.
8. Не использовать глобальный mutable singleton для сайта/runtime.
9. Работать маленькими шагами, без большого неуправляемого рефакторинга.
10. Сохранять Go version `1.26.1`.

Перед началом:

1. Проверь текущее состояние репозитория.
2. Найди существующий файл миграций в проекте. Возможные ориентиры:

    * `0_cms_init.php`
    * `migrations`
    * `database`
    * `schema`
    * `sites`
    * `resources`
3. Используй найденный файл миграций как главный источник структуры БД.
4. Если в промпте ниже есть расхождение с найденным файлом миграции, приоритет у файла миграции.
5. После каждого изменения запускай:

    * `gofmt`
    * `go mod tidy`
    * `go test ./...`

---

## Этап 1. Внедрить Logger в App

Сейчас в `core` должен быть или уже есть интерфейс `Logger`.

Если `core/logger.go` отсутствует, создать:

```go
package core

type Logger interface {
	Debug(message string, fields ...LogField)
	Info(message string, fields ...LogField)
	Warn(message string, fields ...LogField)
	Error(message string, fields ...LogField)
}

type LogField struct {
	Key   string
	Value any
}

type NullLogger struct{}

func (l NullLogger) Debug(message string, fields ...LogField) {}
func (l NullLogger) Info(message string, fields ...LogField) {}
func (l NullLogger) Warn(message string, fields ...LogField) {}
func (l NullLogger) Error(message string, fields ...LogField) {}

var _ Logger = NullLogger{}
```

Если `Logger` уже есть, не дублировать, а привести к аккуратному виду.

Обновить `core/app.go`:

1. Добавить поле `logger Logger`.
2. Добавить аргумент `logger Logger` в `NewApp`.
3. Проверять `logger != nil`.
4. Добавить метод:

```go
func (a *App) Logger() Logger
```

Если в проекте уже используется `NullLogger`, использовать его как default в project bootstrap.

---

## Этап 2. Обновить InfrastructureRegistry

В `internal/project/infrastructure_registry.go`:

1. Добавить поле:

```go
logger core.Logger
```

2. В `NewInfrastructureRegistry()` по умолчанию установить:

```go
logger: core.NullLogger{}
```

3. Добавить метод:

```go
func (r *InfrastructureRegistry) UseLogger(logger core.Logger)
```

Метод должен игнорировать или panic/error на `nil` в стиле текущего файла. Выбери стиль, который уже используется в этом registry.

4. Добавить getter:

```go
func (r *InfrastructureRegistry) Logger() core.Logger
```

В `internal/project/bootstrap.go` обновить вызов `core.NewApp(...)`, чтобы туда передавался logger.

---

## Этап 3. Добавить stdout/slog logger adapter

Создать пакет:

```txt
adapters/logger/stdoutlogger
```

Файл:

```txt
adapters/logger/stdoutlogger/logger.go
```

Реализация должна использовать стандартный `log/slog`.

Требования:

1. Адаптер реализует `core.Logger`.
2. Не тянуть сторонние зависимости.
3. Конструктор:

```go
func New(logger *slog.Logger) *Logger
```

4. Если `logger == nil`, использовать:

```go
slog.Default()
```

5. Преобразовывать `[]core.LogField` в `[]any` для slog.

Пример API:

```go
type Logger struct {
	logger *slog.Logger
}

func (l *Logger) Debug(message string, fields ...core.LogField)
func (l *Logger) Info(message string, fields ...core.LogField)
func (l *Logger) Warn(message string, fields ...core.LogField)
func (l *Logger) Error(message string, fields ...core.LogField)
```

В конце файла добавить compile-time check:

```go
var _ core.Logger = (*Logger)(nil)
```

---

## Этап 4. Добавить local file storage adapter

Создать пакет:

```txt
adapters/storage/localstorage
```

Файл:

```txt
adapters/storage/localstorage/storage.go
```

Реализовать `core.FileStorage`.

Перед реализацией проверь актуальный интерфейс `core.FileStorage`. По текущей архитектуре он должен быть примерно таким:

```go
type FileStorage interface {
	Save(ctx context.Context, path string, content io.Reader) error
	Open(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error
	Exists(ctx context.Context, path string) (bool, error)
}
```

Если интерфейс отличается, реализовать именно текущий интерфейс.

Требования к local storage:

1. Конструктор:

```go
func NewStorage(root string) (*Storage, error)
```

2. `root` не должен быть пустым.
3. `Save` должен создавать директории через `os.MkdirAll`.
4. `Open` должен возвращать `core.ErrFileNotFound`, если файла нет.
5. `Delete` должен возвращать nil, если файл уже отсутствует.
6. `Exists` должен корректно отличать отсутствие файла от другой ошибки.
7. Защититься от path traversal:

    * чистить путь через `filepath.Clean`;
    * не позволять выйти выше root.
8. Для локального dev использовать root вроде:

```txt
./var/storage
```

9. Добавить compile-time check:

```go
var _ core.FileStorage = (*Storage)(nil)
```

---

## Этап 5. Добавить Redis cache adapter

Создать пакет:

```txt
adapters/cache/rediscache
```

Файл:

```txt
adapters/cache/rediscache/store.go
```

Использовать:

```go
github.com/redis/go-redis/v9
```

Перед реализацией проверь текущий интерфейс `core.CacheStore`. По текущей архитектуре он должен быть примерно таким:

```go
type CacheStore interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}
```

Если интерфейс отличается, реализовать именно текущий интерфейс.

Требования:

1. Константа:

```go
const StoreName core.CacheStoreName = "redis"
```

2. Тип:

```go
type Store struct {
	client redis.UniversalClient
}
```

3. Конструктор:

```go
func NewStore(client redis.UniversalClient) (*Store, error)
```

4. Если client nil — вернуть ошибку.
5. `Get`:

    * при `redis.Nil` возвращать `nil, false, nil`;
    * при найденном значении возвращать `value, true, nil`.
6. `Set` должен использовать ttl из аргумента.
7. `Delete` должен удалить ключ.
8. Добавить `Ping(ctx)` или отдельную helper-функцию, если это удобно для dev bootstrap.
9. Добавить compile-time check.

---

## Этап 6. Добавить Kafka EventBus adapter

Создать пакет:

```txt
adapters/eventbus/kafkaeventbus
```

Файл:

```txt
adapters/eventbus/kafkaeventbus/bus.go
```

Использовать:

```go
github.com/segmentio/kafka-go
```

Перед реализацией проверь текущий интерфейс `core.EventBus`. По текущей архитектуре он должен быть примерно таким:

```go
type EventBus interface {
	Publish(ctx context.Context, event Event) error
	Subscribe(name EventName, handler EventHandler) error
}
```

Требования:

1. Реализовать `core.EventBus`.
2. Не усложнять чрезмерно consumer loop на первом шаге.
3. Минимальная реализация:

    * `Publish` отправляет событие в Kafka.
    * `Subscribe` регистрирует handler локально, чтобы интерфейс был выполнен.
    * Если возможно без сильного усложнения, добавить `Run(ctx)` для чтения сообщений из Kafka и вызова handlers.
4. События сериализовать в JSON.
5. В message key использовать `event.Name`.
6. Конфиг:

```go
type Config struct {
	Brokers []string
	Topic   string
	GroupID string
}
```

7. Конструктор:

```go
func NewBus(config Config) (*Bus, error)
```

8. Валидировать:

    * brokers не пустой;
    * topic не пустой.
9. Добавить `Close() error`.
10. Добавить compile-time check.

Важно: `core.EventBus` не должен знать о Kafka.

---

## Этап 7. Обновить docker-compose.yml

Добавить реальные dev-сервисы:

1. PostgreSQL уже должен быть.
2. Добавить Redis.
3. Добавить Kafka.

Требования:

* PostgreSQL оставить на актуальной major-версии, которая уже используется в проекте.
* Redis можно использовать официальный образ.
* Kafka можно использовать простой single-node dev вариант.
* Не добавлять сложный production cluster.
* Добавить healthcheck там, где это разумно.
* Добавить volumes для сохранения данных.

Пример целевой структуры:

```yaml
services:
  postgres:
    ...

  redis:
    image: redis:latest
    ports:
      - "6379:6379"

  kafka:
    ...
    ports:
      - "9092:9092"
```

Если используешь Kafka image, выбери вариант, который реально запускается в single-node режиме без Zookeeper, например современный KRaft-вариант. Проверь docker-compose на корректность.

---

## Этап 8. Обновить .env.example

Добавить переменные:

```env
GO_CMS_HTTP_ADDR=:8080

GO_CMS_DATABASE_DSN=postgres://go_cms:go_cms@localhost:5432/go_cms?sslmode=disable

GO_CMS_REDIS_ADDR=localhost:6379
GO_CMS_REDIS_PASSWORD=
GO_CMS_REDIS_DB=0

GO_CMS_STORAGE_LOCAL_ROOT=./var/storage

GO_CMS_KAFKA_BROKERS=localhost:9092
GO_CMS_KAFKA_TOPIC=cms-events
GO_CMS_KAFKA_GROUP_ID=go-cms
```

Если в проекте уже есть другой формат env — сохранить стиль проекта.

---

## Этап 9. Обновить dev infrastructure registration

В `internal/registry/dev.go` заменить временные in-memory адаптеры на реальные dev-адаптеры:

1. Cache:

    * создать Redis client;
    * создать `rediscache.Store`;
    * зарегистрировать store как `rediscache.StoreName`;
    * назначить `core.CacheScopeDefault` на Redis store;
    * если testmodule использует свой scope — тоже направить его на Redis store.

2. File storage:

    * создать local storage по root из env;
    * зарегистрировать disk для testmodule или default local disk.

3. EventBus:

    * создать Kafka event bus;
    * `UseEventBus(kafkaBus)`.

4. Logger:

    * создать `stdoutlogger.New(slog.Default())`;
    * `UseLogger(logger)`.

Если в `internal/registry/dev.go` сейчас нет доступа к env, можно:

* добавить простой helper env в отдельный маленький файл;
* или оставить значения по умолчанию в dev registry;
* но лучше читать из env в `cmd/app/main.go` и передавать config, если это не ломает текущую простоту.

Не усложнять: для первого шага допустимы dev defaults.

---

## Этап 10. Начать внедрение Resources

Нужно добавить базовую модель ресурсов и первый репозиторий, но не строить всю CMS сразу.

### 10.1. Найти и изучить миграционный файл проекта

Перед созданием миграций обязательно найти существующий файл миграций в проекте. Он должен содержать или подразумевать таблицы:

* `sites`
* `users`
* `user_groups`
* `site_permissions`
* `file_folders`
* `files`
* `media`
* `resources`
* `resource_field_values`
* `resource_permissions`
* `redirects`
* `resource_widgets`
* `template_widgets`

Использовать найденный файл как источник истины.

### 10.2. Добавить core-модели ресурсов

Создать или обновить файлы в `core`:

```txt
core/resource.go
core/resource_repository.go
```

Минимальная модель:

```go
type ResourceID int64
type ResourceType string
type ResourceStatus string

type Resource struct {
	ID          ResourceID
	SiteID      int64
	ParentID    *ResourceID
	Type        ResourceType
	Template    string
	Title       string
	Alias       string
	Path        string
	Sort        int
	IsPublished bool
	Settings    map[string]any
	SEO         map[string]any
}
```

Если в миграции другие поля — адаптировать под миграцию.

Интерфейс:

```go
type ResourceRepository interface {
	FindByID(ctx context.Context, id ResourceID) (Resource, error)
	FindByPath(ctx context.Context, siteID int64, path string) (Resource, error)
	FindChildren(ctx context.Context, parentID ResourceID) ([]Resource, error)
}
```

Добавить ошибку:

```go
var ErrResourceNotFound = errors.New("resource not found")
```

Не добавлять PostgreSQL в core.

### 10.3. Добавить Postgres ResourceRepository adapter

Создать пакет:

```txt
adapters/resource/postgresresource
```

Файл:

```txt
adapters/resource/postgresresource/repository.go
```

Использовать `pgxpool`, как уже сделано для site repository.

Методы:

* `FindByID`
* `FindByPath`
* `FindChildren`
* опционально `Migrate`, если текущий проект пока использует такой подход для репозиториев.

Важно: если в проекте уже есть общий `pgxpool` для site repository, не создавать отдельный пул без необходимости. Лучше использовать общий pool или общий postgres infrastructure object.

---

## Этап 11. Миграции БД

Опирайся на существующий файл миграций проекта. Если нужно перенести PHP/старую миграцию в Go/Postgres SQL, создай SQL-миграции или текущий проектный migration mechanism.

Минимально должны быть такие таблицы.

### 11.1. sites

Назначение: сайты CMS.

Поля ориентировочно:

```txt
id
profile_code
domain
locale
settings jsonb
created_at
updated_at
created_by
updated_by
```

Индексы:

```txt
unique(domain)
index(profile_code)
```

### 11.2. users

Назначение: пользователи CMS.

Поля ориентировочно:

```txt
id
email
password_hash
name
avatar_media_id nullable
settings jsonb
created_at
updated_at
created_by
updated_by
```

Индексы:

```txt
unique(email)
index(avatar_media_id)
```

Связи:

```txt
avatar_media_id -> media.id ON DELETE SET NULL
```

Если в существующей миграции другая структура users — использовать её.

### 11.3. user_groups

Назначение: связь пользователь → группа.

Поля:

```txt
user_id
group_code
created_at
updated_at
created_by
updated_by
```

Индексы:

```txt
primary или unique(user_id, group_code)
index(group_code)
```

Связи:

```txt
user_id -> users.id ON DELETE CASCADE
created_by -> users.id ON DELETE SET NULL
updated_by -> users.id ON DELETE SET NULL
```

Группы декларируются в коде, поэтому отдельная таблица групп не обязательна, если её нет в исходной миграции.

### 11.4. site_permissions

Назначение: права групп на сайт.

Поля:

```txt
site_id
group_code
permission
created_at
updated_at
created_by
updated_by
```

Индексы:

```txt
unique(site_id, group_code, permission)
index(group_code)
index(permission)
```

Связи:

```txt
site_id -> sites.id ON DELETE CASCADE
```

### 11.5. file_folders

Назначение: виртуальные папки файлового менеджера.

Поля:

```txt
id
parent_id nullable
disk
name
sort
created_at
updated_at
created_by
updated_by
```

Индексы:

```txt
index(parent_id)
index(disk)
unique(parent_id, disk, name)
```

Связи:

```txt
parent_id -> file_folders.id ON DELETE RESTRICT
created_by -> users.id ON DELETE SET NULL
updated_by -> users.id ON DELETE SET NULL
```

### 11.6. files

Назначение: физические файлы.

Поля ориентировочно:

```txt
id
folder_id nullable
disk
path
name
original_name
mime_type
extension
size
checksum
metadata jsonb
created_at
updated_at
created_by
updated_by
```

Индексы:

```txt
index(folder_id)
index(disk)
unique(disk, path)
```

Связи:

```txt
folder_id -> file_folders.id ON DELETE SET NULL
```

### 11.7. media

Назначение: медиа-объекты, ссылающиеся на files.

Поля ориентировочно:

```txt
id
file_id
title
alt
caption
is_transformed
metadata jsonb
created_at
updated_at
created_by
updated_by
```

Индексы:

```txt
index(file_id)
```

Связи:

```txt
file_id -> files.id ON DELETE RESTRICT
```

Один file может ссылаться на множество media.

### 11.8. resources

Назначение: дерево ресурсов сайта.

Поля ориентировочно:

```txt
id
site_id
parent_id nullable
type
template
title
alias
path
sort
is_published
published_at nullable
settings jsonb
seo jsonb
created_at
updated_at
created_by
updated_by
```

Индексы:

```txt
index(site_id)
index(parent_id)
index(type)
index(template)
index(is_published)
unique(site_id, path)
```

Связи:

```txt
site_id -> sites.id ON DELETE CASCADE
parent_id -> resources.id ON DELETE RESTRICT
created_by -> users.id ON DELETE SET NULL
updated_by -> users.id ON DELETE SET NULL
```

Важно:

* сайт не является папкой;
* resources принадлежат site через `site_id`;
* path должен быть уникален внутри сайта;
* SEO хранить в `seo jsonb`, если так указано в текущей миграции.

### 11.9. resource_field_values

Назначение: значения полей ресурса.

Поля:

```txt
id
resource_id
field
value jsonb
created_at
updated_at
```

Индексы:

```txt
index(resource_id)
unique(resource_id, field)
```

Связи:

```txt
resource_id -> resources.id ON DELETE CASCADE
```

### 11.10. resource_permissions

Назначение: права групп на ресурс.

Поля:

```txt
resource_id
group_code
permission
created_at
updated_at
created_by
updated_by
```

Индексы:

```txt
unique(resource_id, group_code, permission)
index(group_code)
index(permission)
```

Связи:

```txt
resource_id -> resources.id ON DELETE CASCADE
```

### 11.11. redirects

Назначение: редиректы сайта.

Поля:

```txt
id
site_id
from_path
to_path
status_code
is_active
created_at
updated_at
created_by
updated_by
```

Индексы:

```txt
unique(site_id, from_path)
index(site_id)
index(is_active)
```

Связи:

```txt
site_id -> sites.id ON DELETE CASCADE
```

Важно:

* использовать одно поле `to_path`, а не разделять internal/external url, если текущая миграция уже так спроектирована;
* status_code ограничить допустимыми redirect-кодами, если удобно.

### 11.12. resource_widgets

Назначение: виджеты, прикреплённые к конкретному ресурсу.

Поля:

```txt
id
resource_id
widget
template
params jsonb
sort
area
created_at
updated_at
created_by
updated_by
```

Индексы:

```txt
index(resource_id)
index(widget)
index(area)
index(sort)
```

Связи:

```txt
resource_id -> resources.id ON DELETE CASCADE
```

### 11.13. template_widgets

Назначение: виджеты, прикреплённые к шаблону ресурса.

Поля:

```txt
id
resource_template
widget
template
params jsonb
sort
area
created_at
updated_at
created_by
updated_by
```

Индексы:

```txt
index(resource_template)
index(widget)
index(area)
index(sort)
```

Важно:

* `resource_template` — строковый код шаблона из registry/code;
* `widget` — строковый код виджета из registry/code;
* `template` — строковый код шаблона самого виджета.

---

## Этап 12. Первый resource vertical slice

После миграций и ResourceRepository сделать минимальный вертикальный срез:

1. При старте dev app создать тестовый ресурс для `localhost`, если его нет.
2. Добавить core/controller route:

```txt
GET /_cms/resource
```

или:

```txt
GET /_cms/resources/current?path=/
```

Выбери вариант, который лучше ложится на текущую структуру контроллеров.

3. HTTP server уже определяет site по host.
4. Controller должен брать текущий `runtime.Site()`.
5. Через `ResourceRepository` найти ресурс по `site.ID` и path.
6. Вернуть JSON с данными ресурса.

Важно:

* контроллер/модуль не должен знать, что это PostgreSQL;
* он должен работать через интерфейс `core.ResourceRepository`;
* конкретный Postgres adapter подключается в project bootstrap.

Если для этого нужно добавить `RepositoryManager` в `App`, делай минимальный вариант:

```go
type RepositoryManager interface {
	Sites() SiteRepository
	Resources() ResourceRepository
}
```

Но не усложняй, если можно аккуратно внедрить только `ResourceRepository` отдельным полем в App.

---

## Этап 13. Обновить README_DEV_VERTICAL_SLICE.md

Добавить инструкции:

```bash
docker compose up -d postgres redis kafka
go mod tidy
go run ./cmd/app
curl http://localhost:8080/_cms/site
curl "http://localhost:8080/_cms/resource?path=/"
```

Также указать, какие env-переменные можно поменять.

---

## Этап 14. Проверка

В конце обязательно:

1. `gofmt` на все изменённые `.go` файлы.
2. `go mod tidy`.
3. `go test ./...`.
4. Проверить, что `go.mod` содержит только нужные зависимости:

    * `github.com/jackc/pgx/v5`
    * `github.com/redis/go-redis/v9`
    * `github.com/segmentio/kafka-go`
5. Проверить, что `core` не импортирует:

    * pgx
    * redis
    * kafka-go
    * os/path/filepath как часть конкретного storage adapter
6. Проверить, что конкретные технологии находятся только в `adapters/...` или project bootstrap.
7. Проверить, что `SiteRuntime` и `Registry` остаются site-scoped.
8. Проверить, что не появился глобальный singleton текущего сайта.

---

## Ожидаемый результат

После реализации должно быть возможно:

```bash
docker compose up -d
go run ./cmd/app
curl http://localhost:8080/_cms/site
curl "http://localhost:8080/_cms/resource?path=/"
```

CMS должна:

1. Поднять App с реальными dev-инфраструктурными адаптерами.
2. Подключиться к PostgreSQL.
3. Использовать Redis как cache store.
4. Использовать local storage как file storage.
5. Использовать stdout/slog logger.
6. Использовать Kafka как EventBus.
7. Найти site по Host.
8. Собрать SiteRuntime по SiteProfile.
9. Найти route в runtime registry.
10. Вызвать controller.
11. Для `/ _cms/site` вернуть данные сайта.
12. Для resource route вернуть данные ресурса из PostgreSQL.

Не делай большой переписывающий рефакторинг. Если где-то нужно выбрать между идеальной архитектурой и маленьким безопасным шагом, выбирай маленький безопасный шаг и оставляй TODO-комментарий.
