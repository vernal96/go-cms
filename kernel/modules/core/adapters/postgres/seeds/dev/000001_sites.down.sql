DELETE
FROM core.sites
WHERE profile_code = 'dev'
  AND domain IN ('localhost', 'example.com');
