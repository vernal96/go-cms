# Admin: API и обработчики

Пути указаны относительно Admin handler; общий mount prefix задаёт приложение. Все маршруты требуют аутентификацию, management операции также проверяют права.

| Путь | Назначение |
| --- | --- |
| `GET /session` | Информация о текущей сессии |
| `GET/PATCH /profile` | Чтение и обновление своего профиля |
| `PUT /profile/password`, `/profile/preferences` | Пароль и пользовательские предпочтения |
| `/profile/avatar` | Выбор, загрузка, удаление и preview аватара |
| `GET/POST /users`, `GET/PATCH /users/{userID}` | Список и CRUD пользователей |
| `/users/{userID}/password`, `/groups`, `/block`, `/unblock` | Пароль, членство в группах и блокировка |
| `GET/POST /groups`, `GET/PATCH/DELETE /groups/{groupID}` | Управление группами |
| `GET /groups/options`, `GET /permission-catalog` | Варианты групп и каталог разрешений |
| `GET /navigation`, `GET /dashboard` | Динамическая навигация и dashboard |

## Параметры запросов

### Путь и query

- `{userID}` и `{groupID}` — положительные числовые ID.
- Для `GET /users` поддерживаются `search`, `status`, `page`, `per_page`. `status`: `all`, `active` или `blocked`; по умолчанию `all`. `page` по умолчанию `1`, `per_page` — `10`, допустимый диапазон `per_page`: `1–100`.
- Для `GET /groups` и `GET /groups/options` поддерживаются `search`, `page`, `per_page` с теми же значениями пагинации.
- `GET /navigation` принимает необязательный `site_id` — положительный ID сайта, для которого запрашивается доступная site-навигация. При неправильном значении вернётся `400`.
- Остальные GET-маршруты таблицы не принимают query-параметров.

### JSON-тела

- `PATCH /profile`: `name`, `last_name`, `middle_name`, `phone`.
- `PUT /profile/password`: `current_password`, `new_password`.
- `PUT /profile/preferences`: `color_scheme` (`light`, `dark`, `system`) и `accent_color` (`blue`, `violet`, `indigo`, `emerald`, `amber`, `rose`).
- `PUT /profile/avatar`: `file_id` — положительный ID файла. `POST /profile/avatar/upload` принимает multipart-поле `file`; размер ограничен конфигурацией приложения. `DELETE /profile/avatar` и `GET /profile/avatar/preview` тела не имеют.
- `POST /users`: `login`, `email`, `password`, `name`, необязательные `last_name`, `middle_name`, `phone`, `group_ids` (массив ID групп). `PATCH /users/{userID}` принимает профильные поля без `password` и `group_ids`; пустые/невалидные значения проверяются сервисом.
- `PUT /users/{userID}/password`: `password`. `PUT /users/{userID}/groups`: `group_ids` — полный новый список ID групп (заменяет членство). Block/unblock тела не имеют.
- `POST /groups`: `code`, `name`, `permission_codes` (массив кодов разрешений), `site_access` (массив объектов `{site_id, can_view, can_edit, can_delete}`). `PATCH /groups/{groupID}` принимает `name`, `permission_codes`, `site_access`; отсутствие массива отличается от пустого массива: пустой массив очищает соответствующий список.

Пример создания пользователя:

```http
POST /users
Content-Type: application/json

{"login":"editor","email":"editor@example.test","password":"<secret>","name":"Редактор","group_ids":[4]}
```

Список пользователей из предыдущего примера принимает `status=active` и пагинацию, например `GET /users?search=editor&status=active&page=1&per_page=20`.

Списки возвращают пагинацию и набор разрешённых действий. Способ передачи токена и базовый URL определяются приложением. JSON-декодер отклоняет неизвестные поля.
