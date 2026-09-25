# Работа с профилями

Профиль описывает набор модулей приложения и их bindings. В этом проекте профиль `starter` объявлен переменной `profile.Starter` в [`backend/internal/profile/starter.go`](../backend/internal/profile/starter.go), а `backend/internal/settings/settings.go` передаёт его в декларацию приложения. Сайты ссылаются на профиль по `profile_code`; текущий seed создаёт сайт с кодом `starter`.

## Создание профиля

1. Добавьте декларацию в `backend/internal/profile`, например `editorial.go`:

   ```go
   package profile

   import (
       kernel "github.com/vernal96/go-cms-kernel"
       "github.com/vernal96/go-cms-kernel/cache"
       "github.com/vernal96/go-cms-kernel/modules/admin"
       "github.com/vernal96/go-cms-kernel/modules/core"
   )

   var Editorial = kernel.Profile{
       Code: "editorial",
       Name: "Editorial",
       Modules: []kernel.ProfileModule{
           {Module: core.Module{}, Caches: []cache.Binding{
               {Alias: core.DurableCacheAlias, Code: "shared"},
               {Alias: core.HotCacheAlias, Code: "shared"},
           }},
           {Module: admin.Module{}},
       },
   }
   ```

   `Code` должен быть уникальным и стабильным: по нему сайты выбирают профиль.
2. Добавьте модули в нужном порядке. `core` обязателен и должен быть первым; `admin` также обязателен. Для остальных модулей явно задавайте зависимости и используйте их bindings (например, cache aliases).
3. Передайте профиль в `Profiles` при создании `settings.Config` в `backend/cmd/server/main.go`. Поле уже принимает список `[]kernel.Profile`; например, чтобы подключить новый `profile.Editorial`, добавьте его к существующим профилям:

   ```go
   Profiles: []kernel.Profile{
       profile.Starter,
       profile.Ministry,
       profile.Editorial,
   },
   ```

   `settings.Config.Definition()` передаёт этот список в `appkernel.Definition.Profiles`. При добавлении следующего профиля менять `settings.go` не нужно.
4. Убедитесь, что каждый модуль имеет требуемые database adapters, инфраструктурные bindings, migrations и seeds в composition root (`backend/cmd/server/main.go` и `backend/internal/infrastructure`). Профиль только перечисляет модули, он не создаёт эти зависимости автоматически.
5. Для нового сайта укажите `profile_code`, совпадающий с `Profile.Code`. Если сайт создаётся проектным seed, обновите seed отдельно.

Профили объявляйте как переменные `kernel.Profile`, без фабричных функций. Не помещайте в них lifecycle-код, создание приложения, подключение инфраструктуры или изменяемые runtime-объекты.

## Изменение профиля

Изменяйте декларацию профиля в `backend/internal/profile`, сохраняя уникальный код и обязательные модули. Добавляя модуль, проверьте порядок зависимостей и добавьте нужные адаптеры/миграции в композицию проекта. Изменение набора модулей может менять доступные API и поведение сайтов с этим профилем.

`Profile.Code` — идентификатор, на который ссылаются сайты (`core.sites.profile_code`). Переименование кода — это изменение данных и конфигурации: согласованно обновите декларацию, seed-данные и ссылки сайтов. В pre-production базе проекта допустимо пересоздать данные; не оставляйте сайты со старым неизвестным кодом.

## Удаление профиля

Перед удалением проверьте, что ни один seed, окружение или сайт не ссылается на код профиля. Удалите профиль из `Profiles` в `backend/cmd/server/main.go`, саму декларацию и связанные только с ним проектные seed-данные. Не удаляйте модули/адаптеры из приложения, если они используются другим профилем.

Текущая конфигурация передаёт `starter` и `ministry`; dev seed сайта использует `starter`. Удаление одного профиля допустимо, если нет сайтов или seed-данных с его `profile_code`. Не удаляйте последний профиль, на который ссылается сайт: сначала добавьте замену и перенесите сайт на её код либо пересоздайте локальные данные.

## Runtime

Профиль — декларация, а не runtime экземпляр. Kernel проверяет и упорядочивает модули, строит blueprint, а затем создаёт runtime сайта с нужной областью видимости. Не кэшируйте изменяемое состояние в декларации профиля и не переносите сборку runtime в обработчик запроса.
