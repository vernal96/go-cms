INSERT INTO core.sites (profile_code,domain,locale,settings,is_public) VALUES
 ('extended','extended.example.test','ru-RU','{"message":"Initial"}',true),
 ('plain','plain.example.test','ru-RU','{}',true)
ON CONFLICT DO NOTHING;
INSERT INTO core.users(login,email,password_hash,name) VALUES
 ('extension-test','extension@example.test','disabled-fixture-password','Extension test') ON CONFLICT DO NOTHING;
INSERT INTO core.user_groups(user_id,group_id)
 SELECT u.id,g.id FROM core.users u CROSS JOIN core.groups g
 WHERE u.login='extension-test' AND g.code='admin' ON CONFLICT DO NOTHING;
