DELETE FROM core.user_groups WHERE user_id IN (SELECT id FROM core.users WHERE login = 'admin');
DELETE FROM core.users WHERE login = 'admin';
DELETE FROM core.sites WHERE domain = 'localhost' AND profile_code = 'starter';
