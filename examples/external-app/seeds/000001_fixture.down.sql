DELETE FROM core.sites WHERE domain IN ('extended.example.test','plain.example.test');
DELETE FROM core.users WHERE login='extension-test';
