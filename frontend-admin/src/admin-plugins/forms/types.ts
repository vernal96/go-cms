import type { Pagination } from '../../types/admin'

export type FormsFieldType =
  | 'string' | 'integer' | 'float' | 'checkbox' | 'radio' | 'select'
  | 'textarea' | 'email' | 'phone' | 'json'
  | 'forms.captcha' | 'forms.consent' | 'forms.upload'

export interface FormsChoice { value: string; label: string }
export interface FormsFieldOptions {
  step?: number
  choices?: FormsChoice[]
  multiple?: boolean
  pattern?: string
  mime_types?: string[]
  max_file_size?: number
  max_files?: number
  provider?: string
  text?: string
  url?: string
}
export interface FormsVisibleWhen { field: string; value: unknown }
export interface FormField {
  id: number
  form_id: number
  code: string
  type: FormsFieldType
  label: string
  required: boolean
  rules: string[]
  options?: FormsFieldOptions
  editor?: string
  visible_when?: FormsVisibleWhen
  result_label: string
  show_in_results: boolean
  result_position: number
  created_at: string
  updated_at: string
}
export type FormFieldPayload = Omit<FormField, 'id' | 'form_id' | 'created_at' | 'updated_at'>

export interface FormRecord {
  id: number
  site_id: number
  code: string
  name: string
  description: string
  enabled: boolean
  created_at: string
  updated_at: string
}
export type FormPayload = Pick<FormRecord, 'code' | 'name' | 'description' | 'enabled'>

export type ElementType = 'text' | 'heading' | 'image' | 'submit_button'
export interface FormElement {
  id: number
  form_id: number
  code: string
  type: ElementType
  config: Record<string, unknown>
  created_at: string
  updated_at: string
}
export interface ElementTypeMetadata {
  code: ElementType
  label: string
  fields: ActionConfigField[]
}

export type LayoutKind = 'field' | 'element' | 'container'
export type ContainerType = 'group' | 'slide'
export interface LayoutNode {
  id: number
  form_id: number
  parent_id?: number | null
  kind: LayoutKind
  field_id?: number | null
  element_id?: number | null
  container_type?: ContainerType | ''
  position: number
  config?: Record<string, unknown>
}

export interface FormStatus {
  id: number
  form_id: number
  code: string
  name: string
  color: string
  position: number
  is_default: boolean
  created_at: string
  updated_at: string
}

export type TriggerType = 'submitted' | 'status_changed'
export interface FormTrigger { type: TriggerType; from_status?: string; to_status?: string }
export interface FormAction {
  id: number
  form_id: number
  code: string
  name: string
  enabled: boolean
  trigger: FormTrigger
  action_type: string
  config: Record<string, unknown>
  position: number
  created_at: string
  updated_at: string
}
export interface ActionConfigField { key: string; label: string; type: string; required: boolean }
export interface ActionTypeMetadata {
  code: string
  label: string
  available: boolean
  editor_code?: string
  fields?: ActionConfigField[]
}

export interface FormEditorResponse {
  form: FormRecord
  fields: FormField[]
  elements: FormElement[]
  layout: LayoutNode[]
  statuses: FormStatus[]
  actions: FormAction[]
  available_field_types: FormsFieldType[]
  available_element_types: ElementTypeMetadata[]
  available_action_types: ActionTypeMetadata[]
}
export interface FormsListResponse { items: FormRecord[]; pagination: Pagination }

export interface ResultValue {
  id: number
  field_code: string
  field_label: string
  result_label: string
  field_type: FormsFieldType
  position: number
  value: unknown
}
export interface ResultUpload {
  id: number
  field_code: string
  position: number
  filename: string
  mime_type: string
  size: number
  checksum_sha256: string
  spool_deleted_at?: string | null
}
export type ExecutionStatus = 'pending' | 'running' | 'retryable' | 'succeeded' | 'failed'
export interface ActionExecution {
  id: number
  action_code: string
  action_name: string
  action_type: string
  trigger: FormTrigger
  status: ExecutionStatus
  attempt_count: number
  safe_error: string
  external_reference: string
  started_at?: string | null
  finished_at?: string | null
  created_at: string
}
export interface FormResult {
  id: number
  site_id: number
  form_id: number
  form_code: string
  form_name: string
  status_id: number
  status_code: string
  status_name: string
  status_color: string
  user_id?: number | null
  user_agent: string
  client_address?: string
  created_at: string
  updated_at: string
}
export interface FormResultSummary extends FormResult { values: Record<string, unknown> }
export interface ResultsResponse {
  items: FormResultSummary[]
  columns: FormField[]
  pagination: Pagination
}
export interface ResultDetailResponse {
  result: FormResult
  values: ResultValue[]
  uploads: ResultUpload[]
  action_executions: ActionExecution[]
  available_statuses?: FormStatus[]
}
