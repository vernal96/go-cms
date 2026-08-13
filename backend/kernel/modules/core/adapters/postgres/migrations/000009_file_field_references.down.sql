DROP TABLE IF EXISTS core.file_field_references;

ALTER TABLE core.media
    DROP CONSTRAINT fk_media_file,
    ADD CONSTRAINT fk_media_file
        FOREIGN KEY (file_id)
            REFERENCES core.files (id)
            ON DELETE CASCADE;
