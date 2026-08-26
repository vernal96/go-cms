BEGIN;

CREATE EXTENSION IF NOT EXISTS pg_trgm;

LOCK TABLE core.library_items IN ACCESS EXCLUSIVE MODE;

CREATE TABLE core.library_items_v2
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
    CONSTRAINT fk_library_items_v2_entity_site FOREIGN KEY (id, site_id) REFERENCES core.resource_entities (id, site_id) ON DELETE CASCADE,
    CONSTRAINT fk_library_items_v2_library_site FOREIGN KEY (library_id, site_id) REFERENCES core.resources (id, site_id) ON DELETE CASCADE,
    CONSTRAINT fk_library_items_v2_image_media FOREIGN KEY (image_media_id) REFERENCES core.media (id) ON DELETE RESTRICT,
    CONSTRAINT fk_library_items_v2_created_by FOREIGN KEY (created_by) REFERENCES core.users (id) ON DELETE SET NULL,
    CONSTRAINT fk_library_items_v2_updated_by FOREIGN KEY (updated_by) REFERENCES core.users (id) ON DELETE SET NULL,
    CONSTRAINT fk_library_items_v2_deleted_by FOREIGN KEY (deleted_by) REFERENCES core.users (id) ON DELETE SET NULL,
    CONSTRAINT ck_library_items_v2_publication_window CHECK (published_at IS NULL OR unpublished_at IS NULL OR unpublished_at > published_at)
) PARTITION BY HASH (library_id);

CREATE TABLE core.library_items_v2_h0
    PARTITION OF core.library_items_v2
    FOR VALUES WITH (MODULUS 8, REMAINDER 0)
    PARTITION BY RANGE (partition_at);

CREATE TABLE core.library_items_h0_y2020
    PARTITION OF core.library_items_v2_h0
    FOR VALUES FROM ('2020-01-01 00:00:00+00') TO ('2021-01-01 00:00:00+00');

CREATE TABLE core.library_items_h0_y2021
    PARTITION OF core.library_items_v2_h0
    FOR VALUES FROM ('2021-01-01 00:00:00+00') TO ('2022-01-01 00:00:00+00');

CREATE TABLE core.library_items_h0_y2022
    PARTITION OF core.library_items_v2_h0
    FOR VALUES FROM ('2022-01-01 00:00:00+00') TO ('2023-01-01 00:00:00+00');

CREATE TABLE core.library_items_h0_y2023
    PARTITION OF core.library_items_v2_h0
    FOR VALUES FROM ('2023-01-01 00:00:00+00') TO ('2024-01-01 00:00:00+00');

CREATE TABLE core.library_items_h0_y2024
    PARTITION OF core.library_items_v2_h0
    FOR VALUES FROM ('2024-01-01 00:00:00+00') TO ('2025-01-01 00:00:00+00');

CREATE TABLE core.library_items_h0_y2025
    PARTITION OF core.library_items_v2_h0
    FOR VALUES FROM ('2025-01-01 00:00:00+00') TO ('2026-01-01 00:00:00+00');

CREATE TABLE core.library_items_h0_y2026
    PARTITION OF core.library_items_v2_h0
    FOR VALUES FROM ('2026-01-01 00:00:00+00') TO ('2027-01-01 00:00:00+00');

CREATE TABLE core.library_items_h0_y2027
    PARTITION OF core.library_items_v2_h0
    FOR VALUES FROM ('2027-01-01 00:00:00+00') TO ('2028-01-01 00:00:00+00');

CREATE TABLE core.library_items_h0_y2028
    PARTITION OF core.library_items_v2_h0
    FOR VALUES FROM ('2028-01-01 00:00:00+00') TO ('2029-01-01 00:00:00+00');

CREATE TABLE core.library_items_h0_y2029
    PARTITION OF core.library_items_v2_h0
    FOR VALUES FROM ('2029-01-01 00:00:00+00') TO ('2030-01-01 00:00:00+00');

CREATE TABLE core.library_items_h0_y2030
    PARTITION OF core.library_items_v2_h0
    FOR VALUES FROM ('2030-01-01 00:00:00+00') TO ('2031-01-01 00:00:00+00');

CREATE TABLE core.library_items_h0_y2031
    PARTITION OF core.library_items_v2_h0
    FOR VALUES FROM ('2031-01-01 00:00:00+00') TO ('2032-01-01 00:00:00+00');

CREATE TABLE core.library_items_h0_default
    PARTITION OF core.library_items_v2_h0 DEFAULT;

CREATE TABLE core.library_items_v2_h1
    PARTITION OF core.library_items_v2
    FOR VALUES WITH (MODULUS 8, REMAINDER 1)
    PARTITION BY RANGE (partition_at);

CREATE TABLE core.library_items_h1_y2020
    PARTITION OF core.library_items_v2_h1
    FOR VALUES FROM ('2020-01-01 00:00:00+00') TO ('2021-01-01 00:00:00+00');

CREATE TABLE core.library_items_h1_y2021
    PARTITION OF core.library_items_v2_h1
    FOR VALUES FROM ('2021-01-01 00:00:00+00') TO ('2022-01-01 00:00:00+00');

CREATE TABLE core.library_items_h1_y2022
    PARTITION OF core.library_items_v2_h1
    FOR VALUES FROM ('2022-01-01 00:00:00+00') TO ('2023-01-01 00:00:00+00');

CREATE TABLE core.library_items_h1_y2023
    PARTITION OF core.library_items_v2_h1
    FOR VALUES FROM ('2023-01-01 00:00:00+00') TO ('2024-01-01 00:00:00+00');

CREATE TABLE core.library_items_h1_y2024
    PARTITION OF core.library_items_v2_h1
    FOR VALUES FROM ('2024-01-01 00:00:00+00') TO ('2025-01-01 00:00:00+00');

CREATE TABLE core.library_items_h1_y2025
    PARTITION OF core.library_items_v2_h1
    FOR VALUES FROM ('2025-01-01 00:00:00+00') TO ('2026-01-01 00:00:00+00');

CREATE TABLE core.library_items_h1_y2026
    PARTITION OF core.library_items_v2_h1
    FOR VALUES FROM ('2026-01-01 00:00:00+00') TO ('2027-01-01 00:00:00+00');

CREATE TABLE core.library_items_h1_y2027
    PARTITION OF core.library_items_v2_h1
    FOR VALUES FROM ('2027-01-01 00:00:00+00') TO ('2028-01-01 00:00:00+00');

CREATE TABLE core.library_items_h1_y2028
    PARTITION OF core.library_items_v2_h1
    FOR VALUES FROM ('2028-01-01 00:00:00+00') TO ('2029-01-01 00:00:00+00');

CREATE TABLE core.library_items_h1_y2029
    PARTITION OF core.library_items_v2_h1
    FOR VALUES FROM ('2029-01-01 00:00:00+00') TO ('2030-01-01 00:00:00+00');

CREATE TABLE core.library_items_h1_y2030
    PARTITION OF core.library_items_v2_h1
    FOR VALUES FROM ('2030-01-01 00:00:00+00') TO ('2031-01-01 00:00:00+00');

CREATE TABLE core.library_items_h1_y2031
    PARTITION OF core.library_items_v2_h1
    FOR VALUES FROM ('2031-01-01 00:00:00+00') TO ('2032-01-01 00:00:00+00');

CREATE TABLE core.library_items_h1_default
    PARTITION OF core.library_items_v2_h1 DEFAULT;

CREATE TABLE core.library_items_v2_h2
    PARTITION OF core.library_items_v2
    FOR VALUES WITH (MODULUS 8, REMAINDER 2)
    PARTITION BY RANGE (partition_at);

CREATE TABLE core.library_items_h2_y2020
    PARTITION OF core.library_items_v2_h2
    FOR VALUES FROM ('2020-01-01 00:00:00+00') TO ('2021-01-01 00:00:00+00');

CREATE TABLE core.library_items_h2_y2021
    PARTITION OF core.library_items_v2_h2
    FOR VALUES FROM ('2021-01-01 00:00:00+00') TO ('2022-01-01 00:00:00+00');

CREATE TABLE core.library_items_h2_y2022
    PARTITION OF core.library_items_v2_h2
    FOR VALUES FROM ('2022-01-01 00:00:00+00') TO ('2023-01-01 00:00:00+00');

CREATE TABLE core.library_items_h2_y2023
    PARTITION OF core.library_items_v2_h2
    FOR VALUES FROM ('2023-01-01 00:00:00+00') TO ('2024-01-01 00:00:00+00');

CREATE TABLE core.library_items_h2_y2024
    PARTITION OF core.library_items_v2_h2
    FOR VALUES FROM ('2024-01-01 00:00:00+00') TO ('2025-01-01 00:00:00+00');

CREATE TABLE core.library_items_h2_y2025
    PARTITION OF core.library_items_v2_h2
    FOR VALUES FROM ('2025-01-01 00:00:00+00') TO ('2026-01-01 00:00:00+00');

CREATE TABLE core.library_items_h2_y2026
    PARTITION OF core.library_items_v2_h2
    FOR VALUES FROM ('2026-01-01 00:00:00+00') TO ('2027-01-01 00:00:00+00');

CREATE TABLE core.library_items_h2_y2027
    PARTITION OF core.library_items_v2_h2
    FOR VALUES FROM ('2027-01-01 00:00:00+00') TO ('2028-01-01 00:00:00+00');

CREATE TABLE core.library_items_h2_y2028
    PARTITION OF core.library_items_v2_h2
    FOR VALUES FROM ('2028-01-01 00:00:00+00') TO ('2029-01-01 00:00:00+00');

CREATE TABLE core.library_items_h2_y2029
    PARTITION OF core.library_items_v2_h2
    FOR VALUES FROM ('2029-01-01 00:00:00+00') TO ('2030-01-01 00:00:00+00');

CREATE TABLE core.library_items_h2_y2030
    PARTITION OF core.library_items_v2_h2
    FOR VALUES FROM ('2030-01-01 00:00:00+00') TO ('2031-01-01 00:00:00+00');

CREATE TABLE core.library_items_h2_y2031
    PARTITION OF core.library_items_v2_h2
    FOR VALUES FROM ('2031-01-01 00:00:00+00') TO ('2032-01-01 00:00:00+00');

CREATE TABLE core.library_items_h2_default
    PARTITION OF core.library_items_v2_h2 DEFAULT;

CREATE TABLE core.library_items_v2_h3
    PARTITION OF core.library_items_v2
    FOR VALUES WITH (MODULUS 8, REMAINDER 3)
    PARTITION BY RANGE (partition_at);

CREATE TABLE core.library_items_h3_y2020
    PARTITION OF core.library_items_v2_h3
    FOR VALUES FROM ('2020-01-01 00:00:00+00') TO ('2021-01-01 00:00:00+00');

CREATE TABLE core.library_items_h3_y2021
    PARTITION OF core.library_items_v2_h3
    FOR VALUES FROM ('2021-01-01 00:00:00+00') TO ('2022-01-01 00:00:00+00');

CREATE TABLE core.library_items_h3_y2022
    PARTITION OF core.library_items_v2_h3
    FOR VALUES FROM ('2022-01-01 00:00:00+00') TO ('2023-01-01 00:00:00+00');

CREATE TABLE core.library_items_h3_y2023
    PARTITION OF core.library_items_v2_h3
    FOR VALUES FROM ('2023-01-01 00:00:00+00') TO ('2024-01-01 00:00:00+00');

CREATE TABLE core.library_items_h3_y2024
    PARTITION OF core.library_items_v2_h3
    FOR VALUES FROM ('2024-01-01 00:00:00+00') TO ('2025-01-01 00:00:00+00');

CREATE TABLE core.library_items_h3_y2025
    PARTITION OF core.library_items_v2_h3
    FOR VALUES FROM ('2025-01-01 00:00:00+00') TO ('2026-01-01 00:00:00+00');

CREATE TABLE core.library_items_h3_y2026
    PARTITION OF core.library_items_v2_h3
    FOR VALUES FROM ('2026-01-01 00:00:00+00') TO ('2027-01-01 00:00:00+00');

CREATE TABLE core.library_items_h3_y2027
    PARTITION OF core.library_items_v2_h3
    FOR VALUES FROM ('2027-01-01 00:00:00+00') TO ('2028-01-01 00:00:00+00');

CREATE TABLE core.library_items_h3_y2028
    PARTITION OF core.library_items_v2_h3
    FOR VALUES FROM ('2028-01-01 00:00:00+00') TO ('2029-01-01 00:00:00+00');

CREATE TABLE core.library_items_h3_y2029
    PARTITION OF core.library_items_v2_h3
    FOR VALUES FROM ('2029-01-01 00:00:00+00') TO ('2030-01-01 00:00:00+00');

CREATE TABLE core.library_items_h3_y2030
    PARTITION OF core.library_items_v2_h3
    FOR VALUES FROM ('2030-01-01 00:00:00+00') TO ('2031-01-01 00:00:00+00');

CREATE TABLE core.library_items_h3_y2031
    PARTITION OF core.library_items_v2_h3
    FOR VALUES FROM ('2031-01-01 00:00:00+00') TO ('2032-01-01 00:00:00+00');

CREATE TABLE core.library_items_h3_default
    PARTITION OF core.library_items_v2_h3 DEFAULT;

CREATE TABLE core.library_items_v2_h4
    PARTITION OF core.library_items_v2
    FOR VALUES WITH (MODULUS 8, REMAINDER 4)
    PARTITION BY RANGE (partition_at);

CREATE TABLE core.library_items_h4_y2020
    PARTITION OF core.library_items_v2_h4
    FOR VALUES FROM ('2020-01-01 00:00:00+00') TO ('2021-01-01 00:00:00+00');

CREATE TABLE core.library_items_h4_y2021
    PARTITION OF core.library_items_v2_h4
    FOR VALUES FROM ('2021-01-01 00:00:00+00') TO ('2022-01-01 00:00:00+00');

CREATE TABLE core.library_items_h4_y2022
    PARTITION OF core.library_items_v2_h4
    FOR VALUES FROM ('2022-01-01 00:00:00+00') TO ('2023-01-01 00:00:00+00');

CREATE TABLE core.library_items_h4_y2023
    PARTITION OF core.library_items_v2_h4
    FOR VALUES FROM ('2023-01-01 00:00:00+00') TO ('2024-01-01 00:00:00+00');

CREATE TABLE core.library_items_h4_y2024
    PARTITION OF core.library_items_v2_h4
    FOR VALUES FROM ('2024-01-01 00:00:00+00') TO ('2025-01-01 00:00:00+00');

CREATE TABLE core.library_items_h4_y2025
    PARTITION OF core.library_items_v2_h4
    FOR VALUES FROM ('2025-01-01 00:00:00+00') TO ('2026-01-01 00:00:00+00');

CREATE TABLE core.library_items_h4_y2026
    PARTITION OF core.library_items_v2_h4
    FOR VALUES FROM ('2026-01-01 00:00:00+00') TO ('2027-01-01 00:00:00+00');

CREATE TABLE core.library_items_h4_y2027
    PARTITION OF core.library_items_v2_h4
    FOR VALUES FROM ('2027-01-01 00:00:00+00') TO ('2028-01-01 00:00:00+00');

CREATE TABLE core.library_items_h4_y2028
    PARTITION OF core.library_items_v2_h4
    FOR VALUES FROM ('2028-01-01 00:00:00+00') TO ('2029-01-01 00:00:00+00');

CREATE TABLE core.library_items_h4_y2029
    PARTITION OF core.library_items_v2_h4
    FOR VALUES FROM ('2029-01-01 00:00:00+00') TO ('2030-01-01 00:00:00+00');

CREATE TABLE core.library_items_h4_y2030
    PARTITION OF core.library_items_v2_h4
    FOR VALUES FROM ('2030-01-01 00:00:00+00') TO ('2031-01-01 00:00:00+00');

CREATE TABLE core.library_items_h4_y2031
    PARTITION OF core.library_items_v2_h4
    FOR VALUES FROM ('2031-01-01 00:00:00+00') TO ('2032-01-01 00:00:00+00');

CREATE TABLE core.library_items_h4_default
    PARTITION OF core.library_items_v2_h4 DEFAULT;

CREATE TABLE core.library_items_v2_h5
    PARTITION OF core.library_items_v2
    FOR VALUES WITH (MODULUS 8, REMAINDER 5)
    PARTITION BY RANGE (partition_at);

CREATE TABLE core.library_items_h5_y2020
    PARTITION OF core.library_items_v2_h5
    FOR VALUES FROM ('2020-01-01 00:00:00+00') TO ('2021-01-01 00:00:00+00');

CREATE TABLE core.library_items_h5_y2021
    PARTITION OF core.library_items_v2_h5
    FOR VALUES FROM ('2021-01-01 00:00:00+00') TO ('2022-01-01 00:00:00+00');

CREATE TABLE core.library_items_h5_y2022
    PARTITION OF core.library_items_v2_h5
    FOR VALUES FROM ('2022-01-01 00:00:00+00') TO ('2023-01-01 00:00:00+00');

CREATE TABLE core.library_items_h5_y2023
    PARTITION OF core.library_items_v2_h5
    FOR VALUES FROM ('2023-01-01 00:00:00+00') TO ('2024-01-01 00:00:00+00');

CREATE TABLE core.library_items_h5_y2024
    PARTITION OF core.library_items_v2_h5
    FOR VALUES FROM ('2024-01-01 00:00:00+00') TO ('2025-01-01 00:00:00+00');

CREATE TABLE core.library_items_h5_y2025
    PARTITION OF core.library_items_v2_h5
    FOR VALUES FROM ('2025-01-01 00:00:00+00') TO ('2026-01-01 00:00:00+00');

CREATE TABLE core.library_items_h5_y2026
    PARTITION OF core.library_items_v2_h5
    FOR VALUES FROM ('2026-01-01 00:00:00+00') TO ('2027-01-01 00:00:00+00');

CREATE TABLE core.library_items_h5_y2027
    PARTITION OF core.library_items_v2_h5
    FOR VALUES FROM ('2027-01-01 00:00:00+00') TO ('2028-01-01 00:00:00+00');

CREATE TABLE core.library_items_h5_y2028
    PARTITION OF core.library_items_v2_h5
    FOR VALUES FROM ('2028-01-01 00:00:00+00') TO ('2029-01-01 00:00:00+00');

CREATE TABLE core.library_items_h5_y2029
    PARTITION OF core.library_items_v2_h5
    FOR VALUES FROM ('2029-01-01 00:00:00+00') TO ('2030-01-01 00:00:00+00');

CREATE TABLE core.library_items_h5_y2030
    PARTITION OF core.library_items_v2_h5
    FOR VALUES FROM ('2030-01-01 00:00:00+00') TO ('2031-01-01 00:00:00+00');

CREATE TABLE core.library_items_h5_y2031
    PARTITION OF core.library_items_v2_h5
    FOR VALUES FROM ('2031-01-01 00:00:00+00') TO ('2032-01-01 00:00:00+00');

CREATE TABLE core.library_items_h5_default
    PARTITION OF core.library_items_v2_h5 DEFAULT;

CREATE TABLE core.library_items_v2_h6
    PARTITION OF core.library_items_v2
    FOR VALUES WITH (MODULUS 8, REMAINDER 6)
    PARTITION BY RANGE (partition_at);

CREATE TABLE core.library_items_h6_y2020
    PARTITION OF core.library_items_v2_h6
    FOR VALUES FROM ('2020-01-01 00:00:00+00') TO ('2021-01-01 00:00:00+00');

CREATE TABLE core.library_items_h6_y2021
    PARTITION OF core.library_items_v2_h6
    FOR VALUES FROM ('2021-01-01 00:00:00+00') TO ('2022-01-01 00:00:00+00');

CREATE TABLE core.library_items_h6_y2022
    PARTITION OF core.library_items_v2_h6
    FOR VALUES FROM ('2022-01-01 00:00:00+00') TO ('2023-01-01 00:00:00+00');

CREATE TABLE core.library_items_h6_y2023
    PARTITION OF core.library_items_v2_h6
    FOR VALUES FROM ('2023-01-01 00:00:00+00') TO ('2024-01-01 00:00:00+00');

CREATE TABLE core.library_items_h6_y2024
    PARTITION OF core.library_items_v2_h6
    FOR VALUES FROM ('2024-01-01 00:00:00+00') TO ('2025-01-01 00:00:00+00');

CREATE TABLE core.library_items_h6_y2025
    PARTITION OF core.library_items_v2_h6
    FOR VALUES FROM ('2025-01-01 00:00:00+00') TO ('2026-01-01 00:00:00+00');

CREATE TABLE core.library_items_h6_y2026
    PARTITION OF core.library_items_v2_h6
    FOR VALUES FROM ('2026-01-01 00:00:00+00') TO ('2027-01-01 00:00:00+00');

CREATE TABLE core.library_items_h6_y2027
    PARTITION OF core.library_items_v2_h6
    FOR VALUES FROM ('2027-01-01 00:00:00+00') TO ('2028-01-01 00:00:00+00');

CREATE TABLE core.library_items_h6_y2028
    PARTITION OF core.library_items_v2_h6
    FOR VALUES FROM ('2028-01-01 00:00:00+00') TO ('2029-01-01 00:00:00+00');

CREATE TABLE core.library_items_h6_y2029
    PARTITION OF core.library_items_v2_h6
    FOR VALUES FROM ('2029-01-01 00:00:00+00') TO ('2030-01-01 00:00:00+00');

CREATE TABLE core.library_items_h6_y2030
    PARTITION OF core.library_items_v2_h6
    FOR VALUES FROM ('2030-01-01 00:00:00+00') TO ('2031-01-01 00:00:00+00');

CREATE TABLE core.library_items_h6_y2031
    PARTITION OF core.library_items_v2_h6
    FOR VALUES FROM ('2031-01-01 00:00:00+00') TO ('2032-01-01 00:00:00+00');

CREATE TABLE core.library_items_h6_default
    PARTITION OF core.library_items_v2_h6 DEFAULT;

CREATE TABLE core.library_items_v2_h7
    PARTITION OF core.library_items_v2
    FOR VALUES WITH (MODULUS 8, REMAINDER 7)
    PARTITION BY RANGE (partition_at);

CREATE TABLE core.library_items_h7_y2020
    PARTITION OF core.library_items_v2_h7
    FOR VALUES FROM ('2020-01-01 00:00:00+00') TO ('2021-01-01 00:00:00+00');

CREATE TABLE core.library_items_h7_y2021
    PARTITION OF core.library_items_v2_h7
    FOR VALUES FROM ('2021-01-01 00:00:00+00') TO ('2022-01-01 00:00:00+00');

CREATE TABLE core.library_items_h7_y2022
    PARTITION OF core.library_items_v2_h7
    FOR VALUES FROM ('2022-01-01 00:00:00+00') TO ('2023-01-01 00:00:00+00');

CREATE TABLE core.library_items_h7_y2023
    PARTITION OF core.library_items_v2_h7
    FOR VALUES FROM ('2023-01-01 00:00:00+00') TO ('2024-01-01 00:00:00+00');

CREATE TABLE core.library_items_h7_y2024
    PARTITION OF core.library_items_v2_h7
    FOR VALUES FROM ('2024-01-01 00:00:00+00') TO ('2025-01-01 00:00:00+00');

CREATE TABLE core.library_items_h7_y2025
    PARTITION OF core.library_items_v2_h7
    FOR VALUES FROM ('2025-01-01 00:00:00+00') TO ('2026-01-01 00:00:00+00');

CREATE TABLE core.library_items_h7_y2026
    PARTITION OF core.library_items_v2_h7
    FOR VALUES FROM ('2026-01-01 00:00:00+00') TO ('2027-01-01 00:00:00+00');

CREATE TABLE core.library_items_h7_y2027
    PARTITION OF core.library_items_v2_h7
    FOR VALUES FROM ('2027-01-01 00:00:00+00') TO ('2028-01-01 00:00:00+00');

CREATE TABLE core.library_items_h7_y2028
    PARTITION OF core.library_items_v2_h7
    FOR VALUES FROM ('2028-01-01 00:00:00+00') TO ('2029-01-01 00:00:00+00');

CREATE TABLE core.library_items_h7_y2029
    PARTITION OF core.library_items_v2_h7
    FOR VALUES FROM ('2029-01-01 00:00:00+00') TO ('2030-01-01 00:00:00+00');

CREATE TABLE core.library_items_h7_y2030
    PARTITION OF core.library_items_v2_h7
    FOR VALUES FROM ('2030-01-01 00:00:00+00') TO ('2031-01-01 00:00:00+00');

CREATE TABLE core.library_items_h7_y2031
    PARTITION OF core.library_items_v2_h7
    FOR VALUES FROM ('2031-01-01 00:00:00+00') TO ('2032-01-01 00:00:00+00');

CREATE TABLE core.library_items_h7_default
    PARTITION OF core.library_items_v2_h7 DEFAULT;

INSERT INTO core.library_items_v2 SELECT * FROM core.library_items;

CREATE TEMP TABLE library_items_rebuild_preflight
(
    matched BOOLEAN NOT NULL,
    CONSTRAINT library_items_rebuild_row_count_must_match CHECK (matched)
);
INSERT INTO library_items_rebuild_preflight (matched)
SELECT (SELECT count(*) FROM core.library_items) = (SELECT count(*) FROM core.library_items_v2);
DROP TABLE library_items_rebuild_preflight;

DROP TABLE core.library_items;
ALTER TABLE core.library_items_v2 RENAME TO library_items;

ALTER TABLE core.library_items_v2_h0 RENAME TO library_items_h0;
ALTER TABLE core.library_items_v2_h1 RENAME TO library_items_h1;
ALTER TABLE core.library_items_v2_h2 RENAME TO library_items_h2;
ALTER TABLE core.library_items_v2_h3 RENAME TO library_items_h3;
ALTER TABLE core.library_items_v2_h4 RENAME TO library_items_h4;
ALTER TABLE core.library_items_v2_h5 RENAME TO library_items_h5;
ALTER TABLE core.library_items_v2_h6 RENAME TO library_items_h6;
ALTER TABLE core.library_items_v2_h7 RENAME TO library_items_h7;

CREATE INDEX idx_library_items_library_created
    ON core.library_items (library_id, created_at DESC, id DESC);
CREATE INDEX idx_library_items_library_publication
    ON core.library_items (library_id, is_public, published_at DESC, id DESC);
CREATE INDEX idx_library_items_title_trgm
    ON core.library_items USING gin (lower(title) gin_trgm_ops);
CREATE INDEX idx_library_items_slug_trgm
    ON core.library_items USING gin (lower(slug) gin_trgm_ops);

COMMIT;
