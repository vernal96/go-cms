BEGIN;

CREATE TABLE IF NOT EXISTS sites (
	id BIGSERIAL PRIMARY KEY,
	profile_code TEXT NOT NULL,
	domain TEXT NOT NULL,
	locale TEXT NOT NULL DEFAULT 'ru',
	settings JSONB NOT NULL DEFAULT '{}'::jsonb,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	created_by BIGINT NULL,
	updated_by BIGINT NULL
);

ALTER TABLE sites ADD COLUMN IF NOT EXISTS created_by BIGINT NULL;
ALTER TABLE sites ADD COLUMN IF NOT EXISTS updated_by BIGINT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_sites_domain ON sites(domain);
CREATE INDEX IF NOT EXISTS idx_sites_profile_code ON sites(profile_code);

CREATE TABLE IF NOT EXISTS users (
	id BIGSERIAL PRIMARY KEY,
	email TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	name TEXT NOT NULL,
	avatar_media_id BIGINT NULL,
	settings JSONB NOT NULL DEFAULT '{}'::jsonb,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	created_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
	updated_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_users_avatar_media_id ON users(avatar_media_id);

CREATE TABLE IF NOT EXISTS user_groups (
	user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	group_code TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	created_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
	updated_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
	PRIMARY KEY (user_id, group_code)
);

CREATE INDEX IF NOT EXISTS idx_user_groups_group_code ON user_groups(group_code);

CREATE TABLE IF NOT EXISTS site_permissions (
	site_id BIGINT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
	group_code TEXT NOT NULL,
	permission TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	created_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
	updated_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
	UNIQUE (site_id, group_code, permission)
);

CREATE INDEX IF NOT EXISTS idx_site_permissions_group_code ON site_permissions(group_code);
CREATE INDEX IF NOT EXISTS idx_site_permissions_permission ON site_permissions(permission);

CREATE TABLE IF NOT EXISTS file_folders (
	id BIGSERIAL PRIMARY KEY,
	parent_id BIGINT NULL REFERENCES file_folders(id) ON DELETE RESTRICT,
	disk TEXT NOT NULL,
	name TEXT NOT NULL,
	sort INTEGER NOT NULL DEFAULT 0,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	created_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
	updated_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
	UNIQUE (parent_id, disk, name)
);

CREATE INDEX IF NOT EXISTS idx_file_folders_parent_id ON file_folders(parent_id);
CREATE INDEX IF NOT EXISTS idx_file_folders_disk ON file_folders(disk);

CREATE TABLE IF NOT EXISTS files (
	id BIGSERIAL PRIMARY KEY,
	folder_id BIGINT NULL REFERENCES file_folders(id) ON DELETE SET NULL,
	disk TEXT NOT NULL,
	path TEXT NOT NULL,
	name TEXT NOT NULL,
	original_name TEXT NOT NULL,
	mime_type TEXT NOT NULL,
	extension TEXT NOT NULL,
	size BIGINT NOT NULL DEFAULT 0,
	checksum TEXT NOT NULL DEFAULT '',
	metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	created_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
	updated_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
	UNIQUE (disk, path)
);

CREATE INDEX IF NOT EXISTS idx_files_folder_id ON files(folder_id);
CREATE INDEX IF NOT EXISTS idx_files_disk ON files(disk);

CREATE TABLE IF NOT EXISTS media (
	id BIGSERIAL PRIMARY KEY,
	file_id BIGINT NOT NULL REFERENCES files(id) ON DELETE RESTRICT,
	title TEXT NOT NULL DEFAULT '',
	alt TEXT NOT NULL DEFAULT '',
	caption TEXT NOT NULL DEFAULT '',
	is_transformed BOOLEAN NOT NULL DEFAULT false,
	metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	created_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
	updated_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_media_file_id ON media(file_id);

DO $$
BEGIN
	IF NOT EXISTS (
		SELECT 1 FROM pg_constraint WHERE conname = 'users_avatar_media_id_fkey'
	) THEN
		ALTER TABLE users
			ADD CONSTRAINT users_avatar_media_id_fkey
			FOREIGN KEY (avatar_media_id) REFERENCES media(id) ON DELETE SET NULL;
	END IF;
END
$$;

DO $$
BEGIN
	IF NOT EXISTS (
		SELECT 1 FROM pg_constraint WHERE conname = 'sites_created_by_fkey'
	) THEN
		ALTER TABLE sites
			ADD CONSTRAINT sites_created_by_fkey
			FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL;
	END IF;

	IF NOT EXISTS (
		SELECT 1 FROM pg_constraint WHERE conname = 'sites_updated_by_fkey'
	) THEN
		ALTER TABLE sites
			ADD CONSTRAINT sites_updated_by_fkey
			FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL;
	END IF;
END
$$;

CREATE TABLE IF NOT EXISTS resources (
	id BIGSERIAL PRIMARY KEY,
	site_id BIGINT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
	parent_id BIGINT NULL REFERENCES resources(id) ON DELETE RESTRICT,
	type TEXT NOT NULL,
	template TEXT NOT NULL,
	title TEXT NOT NULL,
	alias TEXT NOT NULL,
	path TEXT NOT NULL,
	sort INTEGER NOT NULL DEFAULT 0,
	is_published BOOLEAN NOT NULL DEFAULT false,
	published_at TIMESTAMPTZ NULL,
	settings JSONB NOT NULL DEFAULT '{}'::jsonb,
	seo JSONB NOT NULL DEFAULT '{}'::jsonb,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	created_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
	updated_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
	UNIQUE (site_id, path)
);

CREATE INDEX IF NOT EXISTS idx_resources_site_id ON resources(site_id);
CREATE INDEX IF NOT EXISTS idx_resources_parent_id ON resources(parent_id);
CREATE INDEX IF NOT EXISTS idx_resources_type ON resources(type);
CREATE INDEX IF NOT EXISTS idx_resources_template ON resources(template);
CREATE INDEX IF NOT EXISTS idx_resources_is_published ON resources(is_published);

CREATE TABLE IF NOT EXISTS resource_field_values (
	id BIGSERIAL PRIMARY KEY,
	resource_id BIGINT NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
	field TEXT NOT NULL,
	value JSONB NOT NULL DEFAULT 'null'::jsonb,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	UNIQUE (resource_id, field)
);

CREATE INDEX IF NOT EXISTS idx_resource_field_values_resource_id ON resource_field_values(resource_id);

CREATE TABLE IF NOT EXISTS resource_permissions (
	resource_id BIGINT NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
	group_code TEXT NOT NULL,
	permission TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	created_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
	updated_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
	UNIQUE (resource_id, group_code, permission)
);

CREATE INDEX IF NOT EXISTS idx_resource_permissions_group_code ON resource_permissions(group_code);
CREATE INDEX IF NOT EXISTS idx_resource_permissions_permission ON resource_permissions(permission);

CREATE TABLE IF NOT EXISTS redirects (
	id BIGSERIAL PRIMARY KEY,
	site_id BIGINT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
	from_path TEXT NOT NULL,
	to_path TEXT NOT NULL,
	status_code INTEGER NOT NULL CHECK (status_code IN (301, 302, 303, 307, 308)),
	is_active BOOLEAN NOT NULL DEFAULT true,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	created_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
	updated_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
	UNIQUE (site_id, from_path)
);

CREATE INDEX IF NOT EXISTS idx_redirects_site_id ON redirects(site_id);
CREATE INDEX IF NOT EXISTS idx_redirects_is_active ON redirects(is_active);

CREATE TABLE IF NOT EXISTS resource_widgets (
	id BIGSERIAL PRIMARY KEY,
	resource_id BIGINT NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
	widget TEXT NOT NULL,
	template TEXT NOT NULL,
	params JSONB NOT NULL DEFAULT '{}'::jsonb,
	sort INTEGER NOT NULL DEFAULT 0,
	area TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	created_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
	updated_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_resource_widgets_resource_id ON resource_widgets(resource_id);
CREATE INDEX IF NOT EXISTS idx_resource_widgets_widget ON resource_widgets(widget);
CREATE INDEX IF NOT EXISTS idx_resource_widgets_area ON resource_widgets(area);
CREATE INDEX IF NOT EXISTS idx_resource_widgets_sort ON resource_widgets(sort);

CREATE TABLE IF NOT EXISTS template_widgets (
	id BIGSERIAL PRIMARY KEY,
	resource_template TEXT NOT NULL,
	widget TEXT NOT NULL,
	template TEXT NOT NULL,
	params JSONB NOT NULL DEFAULT '{}'::jsonb,
	sort INTEGER NOT NULL DEFAULT 0,
	area TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	created_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
	updated_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_template_widgets_resource_template ON template_widgets(resource_template);
CREATE INDEX IF NOT EXISTS idx_template_widgets_widget ON template_widgets(widget);
CREATE INDEX IF NOT EXISTS idx_template_widgets_area ON template_widgets(area);
CREATE INDEX IF NOT EXISTS idx_template_widgets_sort ON template_widgets(sort);

COMMIT;
