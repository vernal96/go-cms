import type { FieldDefinition, Pagination } from '../../types/admin'

export type MailContentType = 'text' | 'html'
export type MailAttachmentSource = 'static' | 'variable' | 'site'
export type MailMessageStatus = 'queued' | 'sending' | 'retryable' | 'accepted' | 'failed'

export interface MailAddressTemplate {
  name: string
  email: string
}

export interface MailAttachmentTemplate {
  source: MailAttachmentSource
  file_id?: number
  variable?: string
  filename_template: string
}

export interface MailTemplate {
  id: number
  site_id: number
  code: string
  name: string
  enabled: boolean
  transport: string
  from: MailAddressTemplate
  to: MailAddressTemplate[]
  cc: MailAddressTemplate[]
  bcc: MailAddressTemplate[]
  reply_to?: MailAddressTemplate | null
  subject: string
  content_type: MailContentType
  text_body: string
  html_body: string
  attachments: MailAttachmentTemplate[]
  variables: FieldDefinition[]
  created_at: string
  updated_at: string
}

export type MailTemplatePayload = Omit<
  MailTemplate,
  'id' | 'site_id' | 'created_at' | 'updated_at'
>

export interface MailTemplateListResponse {
  items: MailTemplate[]
  pagination: Pagination
}

export interface MailAddress {
  name: string
  email: string
}

export interface MailAttachment {
  source: MailAttachmentSource | 'transient'
  file_id?: number
  filename: string
  mime_type: string
  size: number
  checksum_sha256: string
}

export interface MailSiteVariable {
  variable: string
  label: string
  type: FieldDefinition['type']
  source: 'site'
}

export interface MailSiteVariablesResponse { items: MailSiteVariable[] }

export interface MailWarning {
  field: string
  variable: string
  message: string
}

export interface RenderedMailMessage {
  from: MailAddress
  to: MailAddress[]
  cc: MailAddress[]
  bcc: MailAddress[]
  reply_to?: MailAddress | null
  subject: string
  content_type: MailContentType
  text_body: string
  html_body: string
  attachments: MailAttachment[]
  warnings: MailWarning[]
}

export interface MailDeliveryAttempt {
  id: number
  message_id: number
  attempt_number: number
  transport: string
  driver: string
  started_at: string
  finished_at?: string | null
  status: 'sending' | 'accepted' | 'failed'
  remote_message_id: string
  response_code: string
  safe_error: string
  created_at: string
}

export interface MailMessage {
  id: number
  site_id: number
  template_id?: number | null
  template_code: string
  template_name: string
  transport: string
  rfc_message_id: string
  from: MailAddress
  to: MailAddress[]
  cc: MailAddress[]
  bcc: MailAddress[]
  reply_to?: MailAddress | null
  subject: string
  content_type: MailContentType
  text_body: string
  html_body: string
  attachments: MailAttachment[]
  status: MailMessageStatus
  origin: 'manual' | 'automatic'
  origin_source: string
  origin_event: string
  origin_reference: string
  requested_at: string
  requested_by?: number | null
  requested_by_name: string
  accepted_at?: string | null
  created_at: string
  updated_at: string
  attempt_count: number
  latest_attempt?: MailDeliveryAttempt | null
}

export interface MailMessageListResponse {
  items: MailMessageSummary[]
  pagination: Pagination
}

export interface MailMessageSummary {
  id: number
  template_code: string
  template_name: string
  subject: string
  recipients: string[]
  status: MailMessageStatus
  origin: 'manual' | 'automatic'
  origin_source: string
  origin_event: string
  origin_reference: string
  requested_at: string
  requested_by?: number | null
  requested_by_name: string
  accepted_at?: string | null
  attempt_count: number
  latest_attempt?: MailDeliveryAttempt | null
}

export interface MailMessageFilters {
  status?: MailMessageStatus | ''
  template_code?: string
  date_from?: string
  date_to?: string
  recipient?: string
}

export interface MailMessageDetailResponse {
  message: MailMessage
  attempts: MailDeliveryAttempt[]
}
