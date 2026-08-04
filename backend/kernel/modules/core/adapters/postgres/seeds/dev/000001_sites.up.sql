INSERT INTO core.sites (
    profile_code,
    domain,
    locale,
    settings,
    is_public
)
VALUES
    ('dev', 'localhost', 'ru-RU', '{}'::jsonb, true),
    ('dev', 'example.com', 'ru-RU', '{}'::jsonb, true)
ON CONFLICT DO NOTHING;
