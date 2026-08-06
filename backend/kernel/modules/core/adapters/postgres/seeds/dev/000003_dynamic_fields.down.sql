UPDATE core.sites
SET settings = settings
    - 'string_value'
    - 'integer_value'
    - 'float_value'
    - 'checkbox_value'
    - 'radio_value'
    - 'select_value'
    - 'multi_select_value'
    - 'textarea_value'
    - 'email_value'
    - 'phone_value'
WHERE profile_code = 'dev';

UPDATE core.resources AS resources
SET settings = resources.settings
    - 'page_title'
    - 'show_title'
    - 'layout'
FROM core.sites AS sites
WHERE resources.site_id = sites.id
  AND sites.profile_code = 'dev'
  AND resources.type = 'page'
	AND resources.template = 'page';
