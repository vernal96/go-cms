INSERT INTO core.sites (profile_code, domain, locale, settings)
VALUES
    ('dev', 'localhost', 'ru-RU', '{}'::jsonb),
    ('dev', 'example.com', 'ru-RU', '{}'::jsonb)
ON CONFLICT DO NOTHING;
