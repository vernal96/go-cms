# Core: API и обработчики

Маршруты Core приведены относительно Core HTTP handler. Публичные маршруты выбирают сайт обычной маршрутизацией запроса; management API монтируется приложением под административным префиксом и требует авторизацию/права. Path ID должны быть положительными числами.

## Публичные маршруты

| Метод и путь | Назначение |
| --- | --- |
| `GET /site` | Публичные данные текущего сайта |
| `GET /menu` | Меню текущего сайта |
| `GET /{resource-path}` | Публичный ресурс; способ ответа выбирает его тип |

Публичные GET-маршруты не принимают query-параметры для выбора сайта. Переданный клиентом `site_id` не переключает runtime.

## Management API

### Сайты и профили

| Маршруты | Параметры |
| --- | --- |
| `GET /sites` | Query: `search`, `page`, `per_page`; pagination defaults: `1` и `10`, `per_page` максимум `100`. |
| `GET /sites/options` | Те же параметры плюс необязательный `exclude_id` — положительный ID сайта, исключаемого из вариантов. |
| `GET /site-profiles` | Параметров нет. |
| `POST /sites` | JSON: `profile_code`, `domain`, `locale`, `settings` (объект, может быть пустым), `is_public` (boolean). |
| `GET /sites/{siteID}`, `DELETE /sites/{siteID}` | `{siteID}` — положительный ID; body/query нет. |
| `PATCH /sites/{siteID}` | JSON: `profile_code`, `domain`, `locale`, `settings` (обязательно объект), `is_public` (обязательный boolean). Обновление задаёт состояние сайта целиком. |

Пример запроса списка: `GET /sites?search=example&page=1&per_page=20`.

### Ресурсы и дерево

| Маршруты | Параметры |
| --- | --- |
| `GET /sites/{siteID}/resources` | Необязательный query `parent_id`: положительный ID родителя; если не задан, возвращаются корневые ресурсы сайта. |
| `GET .../resources/metadata`, `GET .../resources/options` | Только `{siteID}` в пути; query/body нет. |
| `GET .../resources/lookup` | Query `search`, `page` (default `1`), `per_page` (default `10`, максимум `100`). |
| `POST .../resources` | JSON: `parent_id`, `type`, `template_code`, `content_type`, `content`, `target_resource_id`, `title`, `menu_title`, `slug`, `external_url`, `fields`, `type_settings`. `fields` и `type_settings` обязательны как объекты, в том числе пустые; остальные применяются в соответствии с типом ресурса. |
| `GET/PATCH/DELETE .../resources/{resourceID}` | ID сайта и ресурса в пути. PATCH принимает `expected_version`, `parent_id`, `type`, `template_code`, `image_media_id`, `title`, `menu_title`, `slug`, `annotation`, `content`, `content_type`, `target_resource_id`, `external_url`, `is_public`, `is_searchable`, `in_menu`, `in_sitemap`, `sort`, `published_at`, `unpublished_at`, `fields`, `type_settings`. `expected_version` положителен; `fields`, `type_settings`, все четыре boolean-флага и `sort` обязательны. Даты передаются в JSON формате RFC3339. |
| `POST .../resources/{resourceID}/move` | JSON: `parent_id` (null для корня), обязательные `position` (0 или больше), `expected_version` (положительный). |
| `POST .../resources/{resourceID}/transfer` | JSON: обязательные `target_site_id` и `expected_version`, оба положительные. |
| `DELETE .../resources/{resourceID}` / `/permanent` | Мягкое / окончательное удаление; тело не требуется. |
| `POST .../resources/{resourceID}/restore` | JSON `with_descendants` (boolean): восстановить ли вместе потомков. |

Пример создания страницы:

```http
POST /sites/12/resources
Content-Type: application/json

{"type":"page","title":"О проекте","slug":"about","content_type":"html","content":"<p>Описание</p>","fields":{},"type_settings":{}}
```

Обновление ресурса всегда использует `expected_version`, возвращённую предыдущим чтением/изменением, чтобы обнаруживать конкурентные правки.

### Виджеты, версии и расширения ресурса

| Маршруты | Параметры |
| --- | --- |
| `POST .../resources/{resourceID}/widgets` | JSON: `code`, `area`, `view`, `columns`, `margin_top`, `margin_bottom`, `enabled`, `params` (обязательный объект), `param_bindings`, `expected_version` (обязательный положительный). |
| `PATCH .../widgets/{widgetID}` | Те же параметры представления, `params` и `enabled` обязательны; `expected_version` обязателен. `{widgetID}` — положительный ID привязки. |
| `DELETE .../widgets/{widgetID}` | JSON `expected_version` — положительная версия ресурса. |
| `PUT .../widgets/order` | JSON: `expected_version` и `items` (обязательный массив нового порядка). |
| `GET .../revisions` | `page`, `per_page`; defaults `1` и `10`, максимум `100`. |
| `GET .../revisions/{version}` | `{version}` — положительный номер ревизии. |
| `POST .../revisions/{version}/restore` | JSON `expected_version` — положительная текущая версия ресурса. |
| `DELETE .../revisions` | Purge ревизий этого ресурса; параметров запроса/тела нет. |
| `GET/PATCH .../extensions/{extensionCode}` | Код расширения — часть пути. PATCH передаёт JSON, форму которого определяет конкретное расширение. |
| `POST .../extensions/{extensionCode}/preview` | JSON с параметрами расширения для предпросмотра; схема также определяется расширением. |

### Элементы библиотеки

| Маршруты | Параметры |
| --- | --- |
| `GET .../resources/{libraryID}/items` | Query: `cursor` (opaque cursor следующей страницы), `limit` (default `25`), `search`, `filters`, `sort`. `filters` и `sort` — JSON-массивы, URL-encoded при передаче в query. |
| `POST .../resources/{libraryID}/items` | JSON: `image_media_id`, `template_code`, `title`, `slug`, `annotation`, `content`, `is_public`, `is_searchable`, `published_at`, `unpublished_at`, `fields` (обязательный объект). |
| `GET/PATCH/DELETE .../library-items/{itemID}` | `{itemID}` — ID элемента. PATCH использует поля создания плюс обязательные `expected_version`, `fields`, `is_public` и `is_searchable`. |
| `POST .../library-items/{itemID}/move` | JSON: `library_id` и `expected_version` — положительные ID/версия. |
| `POST .../library-items/{itemID}/restore`, `DELETE .../{itemID}/permanent` | Без тела; path ID определяет элемент. Обычный DELETE мягко удаляет. |

Допустимые значения `filters[].field`: встроенные имена `id`, `title`, `slug`, `template`, `is_public`, `is_searchable`, `published_at`, `created_at`, `updated_at`, либо валидный `resource.field.<key>`. `filters[]` принимает `field`, `operator`, `value`; `operator`: `eq`, `neq`, `in`, `not_in`, `gt`, `gte`, `lt`, `lte`. `sort[]` принимает `field` и `direction` (`asc` или `desc`).

Пример query (значения JSON в реальном URL нужно URL-encode):

```text
GET .../items?limit=10&search=report&filters=[{"field":"is_public","operator":"eq","value":true}]&sort=[{"field":"title","direction":"asc"}]
```

### Файлы и папки

| Маршрут | Параметры |
| --- | --- |
| `GET /files/disks` | Без параметров; возвращает доступные диски. |
| `GET /files/items` | Query `disk` обязателен; необязательный `folder_id` выбирает папку. |
| `GET /files/folders/resolve` | Query `disk` и `path` обязательны. |
| `POST /files/folders/ensure` | JSON `disk`, `path`; создать путь папок при необходимости. |
| `POST /files/folders` | JSON `disk`, `parent_id` (nullable), `name`. |
| `PATCH /files/folders/{folderID}`, `PATCH /files/{fileID}` | JSON `name`; path ID выбирает переименовываемую сущность. |
| `POST /files/uploads` | Multipart: обязательный `file`, `disk`, необязательный `folder_id`; размер ограничен конфигурацией. |
| `GET /files/{fileID}`, `/preview`, `/download` | Положительный `fileID`; query/body не требуются. |
| `POST /files/move` | JSON `disk`, `folder_id` (nullable), `items`: массив `{kind, id}`, где `kind` — `file` или `folder`. |
| `POST /files/delete-impact` | JSON `items`: массив `{kind, id}`; возвращает предварительный анализ последствий. |
| `POST /files/delete` | JSON `items`, `policy`, `impact_token`. Пустая policy означает безопасное удаление; подтверждённый media cascade требует policy `confirmed_media_cascade` и токен из анализа. |

### Изображения, media и site menu

| Маршрут | Параметры |
| --- | --- |
| `GET /files/{fileID}/thumbnail` | Query: `width`, `height`, `fit`, `position`, `profile`; каждый параметр можно передать один раз. Размеры — из набора `64`, `128`, `256`, `512`, `1024` и не выше лимита конфигурации; по умолчанию `128×128`. `fit`: `contain`, `cover`, `stretch`; `position`: `center`, `top`, `bottom`, `left`, `right`, `top-left`, `top-right`, `bottom-left`, `bottom-right`. `profile` выбирает профиль обработки. |
| `POST /media` | JSON `file_id` — положительный ID файла для создания media. |
| `GET /media/{mediaID}/image` | `{mediaID}` — положительный ID media; query/body нет. |
| `POST /media/{mediaID}/image` | JSON `expected_updated_at` (timestamp из image state) и `transform`. Transform поддерживает `crop` (`x`, `y`, `width`, `height`), `rotate` (кратно 90°, от −360 до 360), `scale_x`, `scale_y`, `width`, `height`, `fit`, `position`, `quality` (1–100). Непереданные scale по умолчанию 1, fit — `contain`, position — `center`, quality — 85. Версия защищает от перезаписи параллельных изменений. |
| `POST /media/{mediaID}/image/restore` | JSON `expected_updated_at` из текущего image state. |
| `GET/PUT /sites/{siteID}/media/{mediaID}/settings` | При GET необязательный query `code` выбирает схему настроек. PUT JSON: `code`, `values` (объект настроек), `expected_updated_at`. |
| `GET /sites/{siteID}/menu` | Только положительный `siteID`, без query/body. |

## Примеры

Получить корневые ресурсы сайта и запросить вторую страницу списка сайтов:

```http
GET /sites/12/resources
GET /sites?search=example&page=2&per_page=10
```

Неизвестные JSON-поля отклоняются. Управляющие операции, включая окончательное удаление и очистку истории, требуют соответствующих прав; для mutation API используйте актуальные версии сущностей, когда это поле предусмотрено контрактом.
