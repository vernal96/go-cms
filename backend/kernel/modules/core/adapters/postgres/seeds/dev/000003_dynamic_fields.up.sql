UPDATE core.sites
SET settings = jsonb_build_object(
        'string_value', domain,
        'integer_value', 10,
        'float_value', 1.5,
        'checkbox_value', false,
        'radio_value', 'first',
        'select_value', 'alpha',
        'multi_select_value', jsonb_build_array('alpha', 'beta'),
        'textarea_value', 'Демонстрационные настройки сайта',
        'email_value', 'admin@example.test',
        'phone_value', '+79991234567'
    ) || settings
WHERE profile_code = 'dev';

UPDATE core.resources AS resources
SET settings = jsonb_build_object(
        'page_title', resources.title,
        'show_title', true,
        'layout', 'standard'
    ) || resources.settings
FROM core.sites AS sites
WHERE resources.site_id = sites.id
  AND sites.profile_code = 'dev'
  AND resources.type = 'page'
	AND resources.template = 'page';
