INSERT INTO core.users (
    login,
    email,
    password_hash,
    name
)
VALUES (
    'admin',
    'admin@example.test',
    '$argon2id$v=19$m=19456,t=2,p=1$AQIDBAUGBwgJCgsMDQ4PEA$LfEY87S5TQuDcvS9YYvSRIgL7Bsrha+TSbFU7pwDSZI',
    'Администратор'
)
ON CONFLICT (login) DO UPDATE
SET email = EXCLUDED.email,
    password_hash = EXCLUDED.password_hash,
    name = EXCLUDED.name,
    deleted_at = NULL,
    deleted_by = NULL,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = NULL;

INSERT INTO core.user_groups (user_id, group_id)
SELECT core.users.id, core.groups.id
FROM core.users
JOIN core.groups ON core.groups.code = 'admin'
WHERE core.users.login = 'admin'
ON CONFLICT (user_id, group_id) DO UPDATE
SET updated_at = CURRENT_TIMESTAMP,
    updated_by = NULL;
