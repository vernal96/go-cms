# Go CMS agent department

Менеджер — основной чат Codex при запуске из go-cms. Его инструкции находятся в AGENTS.md и .codex/team/manager.md. Файл .codex/agents/project-manager.toml нужен для явного назначения этой роли, но обычно создавать дочернего PM не требуется.

Подчинённые роли:
- backend_junior — документированные профили/поля/виджеты/проектные модули.
- backend_middle — модули ядра, миграции, валидаторы, API, ограниченные изменения контрактов.
- backend_senior — сложные внутренние механизмы по эскалации.
- admin_frontend — go-cms-admin Vue/TS UI.
- documenter — общая документация go-cms-kernel/docs и профильные README/SDK.
- cms_admin — фактические сценарии пользователя в браузере, QA_PASS/QA_FAIL.

Три соседние рабочие копии рекомендованы в общей директории:
- /home/denis/projects/go-cms
- /home/denis/projects/go-cms-kernel
- /home/denis/projects/go-cms-admin

Пути — примеры, перенастрой под фактические клонирования. Запускай Codex из go-cms и давай дополнительные рабочие директории через --add-dir. Если работаешь в проекте-потребителе mosreg, добавь и его директорию. Инструкция агента о пути не является разрешением sandbox на запись.

Пример для Linux (поменяй пути под машину):
codex -C /home/denis/projects/go-cms --add-dir /home/denis/projects/go-cms-kernel --add-dir /home/denis/projects/go-cms-admin --add-dir /home/denis/projects/mosreg

Нужны запущенные backend и admin и инструмент браузера (Playwright либо Chrome DevTools MCP), настроенный отдельно. Без реального браузера cms_admin должен вернуть BLOCKED, а не QA_PASS.

При запуске Codex можно дать менеджеру задание:
«Подними проект mosreg, создай профиль ministry, шаблоны и виджеты по ТЗ. Делегируй разработку, устрани DOC_GAP/FEATURE_GAP, пройди приёмку через cms_admin и выпусти релизы только после QA_PASS. Уточняй продуктовые неоднозначности».

Учетная запись Codex должна иметь доступ к указанным моделям; при недоступной модели замени model в TOML на доступную. Конфигурации могут требовать доверия к директории в настройках Codex.
