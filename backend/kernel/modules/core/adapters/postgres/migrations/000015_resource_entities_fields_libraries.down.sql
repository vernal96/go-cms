DELETE FROM core.resource_entities
WHERE storage_kind = 'library_item';

DROP TABLE core.library_items;
DROP TABLE core.library_item_routes;

ALTER TABLE core.resources
    ADD COLUMN settings JSONB NOT NULL DEFAULT '{}'::jsonb
        CHECK (jsonb_typeof(settings) = 'object');

UPDATE core.resources AS resource
SET settings = values.fields
FROM (
    SELECT resource_id, jsonb_object_agg(field_key, value) AS fields
    FROM (
        SELECT resource_id, field_key,
            CASE
                WHEN NOT bool_or(is_multi) AND count(*) = 1 AND min(position) = 0 THEN (array_agg(value ORDER BY position))[1]
                ELSE jsonb_agg(value ORDER BY position)
            END AS value
        FROM (
            SELECT resource_id, field_key, position, is_multi,
                CASE value_kind
                    WHEN 'string' THEN to_jsonb(value_string)
                    WHEN 'integer' THEN to_jsonb(value_integer)
                    WHEN 'float' THEN to_jsonb(value_float)
                    WHEN 'boolean' THEN to_jsonb(value_boolean)
                    WHEN 'timestamp' THEN to_jsonb(value_timestamp)
                    WHEN 'reference' THEN to_jsonb(value_reference)
                    ELSE value_json
                END AS value
            FROM core.resource_field_values
        ) AS rows
        GROUP BY resource_id, field_key
    ) AS fields
    GROUP BY resource_id
) AS values
WHERE resource.id = values.resource_id;

DROP TABLE core.resource_field_values;

ALTER TABLE core.resource_widgets
    DROP CONSTRAINT fk_resource_widgets_resource,
    ADD CONSTRAINT fk_resource_widgets_resource
        FOREIGN KEY (resource_id) REFERENCES core.resources (id) ON DELETE CASCADE;

ALTER TABLE core.resources
    DROP CONSTRAINT fk_resources_target_site,
    ADD CONSTRAINT fk_resources_target_site
        FOREIGN KEY (target_resource_id, site_id)
            REFERENCES core.resources (id, site_id)
            ON DELETE NO ACTION,
    DROP CONSTRAINT fk_resources_entity_site,
    DROP COLUMN type_settings,
    ALTER COLUMN id SET DEFAULT nextval('core.resources_id_seq');

SELECT setval(
    'core.resources_id_seq',
    greatest(coalesce((SELECT max(id) FROM core.resources), 1), 1),
    EXISTS (SELECT 1 FROM core.resources)
);

DROP TABLE core.resource_entities;
