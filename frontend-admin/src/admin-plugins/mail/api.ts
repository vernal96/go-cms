import { adminRequest, adminRequestVoid } from '../../api/admin-api'
import type {
  MailMessage,
  MailMessageDetailResponse,
  MailMessageListResponse,
  MailTemplate,
  MailTemplateListResponse,
  MailTemplatePayload,
  RenderedMailMessage,
} from './types'

function root(siteID: number): string {
  return `/api/sites/${encodeURIComponent(String(siteID))}/mail`
}

export function listMailTemplates(
  accessToken: string,
  siteID: number,
  page = 1,
  perPage = 20,
): Promise<MailTemplateListResponse> {
  return adminRequest(`${root(siteID)}/templates?page=${page}&per_page=${perPage}`, accessToken)
}

export function listSendTemplates(
  accessToken: string,
  siteID: number,
): Promise<MailTemplateListResponse> {
  return adminRequest(`${root(siteID)}/send/templates?page=1&per_page=100`, accessToken)
}

export function getMailTemplate(
  accessToken: string,
  siteID: number,
  templateID: number,
): Promise<MailTemplate> {
  return adminRequest(`${root(siteID)}/templates/${templateID}`, accessToken)
}

export function createMailTemplate(
  accessToken: string,
  siteID: number,
  payload: MailTemplatePayload,
): Promise<MailTemplate> {
  return adminRequest(`${root(siteID)}/templates`, accessToken, {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export function updateMailTemplate(
  accessToken: string,
  siteID: number,
  templateID: number,
  payload: MailTemplatePayload,
): Promise<MailTemplate> {
  return adminRequest(`${root(siteID)}/templates/${templateID}`, accessToken, {
    method: 'PATCH',
    body: JSON.stringify(payload),
  })
}

export function deleteMailTemplate(
  accessToken: string,
  siteID: number,
  templateID: number,
): Promise<void> {
  return adminRequestVoid(`${root(siteID)}/templates/${templateID}`, accessToken, { method: 'DELETE' })
}

export function previewMail(
  accessToken: string,
  siteID: number,
  templateID: number,
  values: Record<string, unknown>,
): Promise<RenderedMailMessage> {
  return adminRequest(`${root(siteID)}/preview`, accessToken, {
    method: 'POST',
    body: JSON.stringify({ template_id: templateID, values }),
  })
}

export function queueMail(
  accessToken: string,
  siteID: number,
  templateID: number,
  values: Record<string, unknown>,
): Promise<MailMessage> {
  return adminRequest(`${root(siteID)}/send`, accessToken, {
    method: 'POST',
    body: JSON.stringify({ template_id: templateID, values }),
  })
}

export function listMailMessages(
  accessToken: string,
  siteID: number,
  page = 1,
  perPage = 20,
): Promise<MailMessageListResponse> {
  return adminRequest(`${root(siteID)}/messages?page=${page}&per_page=${perPage}`, accessToken)
}

export function getMailMessage(
  accessToken: string,
  siteID: number,
  messageID: number,
): Promise<MailMessageDetailResponse> {
  return adminRequest(`${root(siteID)}/messages/${messageID}`, accessToken)
}

export function deleteMailMessage(
  accessToken: string,
  siteID: number,
  messageID: number,
): Promise<void> {
  return adminRequestVoid(`${root(siteID)}/messages/${messageID}`, accessToken, { method: 'DELETE' })
}
