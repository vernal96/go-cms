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

INSERT INTO core.resource_field_values (
    resource_id, site_id, field_key, position, is_multi, value_kind,
    value_string, value_boolean
)
SELECT resources.id, resources.site_id, values.field_key, 0, false,
       values.value_kind, values.value_string, values.value_boolean
FROM core.resources AS resources
JOIN core.sites AS sites ON sites.id = resources.site_id
CROSS JOIN LATERAL (VALUES
    ('page_title', 'string', resources.title, NULL::boolean),
    ('show_title', 'boolean', NULL::text, true),
    ('layout', 'string', 'standard', NULL::boolean)
) AS values(field_key, value_kind, value_string, value_boolean)
WHERE sites.profile_code = 'dev'
  AND resources.type = 'page'
  AND resources.template = 'page'
ON CONFLICT (resource_id, field_key, position) DO UPDATE SET
    value_kind = EXCLUDED.value_kind,
    value_string = EXCLUDED.value_string,
    value_boolean = EXCLUDED.value_boolean;
