# Forms: API и обработчики

## Публичное API

Маршруты разрешаются в контексте сайта, где Forms включён. `{code}` — код формы, используемый публичной страницей; форма должна быть включена.

| Метод и путь | Назначение |
| --- | --- |
| `GET /forms/{code}` | Схема включённой формы |
| `POST /forms/{code}/submit` | Отправка значений; multipart поддерживает uploads |
| `GET /forms/{code}/results` | Публичные результаты и пагинация разрешённых колонок |

Пример JSON отправки:

```http
POST /forms/feedback/submit
Content-Type: application/json

{"values":{"name":"Анна","message":"Здравствуйте"}}
```

Для загрузки используйте `multipart/form-data`: обычные поля передаются как значения с ключами-кодами полей, файлы — как части запроса с именем поля формы. Валидируются типы значений, обязательность, правила, согласие, CAPTCHA и ограничения публичной конфигурации. CAPTCHAs и uploads не возвращаются как обычные публичные колонки результатов.

`GET /forms/{code}/results` принимает query-параметры `page` (по умолчанию `1`) и `per_page` (по умолчанию `20`, не более `100`). Оба значения должны быть положительными целыми числами; повторяющийся одноимённый параметр отклоняется.

## Site-management API

Contribution использует segment `forms`, а handler содержит внутренний путь `/forms`. Поэтому список форм доступен по полному пути `GET /api/sites/{siteID}/forms/forms` (с учётом стандартного `/api` префикса сервера).

Другие операции handler: `/forms/{formID}/editor`, `/forms/{formID}/fields`, `/forms/{formID}/elements`, `/forms/{formID}/containers`, `/forms/{formID}/layout`, `/forms/{formID}/statuses`, `/forms/{formID}/actions`, `/results` и `/results/{resultID}`. CRUD требует аутентификацию и site access.

### Path и query параметры

`{siteID}`, `{formID}`, `{fieldID}`, `{elementID}`, `{nodeID}`, `{statusID}`, `{actionID}` и `{resultID}` — положительные целые ID. `GET /forms` поддерживает `search` (после trim максимум 255 символов), `page` (по умолчанию `1`) и `per_page` (по умолчанию `20`, максимум `100`). `GET /results` принимает те же параметры плюс:

- `form_id`, `status_id` — необязательные положительные ID-фильтры.
- `date_from`, `date_to` — необязательные даты RFC3339, например `2026-09-01T00:00:00Z`.

### Тела запросов

- Создание/изменение формы: `code`, `name`, `description`, `enabled`.
- Создание/изменение поля: `code`, `type`, `label`, `required`, `rules`, `options`, `editor`, `visible_when`, `result_label`, `show_on_site`, `show_in_results`, `result_position`. При создании также указываются `parent_id` (ID layout-контейнера или `null` для корня) и `position` (позиция среди соседей).
- Создание/изменение элемента: `code`, `type`, `config`; при создании — также `parent_id` и `position` для размещения.
- Создание контейнера: `parent_id`, `container_type` (`group` или `slide`), `position`, `config`. Замена layout: `nodes` — полный массив новых layout nodes.
- Создание/изменение статуса: `code`, `name`, `color`, `position`, `is_default`.
- Создание/изменение action: `code`, `name`, `enabled`, `trigger`, `action_type`, `config`, `position`. `trigger.type` — `submitted` или `status_changed`; параметры trigger могут содержать `from_status` и `to_status`.
- `PATCH /forms/{formID}/enabled`: `enabled` — boolean. `PATCH /results/{resultID}/status`: `status_id` — положительный ID статуса.
- `PUT /forms/{formID}/layout`: `nodes` заменяет всю раскладку формы; узел задаёт `parent_id`, `kind` (`field`, `element`, `container`), соответствующий `field_id`/`element_id`/`container_type`, `position` и `config`. Delete-маршруты и чтение редактора тела не принимают.

Например, создание формы: `POST /api/sites/12/forms/forms` с телом `{"code":"feedback","name":"Обратная связь","description":"Напишите нам","enabled":false}`. При создании enabled обычно оставляют `false` до завершения настройки.
