# Развертывание проекта с нуля

Инструкция рассчитана на чистую рабочую копию и новую базу. Для Docker-сценария нужны Git, Python 3, GNU Make, Docker и Docker Compose v2 с поддержкой `--wait`; Go и Node.js на хосте не нужны.

## Backend

```sh
git clone https://github.com/vernal96/go-cms.git
cd go-cms
make up
```

`make up` создаёт `.env` из `.env.example`, генерирует уникальные секреты, собирает backend и запускает PostgreSQL, Redis и Kafka. Команда ждёт готовности сервисов. На новой базе применяются миграции. Dev seed по умолчанию выключен.

Проверьте `http://localhost:8080/healthz` — ожидается HTTP 200. Backend предоставляет API; главная страница сайта не создаётся, поэтому ответ 404 на `/` сам по себе не означает ошибку.

Создайте первого администратора, передав пароль через окружение:

```sh
CMS_ADMIN_PASSWORD='замените-на-сильный-пароль' docker compose --env-file .env run --rm -e CMS_ADMIN_PASSWORD server bootstrap-admin
```

При необходимости задайте `CMS_ADMIN_LOGIN` и `CMS_ADMIN_EMAIL` тем же способом. Команда не перезаписывает существующую учётную запись.

Для изолированного локального демо можно включить dev seed:

```sh
CMS_DEV_SEED=true make up
```

Seed создаёт пользователя `admin` с паролем, зафиксированным в SQL seed. Используйте его только для локального демо; пароль можно изменить в админке в разделе пользователей. Обычный запуск seed не включает.

## Админка

В соседней директории клонируйте отдельное приложение:

```sh
cd ..
git clone https://github.com/vernal96/go-cms-admin.git
cd go-cms-admin
ADMIN_API_TARGET=http://host.docker.internal:8080 docker compose up -d --build --wait
```

Откройте `http://localhost:5173` и войдите созданной учётной записью администратора.

Без Docker админке нужны Node.js >=24 и npm:

```sh
cp .env.example .env
npm ci
npm run dev
```

Для локального режима укажите в `.env` `ADMIN_API_TARGET=http://localhost:8080`.

## Остановка и данные

```sh
make down   # остановить сервисы, сохранив БД и файлы
make up     # запустить снова
```

Не используйте `docker compose down -v`, если нужно сохранить данные: эта команда удаляет volumes текущего Compose project. Параметры портов и независимых копий проекта описаны в разделе «Порты и независимые копии» файла [README](../README.md#порты-и-независимые-копии).

