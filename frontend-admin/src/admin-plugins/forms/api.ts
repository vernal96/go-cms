import { adminRequest, adminRequestVoid } from '../../api/admin-api'
import type {
  FormAction, FormEditorResponse, FormElement, FormField, FormFieldPayload, FormPayload,
  FormRecord, FormStatus, FormsListResponse, LayoutNode, ResultDetailResponse,
  ResultsResponse,
} from './types'

function root(siteID: number): string { return `/api/sites/${encodeURIComponent(String(siteID))}/forms` }
function formsRoot(siteID: number): string { return `${root(siteID)}/forms` }

export function listForms(token: string, siteID: number, page = 1, perPage = 20, search = ''): Promise<FormsListResponse> {
  const query = new URLSearchParams({ page: String(page), per_page: String(perPage) })
  if (search.trim()) query.set('search', search.trim())
  return adminRequest(`${formsRoot(siteID)}?${query}`, token)
}
export function createForm(token: string, siteID: number, payload: FormPayload): Promise<FormRecord> {
  return adminRequest(formsRoot(siteID), token, { method: 'POST', body: JSON.stringify(payload) })
}
export function updateForm(token: string, siteID: number, formID: number, payload: FormPayload): Promise<FormRecord> {
  return adminRequest(`${formsRoot(siteID)}/${formID}`, token, { method: 'PATCH', body: JSON.stringify(payload) })
}
export function setFormEnabled(token: string, siteID: number, formID: number, enabled: boolean): Promise<FormRecord> {
  return adminRequest(`${formsRoot(siteID)}/${formID}/enabled`, token, { method: 'PATCH', body: JSON.stringify({ enabled }) })
}
export function deleteForm(token: string, siteID: number, formID: number): Promise<void> {
  return adminRequestVoid(`${formsRoot(siteID)}/${formID}`, token, { method: 'DELETE' })
}
export function getFormEditor(token: string, siteID: number, formID: number): Promise<FormEditorResponse> {
  return adminRequest(`${formsRoot(siteID)}/${formID}/editor`, token)
}
export function createField(token: string, siteID: number, formID: number, payload: FormFieldPayload): Promise<{ field: FormField; layout_node: LayoutNode }> {
  return adminRequest(`${formsRoot(siteID)}/${formID}/fields`, token, { method: 'POST', body: JSON.stringify(payload) })
}
export function updateField(token: string, siteID: number, formID: number, fieldID: number, payload: FormFieldPayload): Promise<FormField> {
  return adminRequest(`${formsRoot(siteID)}/${formID}/fields/${fieldID}`, token, { method: 'PATCH', body: JSON.stringify(payload) })
}
export function deleteField(token: string, siteID: number, formID: number, fieldID: number): Promise<void> {
  return adminRequestVoid(`${formsRoot(siteID)}/${formID}/fields/${fieldID}`, token, { method: 'DELETE' })
}
export function createElement(token: string, siteID: number, formID: number, payload: Pick<FormElement, 'code' | 'type' | 'config'>): Promise<{ element: FormElement; layout_node: LayoutNode }> {
  return adminRequest(`${formsRoot(siteID)}/${formID}/elements`, token, { method: 'POST', body: JSON.stringify(payload) })
}
export function updateElement(token: string, siteID: number, formID: number, elementID: number, payload: Pick<FormElement, 'code' | 'type' | 'config'>): Promise<FormElement> {
  return adminRequest(`${formsRoot(siteID)}/${formID}/elements/${elementID}`, token, { method: 'PATCH', body: JSON.stringify(payload) })
}
export function deleteElement(token: string, siteID: number, formID: number, elementID: number): Promise<void> {
  return adminRequestVoid(`${formsRoot(siteID)}/${formID}/elements/${elementID}`, token, { method: 'DELETE' })
}
export function createContainer(token: string, siteID: number, formID: number, payload: { parent_id?: number | null; container_type: 'group' | 'slide'; position: number; config: Record<string, unknown> }): Promise<LayoutNode> {
  return adminRequest(`${formsRoot(siteID)}/${formID}/containers`, token, { method: 'POST', body: JSON.stringify(payload) })
}
export function replaceLayout(token: string, siteID: number, formID: number, nodes: LayoutNode[]): Promise<{ nodes: LayoutNode[] }> {
  return adminRequest(`${formsRoot(siteID)}/${formID}/layout`, token, { method: 'PUT', body: JSON.stringify({ nodes }) })
}
type StatusPayload = Pick<FormStatus, 'code' | 'name' | 'color' | 'position' | 'is_default'>
export function createStatus(token: string, siteID: number, formID: number, payload: StatusPayload): Promise<FormStatus> {
  return adminRequest(`${formsRoot(siteID)}/${formID}/statuses`, token, { method: 'POST', body: JSON.stringify(payload) })
}
export function updateStatus(token: string, siteID: number, formID: number, statusID: number, payload: StatusPayload): Promise<FormStatus> {
  return adminRequest(`${formsRoot(siteID)}/${formID}/statuses/${statusID}`, token, { method: 'PATCH', body: JSON.stringify(payload) })
}
export function deleteStatus(token: string, siteID: number, formID: number, statusID: number): Promise<void> {
  return adminRequestVoid(`${formsRoot(siteID)}/${formID}/statuses/${statusID}`, token, { method: 'DELETE' })
}
type ActionPayload = Pick<FormAction, 'code' | 'name' | 'enabled' | 'trigger' | 'action_type' | 'config' | 'position'>
export function createAction(token: string, siteID: number, formID: number, payload: ActionPayload): Promise<FormAction> {
  return adminRequest(`${formsRoot(siteID)}/${formID}/actions`, token, { method: 'POST', body: JSON.stringify(payload) })
}
export function updateAction(token: string, siteID: number, formID: number, actionID: number, payload: ActionPayload): Promise<FormAction> {
  return adminRequest(`${formsRoot(siteID)}/${formID}/actions/${actionID}`, token, { method: 'PATCH', body: JSON.stringify(payload) })
}
export function deleteAction(token: string, siteID: number, formID: number, actionID: number): Promise<void> {
  return adminRequestVoid(`${formsRoot(siteID)}/${formID}/actions/${actionID}`, token, { method: 'DELETE' })
}
export interface ResultFilters { form_id?: number; status_id?: number; date_from?: string; date_to?: string }
export function listResults(token: string, siteID: number, page = 1, perPage = 20, filters: ResultFilters = {}): Promise<ResultsResponse> {
  const query = new URLSearchParams({ page: String(page), per_page: String(perPage) })
  for (const [key, value] of Object.entries(filters)) if (value !== undefined && value !== 0 && value !== '') query.set(key, String(value))
  return adminRequest(`${root(siteID)}/results?${query}`, token)
}
export function getResult(token: string, siteID: number, resultID: number): Promise<ResultDetailResponse> {
  return adminRequest(`${root(siteID)}/results/${resultID}`, token)
}
export function changeResultStatus(token: string, siteID: number, resultID: number, statusID: number): Promise<ResultDetailResponse> {
  return adminRequest(`${root(siteID)}/results/${resultID}/status`, token, { method: 'PATCH', body: JSON.stringify({ status_id: statusID }) })
}
export function deleteResult(token: string, siteID: number, resultID: number): Promise<void> {
  return adminRequestVoid(`${root(siteID)}/results/${resultID}`, token, { method: 'DELETE' })
}
