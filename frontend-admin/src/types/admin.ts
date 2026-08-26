export interface Pagination {
  page: number
  per_page: number
  total: number
}

export type AdminNavigationScope = 'global' | 'site'

export interface AdminNavigationItem {
  code: string
  label: string
  route?: string
  icon?: string
  order: number
  scope: AdminNavigationScope
  children?: AdminNavigationItem[]
}

export interface AdminNavigationResponse {
  items: AdminNavigationItem[]
}

export interface PermissionSet {
  read: boolean
  create: boolean
  update: boolean
  delete: boolean
}

export interface Site {
  id: number
  profile_code: string
  domain: string
  locale: string
  settings: Record<string, unknown>
  is_public: boolean
	capabilities: SiteCapabilities
}

export interface SiteCapabilities {
	view: boolean
	edit: boolean
	delete: boolean
}

export interface SiteOption {
  id: number
  domain: string
}

export interface SiteListResponse {
  items: Site[]
  pagination: Pagination
  permissions: PermissionSet
}

export interface SiteOptionsResponse {
  items: SiteOption[]
  pagination: Pagination
}

export interface SiteDetailsResponse {
  site: Site
  permissions: PermissionSet
}

export interface SiteProfile {
  code: string
  name: string
  fields: FieldDefinition[]
}

export type FieldType =
  | 'string'
  | 'int'
  | 'float'
  | 'checkbox'
  | 'radio'
  | 'select'
  | 'textarea'
  | 'email'
  | 'phone'
  | 'file'

export interface FieldChoice {
  value: string
  label: string
}

export interface FieldOptions {
  step?: number
  choices?: FieldChoice[]
  multiple?: boolean
  pattern?: string
  storages?: string[]
  mime_types?: string[]
}

export type FilesystemItemKind = 'file' | 'folder'

export interface FilesystemDisk {
  code: string
  visibility: 'public' | 'private'
}

export interface FilesystemItem {
  kind: FilesystemItemKind
  id: number
  parent_id: number | null
  storage: string
  name: string
  mime_type?: string
  size?: number
  item_count?: number
  created_at: string
  updated_at: string
}

export interface FilesystemDisksResponse {
  items: FilesystemDisk[]
  permissions: PermissionSet
}

export interface FilesystemListingResponse {
  disk: FilesystemDisk
  folder: FilesystemItem | null
  breadcrumbs: Array<{ id: number; name: string }>
  items: FilesystemItem[]
  permissions: PermissionSet
}

export interface ProfileAvatar {
  file_id: number
  name: string
  mime_type: string
  size: number
  updated_at: string
}

export interface ProfileUser {
  id: number
  login: string
  email: string
  name: string
  last_name: string | null
  middle_name: string | null
  phone: string | null
  color_scheme: import('./auth').ColorScheme
  accent_color: import('./auth').AccentColor
  avatar: ProfileAvatar | null
}

export interface ProfileResponse {
  user: ProfileUser
}

export interface FieldDefinition {
  key: string
  type: FieldType | string
  label: string
  required: boolean
  rules: string[]
  options?: FieldOptions
	editor?: string
	visible_when?: { field: string; value: unknown }
}

export interface SiteProfilesResponse {
  items: SiteProfile[]
}

export interface SiteFormPayload {
  profile_code: string
  domain: string
  locale: string
  is_public: boolean
  settings: Record<string, unknown>
}

export interface DashboardSite {
  id: number
  domain: string
  is_public: boolean
  resource_count?: number
}

export interface DashboardResponse {
  sites?: {
    total: number
    public: number
    private: number
    items: DashboardSite[]
  }
  resources?: {
    total: number
  }
  users?: {
    total: number
    active: number
    blocked: number
  }
  groups?: {
    total: number
  }
}

export interface ResourceTreeItem {
	id: number
	version: number
  parent_id: number | null
  template_code: string | null
  icon: string
  title: string
  menu_title: string
  display_title: string
  sort: number
  deleted: boolean
  published: boolean
  deleted_at: string | null
  has_children: boolean
  can_create_child: boolean
}

export interface ResourceChildrenResponse {
  items: ResourceTreeItem[]
  permissions: { create_root: boolean }
}

export interface ResourceMetadata {
  types: ResourceTypeMetadata[]
  templates: ResourceTemplate[]
  widgets: WidgetDefinition[]
  extensions: ResourceExtensionMetadata[]
}

export type ResourceTypeCode = string
export interface ResourceTypeCapabilities {
  supports_template?: boolean
  supports_content?: boolean
  supports_widgets?: boolean
  supports_fields?: boolean
  supports_external_url?: boolean
  supports_target_resource?: boolean
  mutable_type?: boolean
  owns_library_items?: boolean
  default_icon?: string
}
export interface ResourceContentTypeOption {
  code: string
  label: string
  editor: string
}
export interface ResourceTypeMetadata {
  code: ResourceTypeCode
  label: string
  capabilities: ResourceTypeCapabilities
  settings_fields: FieldDefinition[]
  settings_defaults: Record<string, unknown>
  content_types: ResourceContentTypeOption[]
}

export type WidgetArea = 'body' | 'sidebar'

export interface ResourceTemplate {
  code: string
  label: string
  icon: string
  fields: FieldDefinition[]
  supports_resource_widgets: boolean
  widget_areas: WidgetArea[]
}

export interface WidgetDefinition {
  code: string
  module_code: string
  module_label: string
  module_description: string
  label: string
  description: string
  fields: FieldDefinition[]
  editor_tabs: Array<{ code: string; label: string; fields: string[] }>
  summary_fields: string[]
  views: Array<{ code: string; label: string }>
}

export interface ResourceWidget {
  id: number
  code: string
  area: WidgetArea
  position: number
  view: string
  columns: number
  margin_top: number
  margin_bottom: number
  enabled: boolean
	params: Record<string, unknown>
	resource_version?: number
}

export interface ResourceExtensionMetadata {
  code: string
  title: string
  applies_to: ResourceTypeCode[]
  fields: Array<{
    key: string
    label: string
    control: 'text' | 'textarea' | 'switch'
    rows?: number
  }>
  variables: Array<{ code: string; label: string }>
}

export interface SEOSettings {
  title_template: string
  description_template: string
  keywords_template: string
  canonical_template: string
  robots_index: boolean
  robots_follow: boolean
  og_title_template: string
  og_description_template: string
}

export interface SEOPreview {
  title: string
  description: string
  keywords: string[]
  canonical_url: string
  robots: { index: boolean; follow: boolean }
  open_graph: { title: string; description: string }
  warnings: Array<{ field: string; variable: string; message: string }>
  title_characters: number
  description_characters: number
}

export interface ResourceCreatePayload {
  parent_id: number | null
  type: ResourceTypeCode
  template_code?: string | null
  content_type?: string | null
  content?: string
  target_resource_id?: number | null
  title: string
  menu_title: string
  slug: string
  external_url?: string
  fields: Record<string, unknown>
  type_settings: Record<string, unknown>
}

export interface Resource {
	id: number
	site_id: number
	version: number
  parent_id: number | null
  type: ResourceTypeCode
  template_code: string | null
  title: string
  menu_title: string
  slug: string
  path: string | null
  annotation: string
  content_type: string | null
  content: string
  target_resource_id: number | null
  external_url: string | null
  is_public: boolean
  is_searchable: boolean
  in_menu: boolean
  in_sitemap: boolean
  sort: number
  published_at: string | null
  unpublished_at: string | null
  deleted: boolean
  deleted_at: string | null
  fields: Record<string, unknown>
  type_settings: Record<string, unknown>
  widgets: ResourceWidget[]
}

export interface ResourceDetailsResponse {
	resource: Resource
	permissions: { update: boolean; delete: boolean; restore: boolean; history_read: boolean; history_delete: boolean }
}

export interface ResourceOption {
  id: number
  parent_id: number | null
  type: ResourceTypeCode
  display_title: string
  path: string | null
}

export interface ResourceOptionsResponse {
  items: ResourceOption[]
}

export interface ResourceLookupResponse {
  items: ResourceOption[]
  pagination: Pagination
}

export interface ResourceUpdatePayload {
	expected_version: number
  parent_id: number | null
  type: ResourceTypeCode
  template_code: string | null
  title: string
  menu_title: string
  slug: string
  annotation: string
  content_type: string | null
  content: string
  target_resource_id: number | null
  external_url: string | null
  is_public: boolean
  is_searchable: boolean
  in_menu: boolean
  in_sitemap: boolean
  sort: number
  published_at: string | null
  unpublished_at: string | null
  fields: Record<string, unknown>
  type_settings: Record<string, unknown>
}

export interface ResourceRevision {
	id: number
	resource_id: number
	site_id: number
	version: number
	kind: 'created' | 'updated' | 'restored'
	source_version?: number
	created_at: string
	created_by?: number
	created_by_name: string
	snapshot?: Record<string, unknown>
}
export interface ResourceRevisionPage { items: ResourceRevision[]; page: number; per_page: number; total: number }

export interface LibraryItem {
	id: number
	version: number
  site_id: number
  library_id: number
  template_code: string | null
  title: string
  slug: string
  annotation: string
  content_type: string | null
  content: string
  is_public: boolean
  is_searchable: boolean
  published_at: string | null
  unpublished_at: string | null
  deleted: boolean
  deleted_at: string | null
  fields: Record<string, unknown>
  widgets: ResourceWidget[]
  effective_url: string
}
export interface LibraryItemDetailsResponse { item: LibraryItem; permissions: { update: boolean; delete: boolean; restore: boolean; history_read: boolean; history_delete: boolean } }
export interface LibraryItemsResponse { items: LibraryItem[]; next_cursor: string }
export interface LibraryItemPayload {
	expected_version?: number
  template_code: string | null
  title: string
  slug: string
  annotation: string
  content: string
  is_public: boolean
  is_searchable: boolean
  published_at: string | null
  unpublished_at: string | null
  fields: Record<string, unknown>
}

export interface AdminTableColumn {
  prop: string
  label: string
  width?: string | number
  formatter?: (row: Record<string, unknown>) => string
}

export interface AdminTableAction {
  key: string
  label: string
  danger?: boolean
  visible?: (row: Record<string, unknown>) => boolean
}

export interface UserCapabilities {
  update: boolean
  change_password: boolean
  edit_groups: boolean
  block: boolean
  unblock: boolean
}

export interface AdminUser {
  id: number
  login: string
  email: string
  name: string
  last_name: string | null
  middle_name: string | null
  phone: string | null
  last_login_at: string | null
  created_at: string
  updated_at: string
  blocked: boolean
  blocked_at: string | null
  capabilities: UserCapabilities
}

export interface UserListResponse {
  items: AdminUser[]
  pagination: Pagination
  permissions: { read: boolean; create: boolean; update: boolean; block: boolean }
}

export interface UserDetailsResponse {
  user: AdminUser
}

export interface UserInfoPayload {
  login: string
  email: string
  name: string
  last_name: string | null
  middle_name: string | null
  phone: string | null
}

export interface AdminGroup {
  id: number
  code: string
  name: string
  system: boolean
  super: boolean
  can_update: boolean
  can_delete: boolean
  can_manage_permissions: boolean
}

export interface GroupListResponse {
  items: AdminGroup[]
  pagination: Pagination
  permissions: PermissionSet
}

export interface GroupOptionsResponse {
  items: AdminGroup[]
  pagination: Pagination
}

export interface GroupDetailsResponse {
  group: AdminGroup
  permission_codes: string[]
	site_access: GroupSiteAccess[]
}

export interface GroupSiteAccess {
	site_id: number
	can_view: boolean
	can_edit: boolean
	can_delete: boolean
}

export interface PermissionDefinition {
  code: string
  module: string
  entity: string
  action: 'read' | 'create' | 'update' | 'delete'
}

export interface PermissionCatalogResponse {
  items: PermissionDefinition[]
  can_manage: boolean
}
