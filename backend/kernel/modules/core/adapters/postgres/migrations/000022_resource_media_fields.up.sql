CREATE TABLE core.resource_media_references (
    resource_id BIGINT NOT NULL,
    field_key TEXT NOT NULL,
    position INTEGER NOT NULL,
    value_path TEXT[] NOT NULL DEFAULT '{}',
    media_id BIGINT NOT NULL UNIQUE,
    PRIMARY KEY (resource_id, field_key, position, value_path),
    FOREIGN KEY (resource_id, field_key, position)
        REFERENCES core.resource_field_values(resource_id, field_key, position) ON DELETE CASCADE,
    FOREIGN KEY (media_id) REFERENCES core.media(id) ON DELETE RESTRICT
);
