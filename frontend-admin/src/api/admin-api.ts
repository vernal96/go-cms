import type {
  AdminSessionResponse,
  APIErrorEnvelope,
  LoginCredentials,
  LoginResponse,
  FieldValidationError,
} from '../types/auth'

export class AdminAPIError extends Error {
  readonly status: number
  readonly code: string
  readonly fieldErrors: FieldValidationError[]

  constructor(
    status: number,
    code: string,
    message: string,
    fieldErrors: FieldValidationError[] = [],
  ) {
    super(message)
    this.name = 'AdminAPIError'
    this.status = status
    this.code = code
    this.fieldErrors = fieldErrors
  }
}

let unauthorizedHandler: (() => void) | null = null

export function setAdminUnauthorizedHandler(
  handler: (() => void) | null,
): void {
  unauthorizedHandler = handler
}

export async function login(
  credentials: LoginCredentials,
): Promise<LoginResponse> {
  return requestJSON<LoginResponse>('/api/auth/login', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(credentials),
  })
}

export async function loadAdminSession(
  accessToken: string,
): Promise<AdminSessionResponse> {
  return requestJSON<AdminSessionResponse>('/api/admin/session', {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  })
}

export async function adminRequest<T>(
  path: string,
  accessToken: string,
  init: RequestInit = {},
): Promise<T> {
  const headers = new Headers(init.headers)
  headers.set('Authorization', `Bearer ${accessToken}`)
  if (init.body !== undefined) {
    headers.set('Content-Type', 'application/json')
  }
  try {
    return await requestJSON<T>(path, { ...init, headers })
  } catch (error) {
    handleAdminError(error)
    throw error
  }
}

export async function adminRequestVoid(
  path: string,
  accessToken: string,
  init: RequestInit,
): Promise<void> {
  const headers = new Headers(init.headers)
  headers.set('Authorization', `Bearer ${accessToken}`)
  const response = await fetch(path, { ...init, headers })
  if (!response.ok) {
    const error = await responseError(response)
    handleAdminError(error)
    throw error
  }
}

function handleAdminError(error: unknown): void {
  if (error instanceof AdminAPIError && error.status === 401)
    unauthorizedHandler?.()
}

async function requestJSON<T>(
  input: RequestInfo | URL,
  init?: RequestInit,
): Promise<T> {
  const response = await fetch(input, init)
  if (!response.ok) {
    throw await responseError(response)
  }

  try {
    return (await response.json()) as T
  } catch {
    throw new AdminAPIError(
      response.status,
      'invalid_response',
      'Сервер вернул некорректный ответ',
    )
  }
}

async function responseError(response: Response): Promise<AdminAPIError> {
  let code = 'request_failed'
  let message = `Запрос завершился с кодом ${response.status}`
  let fieldErrors: FieldValidationError[] = []

  try {
    const payload = (await response.json()) as Partial<APIErrorEnvelope>
    if (
      payload.error &&
      typeof payload.error.code === 'string' &&
      typeof payload.error.message === 'string'
    ) {
      code = payload.error.code
      message = payload.error.message
      if (Array.isArray(payload.error.details?.fields)) {
        fieldErrors = payload.error.details.fields.filter(
          (item): item is FieldValidationError =>
            typeof item?.key === 'string' &&
            typeof item?.rule === 'string' &&
            typeof item?.param === 'string',
        )
      }
    }
  } catch {
    // The status still carries the server-side authorization decision.
  }

  return new AdminAPIError(response.status, code, message, fieldErrors)
}
