# Создание расширений

Этот пакет — компилируемый пример, не подключённый к приложению. Проверка:

```bash
go -C backend test ./examples/extensions
```

Рабочий профиль находится в `internal/profiles/dev/profile.go`. Для нового
профиля достаточно функции, возвращающей `kernel.Profile`: код, название,
модули, шаблоны и параметры. Подключите её результат в `Config.Application`.
Обязательны `core` и `admin`; `core` идёт первым, зависимости объявляются
перед потребителями. Возвращайте готовый список модулей без ручных вставок.

Для шаблона верните `template.Definition`, затем добавьте его в
`Profile.Templates`. Валидацию полей, раскладки и ссылок выполняет компилятор.
Представление виджета объявляется через `widget.NewView` и `Profile.WidgetViews`.

Минимальному модулю нужны только `Code` и `Build`; его runtime предоставляет
`ModuleCode`. Дополнительные возможности — отдельные необязательные интерфейсы:

| Возможность | Подключение |
|---|---|
| Типы полей, ресурсов, права | `kernel.RegistryProvider` |
| Виджеты | `widget.Provider` |
| HTTP | `httptransport.Provider` / `SiteManagementProvider` |
| Меню админки | `adminui.NavigationProvider` |
| Фоновые задачи | `background.Provider` |
| Миграции / сиды | Провайдеры адаптера базы данных |
| Команды | `console.Provider` |

Обычный виджет описывайте через `widget.Functional`: метаданные и функцию
рендеринга. Каталог проверит параметры до вызова обработчика. Зависимости
сервисного виджета передавайте при построении runtime конкретного сайта.

Собственный тип поля реализует `field.Type`, включая
`Compile(ctx field.CompileContext, options any) (field.ValueType, error)`.
`ctx.Types` содержит текущий resolver; составные типы компилируют дочерние
определения через `ctx.Compile(fields)`, сохраняя контекст вложенности. Оберните тип в
`field.DescribedType`, чтобы объявить название, стандартный или специальный
редактор значения и список полей конфигурации `Presentation.Options`.
`OptionsEditor` задаёт специальный редактор всей конфигурации, если он нужен.
Для чтения настроек в `Compile` используйте `field.DecodeOptions[MyOptions]`:
он поддерживает Go-объявления и JSON-объекты, проверяет неизвестные ключи.
У полей структуры настроек должны быть явные JSON-теги. Тип, сохраняемый
в ресурсах/результатах, возвращает `field.StorageValueType`.

В модуле, дополняющем Forms, объявите зависимость `forms.ModuleCode`, получите
`forms.ElementRegistrar` или `forms.ActionRegistrar` через
`kernel.ModuleDependencyFrom` и зарегистрируйте расширения внутри `Build`.
Для простого элемента достаточно `forms.ElementDefinition` с полями настроек;
дополнительные доменные проверки задаются функцией `Validate`. Дубликаты и
регистрация после завершения сборки отклоняются. Каталоги принадлежат сайту.

Frontend-плагин подключается в `frontend-admin/src/admin-plugins.ts`.
`fieldEditors` содержит редакторы значений, `configEditors` — редакторы
объектов конфигурации. Оба используют семантические коды своего плагина,
например `example.notice`. Backend не знает имён Vue-компонентов.

Обычные настройки рисует `ConfigurationEditor` по метаданным. Он сохраняет
числа, флаги, массивы и объекты. Специальный редактор принимает объект через
`v-model`, `fields`, `siteId`, `accessToken`, `context`; может выставить метод
`validate()`, бросающий ошибку при некорректных настройках. Не мутируйте props:
отправляйте новый объект через `update:modelValue`. Авторитетная валидация и
проверка прав остаются на backend.

Периодическая задача может вызвать
`background.RunPeriodic(ctx, interval, operation)`: первый запуск немедленный,
отмена контекста завершает работу, первая ошибка операции останавливает задачу.
Политику повторной доставки бизнес-заданий задавайте через Jobs, отдельно от
этого исполнителя. Доменные очистки и обработку временных файлов сохраняйте
в модуле.

Физические диски и кеши регистрируются в приложении. Профиль связывает их
с логическими alias модулей; новый диск не требует изменения каждого сервиса.
Описание дисков: `internal/filesystems/README.md`.

## Конструктор (`repeater`)

`field.TypeRepeater` принимает `field.RepeaterOptions{Fields: []field.Definition{...},
MinItems: 0, MaxItems: 10}`. Внутри используются обычные поля, включая типы
модулей и их редакторы. `MaxItems: 0` означает отсутствие верхней границы.
Полный пример — `internal/profiles/dev/templates/landing.go`, вкладка «Слайды».

Каждая строка проходит обычную `field.Schema`; ошибки содержат путь
`slides[0].title`. Отсутствующее необязательное значение нормализуется в `[]`.
Весь список сохраняется одним `StorageJSON`, без вложенных EAV-полей.
Вложенные конструкторы, включая обёртки над ними, отклоняются при компиляции.

Типы со ссылками реализуют `field.ReferenceCollector`. `Reference.Path` —
массив ключей/индексов относительно значения, `Target` отличает файл от медиа.
Конструктор добавляет индекс строки, а схема — ключ внешнего поля. Для JSON
эти пути передаются вместе со `StoredValue.References`; файловые ограничения,
проверка медиа, учёт владения и очистка ссылок работают через общие сервисы.
Обычные скалярные ссылки сохраняют существующий `ReferenceValueType` контракт.

`RepeaterField.vue` рисует вложенные поля через `DynamicField`. Пользовательские
редакторы получают `field`, `siteId`, `accessToken`, `resourceTemplates` и `v-model`.
В интерфейсе доступны добавление, удаление и перемещение вверх/вниз.

## Множественные стандартные поля

Строка, многострочный текст, email, телефон, целое/дробное число, файл, медиа и
селект поддерживают `multiple`, `min_items`, `max_items` в `Options`:

```go
field.Definition{
    Key: "gallery", Type: field.TypeMedia, Label: "Галерея",
    Options: field.MediaOptions{Multiple: true, MaxItems: 10},
}
field.Definition{
    Key: "scores", Type: field.TypeInteger, Label: "Оценки",
    Options: field.IntegerOptions{Multiple: true, MinItems: 1, MaxItems: 5},
    Rules: []string{"min=0", "max=100"},
}
```

Для `string`, `textarea`, `email` используется `field.StringOptions`; остальные
типы расширяют собственные `IntegerOptions`, `FloatOptions`, `PhoneOptions`,
`FileOptions`, `MediaOptions`, `SelectOptions`. Ограничения количества допустимы
только при `Multiple: true`, нулевой максимум означает отсутствие ограничения.
`Required` требует непустой список, положительный `MinItems` также запрещает пустое
значение. Правила проверяют каждый элемент; ошибки включают индекс (`scores[1]`).
Добавленная пустая строка списка должна быть заполнена или удалена. Ноль — число,
а не пустое значение. Повторы сохраняются, кроме уникальных вариантов селекта.

HTTP передаёт массив даже для одного элемента: `{"scores":[0]}`. Необязательный
пустой список сохраняет существующую семантику отсутствующего поля. Массивы
ресурсов хранятся типизированными строками с позицией; по ним можно фильтровать,
но нельзя сортировать ресурсы. Множественные поля доступны и внутри конструктора.
В Forms признак множественности сохраняется вместе с результатом: изменение
настройки поля не меняет форму старого значения при чтении или передаче в Mail.
Multipart принимает JSON-массив либо повторяющиеся части с кодом поля; единственная
часть множественного поля также становится массивом.

В dev-шаблоне `page` вкладка «Множественные поля» содержит галерею, теги и числа.
Редакторы списка сохраняют порядок и поддерживают «Добавить», «Удалить», «Вверх»,
«Вниз»; селект использует обычный мультивыбор. Каждая строка сохраняет собственный
редактор, включая HTML и редакторы расширений. Повтор одного Media разрешён внутри
ресурса-владельца; использование другим владельцем по-прежнему отклоняется.

## Настройки медиа

Именованные наборы объявляются в `core.Config.MediaSettings` и используют
обычные типы полей и правила:

```go
optional := false
config := core.Config{
    MediaSettings: []media.SettingsDefinition{
        {Code: "image", Fields: []field.Definition{
            {Key: "alt", Type: field.TypeString, Label: "Альтернативный текст", Required: &optional},
            {Key: "title", Type: field.TypeString, Label: "Заголовок", Required: &optional},
        }},
    },
}
imageField := field.Definition{
    Key: "image", Type: field.TypeMedia, Label: "Изображение",
    Options: field.MediaOptions{SettingsCode: "image"},
}
```

Передайте `config` в декларацию Core профиля, а `imageField` — в шаблон или
другую схему полей этого профиля. `Multiple: true` и использование внутри
`RepeaterOptions.Fields` работают с тем же набором. Неизвестные коды наборов,
дубли и некорректные схемы отклоняются при построении профиля/runtime.

Кнопка «Настройки» сохраняет значения сразу, независимо от сохранения
основной формы. Значение самого поля остаётся ID Media, а настройки доступны
через `Media.Params["settings"]`. `title` внутри настроек не связан с
`Media.Title`; служебные преобразования в `Params["image"]` сохраняются.
Новая Media для того же файла получает собственные пустые настройки.
Ревизии ресурса не содержат снимок этих настроек.

GET `/api/sites/{siteID}/media/{mediaID}/settings?code=image` возвращает
`code`, `fields`, `values`, `expected_updated_at`. PUT на тот же путь принимает
`code`, `values`, `expected_updated_at`. Схема берётся из runtime сайта;
конфликт версии возвращает 409. Проверяются доступ к сайту и права Media.
Поддерживаются ссылки `file` с проверкой доступности, диска и MIME;
вложенные `media` и неизвестные схемы ссылок запрещены, поскольку требуют
отдельного учёта владения и удаления.

## Widget resource bindings

`BoundPage()` demonstrates template widgets receiving whole values from the
current resource. Declare `ParamBindings: widget.ParamBindings{"name":
widget.ResourceField("visitor")}` for a template field, or use
`widget.ResourceProperty("title")` for a standard property. The same parameter
must not appear in `Params`.

The admin widget editor exposes the same contract through `param_bindings`.
Source and target must have exactly the same field type and multiplicity.
References are checked when configured; values and target constraints are checked
when rendered. An invalid current value fails only that widget. Lists, JSON and
repeaters are passed whole. References never copy a value into persisted params,
and strings containing `{{ ... }}` remain literal text.
