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

## Поля параметров профиля

`Profile.Params` задаёт поля настроек сайта для всех сайтов, использующих профиль. Это именно параметры сайта, сохраняемые в `core.sites.settings`; это не поля контентных ресурсов (они объявляются в шаблонах ресурсов).

Тип поля указывается через `field.Definition`. В `backend/go.mod` закреплена версия kernel `v0.2.0`; стандартные типы регистрирует модуль `core`:

| Код типа | Назначение | Опции |
| --- | --- | --- |
| `field.TypeString` (`string`) | Однострочный текст | `field.StringOptions`: `Multiple`, `MinItems`, `MaxItems` |
| `field.TypeTextarea` (`textarea`) | Многострочный текст | `field.StringOptions`: `Multiple`, `MinItems`, `MaxItems` |
| `field.TypeInteger` (`int`) | Целое число | `field.IntegerOptions`: `Step`, `Multiple`, `MinItems`, `MaxItems` |
| `field.TypeFloat` (`float`) | Дробное число | `field.FloatOptions`: `Step`, `Multiple`, `MinItems`, `MaxItems` |
| `field.TypeCheckbox` (`checkbox`) | Логический флаг | Нет |
| `field.TypeRadio` (`radio`) | Выбор одного значения из вариантов | `field.RadioOptions{Choices: []field.Choice{...}}` |
| `field.TypeSelect` (`select`) | Выбор из списка; может быть множественным | `field.SelectOptions`: `Choices`, `Multiple`, `MinItems`, `MaxItems` |
| `field.TypeEmail` (`email`) | Строка с проверкой формата email | `field.StringOptions`: `Multiple`, `MinItems`, `MaxItems` |
| `field.TypePhone` (`phone`) | Телефон | `field.PhoneOptions`: `Pattern`, `Multiple`, `MinItems`, `MaxItems` |
| `field.TypeFile` (`file`) | Ссылка на файл | `field.FileOptions`: `Storages`, `MIMETypes`, `Multiple`, `MinItems`, `MaxItems` |
| `field.TypeMedia` (`media`) | Медиа, в текущей реализации — изображение | `field.MediaOptions`: `Multiple`, `MinItems`, `MaxItems`, `SettingsCode` |
| `field.TypeJSON` (`json`) | Структурированный JSON-объект или массив | Нет |
| `field.TypeRepeater` (`repeater`) | Упорядоченный список групп вложенных полей | `field.RepeaterOptions{Fields: []field.Definition{...}}` |

`Multiple` включает список значений там, где тип его поддерживает; `MinItems` и `MaxItems` ограничивают его размер, причём `MaxItems: 0` означает отсутствие верхнего ограничения. `Choices` задаёт пары стабильных значений `Value` и отображаемых подписей `Label`. Для файла можно ограничить допустимые коды хранилищ и MIME-типы, например `image/*`. Для телефона `Pattern` задаёт дополнительный шаблон. `Step` задаёт шаг числового редактора.

У определения также есть общие свойства: `Key` — уникальный ключ параметра, `Label` — подпись, `Required` — указатель на bool для явного включения или выключения обязательности, `Rules` — дополнительные правила валидатора, `Public` — разрешение включить значение в публичные данные сайта, `Editor` — код редактора, а `VisibleWhen` — простое условие показа относительно другого поля. `Editor` меняет представление в админке, но не тип и правила хранения значения.

### Как добавить поля в профиль

Добавьте определения в `Params` конкретного `kernel.Profile`. Необходимо импортировать пакет полей kernel как `field`. Например, в существующем файле профиля:

```go
import "github.com/vernal96/go-cms-kernel/modules/core/field"

var required = true

var editorialParams = []field.Definition{
	{
		Key: "organization_name", Type: field.TypeString,
		Label: "Название организации", Required: &required,
	},
	{
		Key: "contact_email", Type: field.TypeEmail,
		Label: "Контактный email", Public: true,
	},
	{
		Key: "office_phone", Type: field.TypePhone,
		Label: "Телефон приёмной",
		Options: field.PhoneOptions{Pattern: `^\+7`},
	},
}
```

В декларации профиля укажите `Params: editorialParams` рядом с `Code`, `Name` и `Modules`.

При добавлении определений:

1. Выбирайте стабильные уникальные `Key` в пределах параметров профиля: ключ используется для сохранения значения в настройках сайта.
2. Указывайте `Type` из списка выше и опции соответствующего типа. Не задавайте `Options`, если тип их не принимает.
3. Если параметр должен попадать в публичную проекцию сайта, явно установите `Public: true`; без этого значения остаются непубличными.
4. Добавьте профиль в `Profiles` конфигурации приложения по инструкции выше. Отдельная миграция БД для добавления определения не нужна: значения профиля хранятся в настройках сайта.
5. Если поле удаляется или переименовывается, согласуйте это с чтением его ключа в коде и с используемыми настройками сайтов: смена `Key` означает новый параметр.

`EditorTabs`, если они заданы у профиля, группируют поля редактора; каждый ключ в группировке должен ссылаться на объявленный параметр. Для обычного добавления параметров достаточно `Params`.

## Изменение профиля

Изменяйте декларацию профиля в `backend/internal/profile`, сохраняя уникальный код и обязательные модули. Добавляя модуль, проверьте порядок зависимостей и добавьте нужные адаптеры/миграции в композицию проекта. Изменение набора модулей может менять доступные API и поведение сайтов с этим профилем.

`Profile.Code` — идентификатор, на который ссылаются сайты (`core.sites.profile_code`). Переименование кода — это изменение данных и конфигурации: согласованно обновите декларацию, seed-данные и ссылки сайтов. В pre-production базе проекта допустимо пересоздать данные; не оставляйте сайты со старым неизвестным кодом.

## Удаление профиля

Перед удалением проверьте, что ни один seed, окружение или сайт не ссылается на код профиля. Удалите профиль из `Profiles` в `backend/cmd/server/main.go`, саму декларацию и связанные только с ним проектные seed-данные. Не удаляйте модули/адаптеры из приложения, если они используются другим профилем.

Текущая конфигурация передаёт `starter` и `ministry`; dev seed сайта использует `starter`. Удаление одного профиля допустимо, если нет сайтов или seed-данных с его `profile_code`. Не удаляйте последний профиль, на который ссылается сайт: сначала добавьте замену и перенесите сайт на её код либо пересоздайте локальные данные.

## Runtime

Профиль — декларация, а не runtime экземпляр. Kernel проверяет и упорядочивает модули, строит blueprint, а затем создаёт runtime сайта с нужной областью видимости. Не кэшируйте изменяемое состояние в декларации профиля и не переносите сборку runtime в обработчик запроса.
