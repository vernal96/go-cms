CREATE TABLE IF NOT EXISTS users (
	id BIGINT NOT NULL AUTO_INCREMENT,
	email VARCHAR(255) NOT NULL,
	password_hash TEXT NOT NULL,
	name VARCHAR(255) NOT NULL,
	avatar_media_id BIGINT NULL,
	settings JSON NULL,
	created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	created_by BIGINT NULL,
	updated_by BIGINT NULL,
	PRIMARY KEY (id),
	UNIQUE KEY uq_users_email (email),
	KEY idx_users_avatar_media_id (avatar_media_id),
	CONSTRAINT fk_users_created_by
		FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
	CONSTRAINT fk_users_updated_by
		FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB
DEFAULT CHARSET=utf8mb4
COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS sites (
	id BIGINT NOT NULL AUTO_INCREMENT,
	profile_code VARCHAR(128) NOT NULL,
	domain VARCHAR(255) NOT NULL,
	locale VARCHAR(32) NOT NULL DEFAULT 'ru',
	settings JSON NOT NULL,
	created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	created_by BIGINT NULL,
	updated_by BIGINT NULL,
	PRIMARY KEY (id),
	UNIQUE KEY uq_sites_domain (domain),
	KEY idx_sites_profile_code (profile_code),
	CONSTRAINT fk_sites_created_by
		FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
	CONSTRAINT fk_sites_updated_by
		FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB
DEFAULT CHARSET=utf8mb4
COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS user_groups (
	user_id BIGINT NOT NULL,
	group_code VARCHAR(128) NOT NULL,
	created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	created_by BIGINT NULL,
	updated_by BIGINT NULL,
	PRIMARY KEY (user_id, group_code),
	KEY idx_user_groups_group_code (group_code),
	CONSTRAINT fk_user_groups_user
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
	CONSTRAINT fk_user_groups_created_by
		FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
	CONSTRAINT fk_user_groups_updated_by
		FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB
DEFAULT CHARSET=utf8mb4
COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS site_permissions (
	site_id BIGINT NOT NULL,
	group_code VARCHAR(128) NOT NULL,
	permission VARCHAR(128) NOT NULL,
	created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	created_by BIGINT NULL,
	updated_by BIGINT NULL,
	UNIQUE KEY uq_site_permissions_site_group_permission (
		site_id,
		group_code,
		permission
	),
	KEY idx_site_permissions_group_code (group_code),
	KEY idx_site_permissions_permission (permission),
	CONSTRAINT fk_site_permissions_site
		FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
	CONSTRAINT fk_site_permissions_created_by
		FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
	CONSTRAINT fk_site_permissions_updated_by
		FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB
DEFAULT CHARSET=utf8mb4
COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS file_folders (
	id BIGINT NOT NULL AUTO_INCREMENT,
	parent_id BIGINT NULL,
	disk VARCHAR(128) NOT NULL,
	name VARCHAR(255) NOT NULL,
	sort INTEGER NOT NULL DEFAULT 0,
	created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	created_by BIGINT NULL,
	updated_by BIGINT NULL,
	PRIMARY KEY (id),
	UNIQUE KEY uq_file_folders_parent_disk_name (parent_id, disk, name),
	KEY idx_file_folders_parent_id (parent_id),
	KEY idx_file_folders_disk (disk),
	CONSTRAINT fk_file_folders_parent
		FOREIGN KEY (parent_id) REFERENCES file_folders(id) ON DELETE RESTRICT,
	CONSTRAINT fk_file_folders_created_by
		FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
	CONSTRAINT fk_file_folders_updated_by
		FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB
DEFAULT CHARSET=utf8mb4
COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS files (
	id BIGINT NOT NULL AUTO_INCREMENT,
	folder_id BIGINT NULL,
	disk VARCHAR(128) NOT NULL,
	path VARCHAR(512) NOT NULL,
	name VARCHAR(255) NOT NULL,
	original_name VARCHAR(255) NOT NULL,
	mime_type VARCHAR(255) NOT NULL,
	extension VARCHAR(64) NOT NULL,
	size BIGINT NOT NULL DEFAULT 0,
	checksum VARCHAR(255) NOT NULL DEFAULT '',
	metadata JSON NULL,
	created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	created_by BIGINT NULL,
	updated_by BIGINT NULL,
	PRIMARY KEY (id),
	UNIQUE KEY uq_files_disk_path (disk, path),
	KEY idx_files_folder_id (folder_id),
	KEY idx_files_disk (disk),
	CONSTRAINT fk_files_folder
		FOREIGN KEY (folder_id) REFERENCES file_folders(id) ON DELETE SET NULL,
	CONSTRAINT fk_files_created_by
		FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
	CONSTRAINT fk_files_updated_by
		FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB
DEFAULT CHARSET=utf8mb4
COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS media (
	id BIGINT NOT NULL AUTO_INCREMENT,
	file_id BIGINT NOT NULL,
	title TEXT NOT NULL,
	alt TEXT NOT NULL,
	caption TEXT NOT NULL,
	is_transformed BOOLEAN NOT NULL DEFAULT FALSE,
	metadata JSON NULL,
	created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	created_by BIGINT NULL,
	updated_by BIGINT NULL,
	PRIMARY KEY (id),
	KEY idx_media_file_id (file_id),
	CONSTRAINT fk_media_file
		FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE RESTRICT,
	CONSTRAINT fk_media_created_by
		FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
	CONSTRAINT fk_media_updated_by
		FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB
DEFAULT CHARSET=utf8mb4
COLLATE=utf8mb4_unicode_ci;

SET @users_avatar_media_fk_exists = (
	SELECT COUNT(*)
	FROM information_schema.TABLE_CONSTRAINTS
	WHERE CONSTRAINT_SCHEMA = DATABASE()
		AND TABLE_NAME = 'users'
		AND CONSTRAINT_NAME = 'fk_users_avatar_media'
		AND CONSTRAINT_TYPE = 'FOREIGN KEY'
);
SET @users_avatar_media_fk_sql = IF(
	@users_avatar_media_fk_exists = 0,
	'ALTER TABLE users ADD CONSTRAINT fk_users_avatar_media FOREIGN KEY (avatar_media_id) REFERENCES media(id) ON DELETE SET NULL',
	'DO 0'
);
PREPARE users_avatar_media_fk_statement FROM @users_avatar_media_fk_sql;
EXECUTE users_avatar_media_fk_statement;
DEALLOCATE PREPARE users_avatar_media_fk_statement;

CREATE TABLE IF NOT EXISTS resources (
	id BIGINT NOT NULL AUTO_INCREMENT,
	site_id BIGINT NOT NULL,
	parent_id BIGINT NULL,
	type VARCHAR(128) NOT NULL,
	template VARCHAR(128) NOT NULL,
	title TEXT NOT NULL,
	alias VARCHAR(255) NOT NULL,
	path VARCHAR(512) NOT NULL,
	sort INTEGER NOT NULL DEFAULT 0,
	is_published BOOLEAN NOT NULL DEFAULT FALSE,
	published_at DATETIME(6) NULL,
	settings JSON NOT NULL,
	seo JSON NOT NULL,
	created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	created_by BIGINT NULL,
	updated_by BIGINT NULL,
	PRIMARY KEY (id),
	UNIQUE KEY uq_resources_site_path (site_id, path),
	KEY idx_resources_site_id (site_id),
	KEY idx_resources_parent_id (parent_id),
	KEY idx_resources_type (type),
	KEY idx_resources_template (template),
	KEY idx_resources_is_published (is_published),
	CONSTRAINT fk_resources_site
		FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
	CONSTRAINT fk_resources_parent
		FOREIGN KEY (parent_id) REFERENCES resources(id) ON DELETE RESTRICT,
	CONSTRAINT fk_resources_created_by
		FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
	CONSTRAINT fk_resources_updated_by
		FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB
DEFAULT CHARSET=utf8mb4
COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS resource_field_values (
	id BIGINT NOT NULL AUTO_INCREMENT,
	resource_id BIGINT NOT NULL,
	field VARCHAR(128) NOT NULL,
	value JSON NOT NULL,
	created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	PRIMARY KEY (id),
	UNIQUE KEY uq_resource_field_values_resource_field (resource_id, field),
	KEY idx_resource_field_values_resource_id (resource_id),
	CONSTRAINT fk_resource_field_values_resource
		FOREIGN KEY (resource_id) REFERENCES resources(id) ON DELETE CASCADE
) ENGINE=InnoDB
DEFAULT CHARSET=utf8mb4
COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS resource_permissions (
	resource_id BIGINT NOT NULL,
	group_code VARCHAR(128) NOT NULL,
	permission VARCHAR(128) NOT NULL,
	created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	created_by BIGINT NULL,
	updated_by BIGINT NULL,
	UNIQUE KEY uq_resource_permissions_resource_group_permission (
		resource_id,
		group_code,
		permission
	),
	KEY idx_resource_permissions_group_code (group_code),
	KEY idx_resource_permissions_permission (permission),
	CONSTRAINT fk_resource_permissions_resource
		FOREIGN KEY (resource_id) REFERENCES resources(id) ON DELETE CASCADE,
	CONSTRAINT fk_resource_permissions_created_by
		FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
	CONSTRAINT fk_resource_permissions_updated_by
		FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB
DEFAULT CHARSET=utf8mb4
COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS redirects (
	id BIGINT NOT NULL AUTO_INCREMENT,
	site_id BIGINT NOT NULL,
	from_path VARCHAR(512) NOT NULL,
	to_path TEXT NOT NULL,
	status_code INTEGER NOT NULL,
	is_active BOOLEAN NOT NULL DEFAULT TRUE,
	created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	created_by BIGINT NULL,
	updated_by BIGINT NULL,
	PRIMARY KEY (id),
	UNIQUE KEY uq_redirects_site_from_path (site_id, from_path),
	KEY idx_redirects_site_id (site_id),
	KEY idx_redirects_is_active (is_active),
	CONSTRAINT chk_redirects_status_code
		CHECK (status_code IN (301, 302, 303, 307, 308)),
	CONSTRAINT fk_redirects_site
		FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
	CONSTRAINT fk_redirects_created_by
		FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
	CONSTRAINT fk_redirects_updated_by
		FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB
DEFAULT CHARSET=utf8mb4
COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS resource_widgets (
	id BIGINT NOT NULL AUTO_INCREMENT,
	resource_id BIGINT NOT NULL,
	widget VARCHAR(255) NOT NULL,
	template VARCHAR(128) NOT NULL,
	params JSON NULL,
	sort INTEGER NOT NULL DEFAULT 0,
	area VARCHAR(128) NOT NULL,
	created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	created_by BIGINT NULL,
	updated_by BIGINT NULL,
	PRIMARY KEY (id),
	KEY idx_resource_widgets_resource_id (resource_id),
	KEY idx_resource_widgets_widget (widget),
	KEY idx_resource_widgets_area (area),
	KEY idx_resource_widgets_sort (sort),
	CONSTRAINT fk_resource_widgets_resource
		FOREIGN KEY (resource_id) REFERENCES resources(id) ON DELETE CASCADE,
	CONSTRAINT fk_resource_widgets_created_by
		FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
	CONSTRAINT fk_resource_widgets_updated_by
		FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB
DEFAULT CHARSET=utf8mb4
COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS template_widgets (
	id BIGINT NOT NULL AUTO_INCREMENT,
	resource_template VARCHAR(128) NOT NULL,
	widget VARCHAR(255) NOT NULL,
	template VARCHAR(128) NOT NULL,
	params JSON NULL,
	sort INTEGER NOT NULL DEFAULT 0,
	area VARCHAR(128) NOT NULL,
	created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	created_by BIGINT NULL,
	updated_by BIGINT NULL,
	PRIMARY KEY (id),
	KEY idx_template_widgets_resource_template (resource_template),
	KEY idx_template_widgets_widget (widget),
	KEY idx_template_widgets_area (area),
	KEY idx_template_widgets_sort (sort),
	CONSTRAINT fk_template_widgets_created_by
		FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
	CONSTRAINT fk_template_widgets_updated_by
		FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB
DEFAULT CHARSET=utf8mb4
COLLATE=utf8mb4_unicode_ci;
