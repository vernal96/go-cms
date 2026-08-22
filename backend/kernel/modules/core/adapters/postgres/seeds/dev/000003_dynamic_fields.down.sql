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

DELETE FROM core.resource_field_values AS value
USING core.resources AS resources, core.sites AS sites
WHERE value.resource_id = resources.id
  AND resources.site_id = sites.id
  AND sites.profile_code = 'dev'
  AND resources.type = 'page'
  AND resources.template = 'page'
  AND value.field_key IN ('page_title', 'show_title', 'layout');
