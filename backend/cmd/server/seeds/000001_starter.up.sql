INSERT INTO core.sites (profile_code, domain, locale, settings, is_public)
VALUES ('starter', 'localhost', 'ru-RU', '{}'::jsonb, true)
ON CONFLICT DO NOTHING;

INSERT INTO core.users (login, email, password_hash, name)
VALUES ('admin', 'admin@example.test', '$argon2id$v=19$m=19456,t=2,p=1$AQIDBAUGBwgJCgsMDQ4PEA$LfEY87S5TQuDcvS9YYvSRIgL7Bsrha+TSbFU7pwDSZI', 'Administrator')
ON CONFLICT (login) DO UPDATE SET email = EXCLUDED.email, password_hash = EXCLUDED.password_hash, name = EXCLUDED.name, blocked_at = NULL, blocked_by = NULL, updated_at = CURRENT_TIMESTAMP, updated_by = NULL;

INSERT INTO core.user_groups (user_id, group_id)
SELECT users.id, groups.id FROM core.users users CROSS JOIN core.groups groups
WHERE users.login = 'admin' AND groups.code = 'admin'
ON CONFLICT (user_id, group_id) DO NOTHING;
