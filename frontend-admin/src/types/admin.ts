export interface Pagination {
  page: number
  per_page: number
  total: number
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

export interface FieldChoice {
  value: string
  label: string
}

export interface FieldOptions {
  step?: number
  choices?: FieldChoice[]
  multiple?: boolean
  pattern?: string
}

export interface FieldDefinition {
  key: string
  type: FieldType | string
  label: string
  required: boolean
  rules: string[]
  options?: FieldOptions
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

export interface ResourceTreeItem {
  id: number
  parent_id: number | null
  template_code: string | null
  icon: string
  title: string
  menu_title: string
  display_title: string
  has_children: boolean
  can_create_child: boolean
}

export interface ResourceChildrenResponse {
  items: ResourceTreeItem[]
  permissions: { create_root: boolean }
}

export interface ResourceMetadata {
  types: Array<{ code: 'page' | 'link'; label: string }>
  templates: Array<{
    code: string
    label: string
    icon: string
    fields: FieldDefinition[]
  }>
}

export interface ResourceCreatePayload {
  parent_id: number | null
  type: 'page' | 'link'
  template_code?: string
  title: string
  menu_title: string
  slug: string
  external_url?: string
  settings: Record<string, unknown>
}

export interface Resource {
  id: number
  site_id: number
  parent_id: number | null
  type: 'page' | 'link'
  template_code: string | null
  title: string
  menu_title: string
  slug: string
  path: string | null
  content: string
  external_url: string | null
  is_public: boolean
  is_searchable: boolean
  in_menu: boolean
  in_sitemap: boolean
  sort: number
  settings: Record<string, unknown>
}

export interface ResourceDetailsResponse {
  resource: Resource
  permissions: { update: boolean }
}

export interface ResourceOption {
  id: number
  parent_id: number | null
  display_title: string
  path: string | null
}

export interface ResourceOptionsResponse {
  items: ResourceOption[]
}

export interface ResourceUpdatePayload {
  parent_id: number | null
  type: 'page' | 'link'
  template_code: string | null
  title: string
  menu_title: string
  slug: string
  content: string
  external_url: string | null
  is_public: boolean
  is_searchable: boolean
  in_menu: boolean
  in_sitemap: boolean
  sort: number
  settings: Record<string, unknown>
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
