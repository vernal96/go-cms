BEGIN;

LOCK TABLE core.library_items IN ACCESS EXCLUSIVE MODE;

CREATE TABLE core.library_items_v1
(
    id BIGINT NOT NULL,
    site_id BIGINT NOT NULL,
    library_id BIGINT NOT NULL,
    partition_at TIMESTAMPTZ NOT NULL,
    template TEXT NULL,
    content_type TEXT NULL DEFAULT 'html',
    title TEXT NOT NULL CHECK (btrim(title) <> ''),
    slug TEXT NOT NULL CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    annotation TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    image_media_id BIGINT NULL,
    is_public BOOLEAN NOT NULL DEFAULT TRUE,
    is_searchable BOOLEAN NOT NULL DEFAULT TRUE,
    published_at TIMESTAMPTZ NULL,
    unpublished_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by BIGINT NULL,
    updated_by BIGINT NULL,
    deleted_at TIMESTAMPTZ NULL,
    deleted_by BIGINT NULL,
    PRIMARY KEY (id, library_id, partition_at),
    CONSTRAINT fk_library_items_v1_entity_site FOREIGN KEY (id, site_id) REFERENCES core.resource_entities (id, site_id) ON DELETE CASCADE,
    CONSTRAINT fk_library_items_v1_library_site FOREIGN KEY (library_id, site_id) REFERENCES core.resources (id, site_id) ON DELETE CASCADE,
    CONSTRAINT fk_library_items_v1_image_media FOREIGN KEY (image_media_id) REFERENCES core.media (id) ON DELETE RESTRICT,
    CONSTRAINT fk_library_items_v1_created_by FOREIGN KEY (created_by) REFERENCES core.users (id) ON DELETE SET NULL,
    CONSTRAINT fk_library_items_v1_updated_by FOREIGN KEY (updated_by) REFERENCES core.users (id) ON DELETE SET NULL,
    CONSTRAINT fk_library_items_v1_deleted_by FOREIGN KEY (deleted_by) REFERENCES core.users (id) ON DELETE SET NULL,
    CONSTRAINT ck_library_items_v1_publication_window CHECK (published_at IS NULL OR unpublished_at IS NULL OR unpublished_at > published_at)
) PARTITION BY HASH (library_id);

CREATE TABLE core.library_items_v1_h0
    PARTITION OF core.library_items_v1
    FOR VALUES WITH (MODULUS 8, REMAINDER 0)
    PARTITION BY RANGE (partition_at);

CREATE TABLE core.library_items_h0_before_2020
    PARTITION OF core.library_items_v1_h0
    FOR VALUES FROM (MINVALUE) TO ('2020-01-01 00:00:00+00');

CREATE TABLE core.library_items_h0_2020s
    PARTITION OF core.library_items_v1_h0
    FOR VALUES FROM ('2020-01-01 00:00:00+00') TO ('2030-01-01 00:00:00+00');

CREATE TABLE core.library_items_h0_after_2030
    PARTITION OF core.library_items_v1_h0
    FOR VALUES FROM ('2030-01-01 00:00:00+00') TO (MAXVALUE);

CREATE TABLE core.library_items_v1_h1
    PARTITION OF core.library_items_v1
    FOR VALUES WITH (MODULUS 8, REMAINDER 1)
    PARTITION BY RANGE (partition_at);

CREATE TABLE core.library_items_h1_before_2020
    PARTITION OF core.library_items_v1_h1
    FOR VALUES FROM (MINVALUE) TO ('2020-01-01 00:00:00+00');

CREATE TABLE core.library_items_h1_2020s
    PARTITION OF core.library_items_v1_h1
    FOR VALUES FROM ('2020-01-01 00:00:00+00') TO ('2030-01-01 00:00:00+00');

CREATE TABLE core.library_items_h1_after_2030
    PARTITION OF core.library_items_v1_h1
    FOR VALUES FROM ('2030-01-01 00:00:00+00') TO (MAXVALUE);

CREATE TABLE core.library_items_v1_h2
    PARTITION OF core.library_items_v1
    FOR VALUES WITH (MODULUS 8, REMAINDER 2)
    PARTITION BY RANGE (partition_at);

CREATE TABLE core.library_items_h2_before_2020
    PARTITION OF core.library_items_v1_h2
    FOR VALUES FROM (MINVALUE) TO ('2020-01-01 00:00:00+00');

CREATE TABLE core.library_items_h2_2020s
    PARTITION OF core.library_items_v1_h2
    FOR VALUES FROM ('2020-01-01 00:00:00+00') TO ('2030-01-01 00:00:00+00');

CREATE TABLE core.library_items_h2_after_2030
    PARTITION OF core.library_items_v1_h2
    FOR VALUES FROM ('2030-01-01 00:00:00+00') TO (MAXVALUE);

CREATE TABLE core.library_items_v1_h3
    PARTITION OF core.library_items_v1
    FOR VALUES WITH (MODULUS 8, REMAINDER 3)
    PARTITION BY RANGE (partition_at);

CREATE TABLE core.library_items_h3_before_2020
    PARTITION OF core.library_items_v1_h3
    FOR VALUES FROM (MINVALUE) TO ('2020-01-01 00:00:00+00');

CREATE TABLE core.library_items_h3_2020s
    PARTITION OF core.library_items_v1_h3
    FOR VALUES FROM ('2020-01-01 00:00:00+00') TO ('2030-01-01 00:00:00+00');

CREATE TABLE core.library_items_h3_after_2030
    PARTITION OF core.library_items_v1_h3
    FOR VALUES FROM ('2030-01-01 00:00:00+00') TO (MAXVALUE);

CREATE TABLE core.library_items_v1_h4
    PARTITION OF core.library_items_v1
    FOR VALUES WITH (MODULUS 8, REMAINDER 4)
    PARTITION BY RANGE (partition_at);

CREATE TABLE core.library_items_h4_before_2020
    PARTITION OF core.library_items_v1_h4
    FOR VALUES FROM (MINVALUE) TO ('2020-01-01 00:00:00+00');

CREATE TABLE core.library_items_h4_2020s
    PARTITION OF core.library_items_v1_h4
    FOR VALUES FROM ('2020-01-01 00:00:00+00') TO ('2030-01-01 00:00:00+00');

CREATE TABLE core.library_items_h4_after_2030
    PARTITION OF core.library_items_v1_h4
    FOR VALUES FROM ('2030-01-01 00:00:00+00') TO (MAXVALUE);

CREATE TABLE core.library_items_v1_h5
    PARTITION OF core.library_items_v1
    FOR VALUES WITH (MODULUS 8, REMAINDER 5)
    PARTITION BY RANGE (partition_at);

CREATE TABLE core.library_items_h5_before_2020
    PARTITION OF core.library_items_v1_h5
    FOR VALUES FROM (MINVALUE) TO ('2020-01-01 00:00:00+00');

CREATE TABLE core.library_items_h5_2020s
    PARTITION OF core.library_items_v1_h5
    FOR VALUES FROM ('2020-01-01 00:00:00+00') TO ('2030-01-01 00:00:00+00');

CREATE TABLE core.library_items_h5_after_2030
    PARTITION OF core.library_items_v1_h5
    FOR VALUES FROM ('2030-01-01 00:00:00+00') TO (MAXVALUE);

CREATE TABLE core.library_items_v1_h6
    PARTITION OF core.library_items_v1
    FOR VALUES WITH (MODULUS 8, REMAINDER 6)
    PARTITION BY RANGE (partition_at);

CREATE TABLE core.library_items_h6_before_2020
    PARTITION OF core.library_items_v1_h6
    FOR VALUES FROM (MINVALUE) TO ('2020-01-01 00:00:00+00');

CREATE TABLE core.library_items_h6_2020s
    PARTITION OF core.library_items_v1_h6
    FOR VALUES FROM ('2020-01-01 00:00:00+00') TO ('2030-01-01 00:00:00+00');

CREATE TABLE core.library_items_h6_after_2030
    PARTITION OF core.library_items_v1_h6
    FOR VALUES FROM ('2030-01-01 00:00:00+00') TO (MAXVALUE);

CREATE TABLE core.library_items_v1_h7
    PARTITION OF core.library_items_v1
    FOR VALUES WITH (MODULUS 8, REMAINDER 7)
    PARTITION BY RANGE (partition_at);

CREATE TABLE core.library_items_h7_before_2020
    PARTITION OF core.library_items_v1_h7
    FOR VALUES FROM (MINVALUE) TO ('2020-01-01 00:00:00+00');

CREATE TABLE core.library_items_h7_2020s
    PARTITION OF core.library_items_v1_h7
    FOR VALUES FROM ('2020-01-01 00:00:00+00') TO ('2030-01-01 00:00:00+00');

CREATE TABLE core.library_items_h7_after_2030
    PARTITION OF core.library_items_v1_h7
    FOR VALUES FROM ('2030-01-01 00:00:00+00') TO (MAXVALUE);

INSERT INTO core.library_items_v1 SELECT * FROM core.library_items;

CREATE TEMP TABLE library_items_down_rebuild_preflight
(
    matched BOOLEAN NOT NULL,
    CONSTRAINT library_items_down_rebuild_row_count_must_match CHECK (matched)
);
INSERT INTO library_items_down_rebuild_preflight (matched)
SELECT (SELECT count(*) FROM core.library_items) = (SELECT count(*) FROM core.library_items_v1);
DROP TABLE library_items_down_rebuild_preflight;

DROP TABLE core.library_items;
ALTER TABLE core.library_items_v1 RENAME TO library_items;

ALTER TABLE core.library_items_v1_h0 RENAME TO library_items_h0;
ALTER TABLE core.library_items_v1_h1 RENAME TO library_items_h1;
ALTER TABLE core.library_items_v1_h2 RENAME TO library_items_h2;
ALTER TABLE core.library_items_v1_h3 RENAME TO library_items_h3;
ALTER TABLE core.library_items_v1_h4 RENAME TO library_items_h4;
ALTER TABLE core.library_items_v1_h5 RENAME TO library_items_h5;
ALTER TABLE core.library_items_v1_h6 RENAME TO library_items_h6;
ALTER TABLE core.library_items_v1_h7 RENAME TO library_items_h7;

CREATE INDEX idx_library_items_library_created
    ON core.library_items (library_id, created_at DESC, id DESC);
CREATE INDEX idx_library_items_library_publication
    ON core.library_items (library_id, is_public, published_at DESC, id DESC);

COMMIT;
