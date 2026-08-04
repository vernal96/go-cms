import type {
  AdminSessionResponse,
  APIErrorEnvelope,
  LoginCredentials,
  LoginResponse,
} from '../types/auth'

export class AdminAPIError extends Error {
  readonly status: number
  readonly code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'AdminAPIError'
    this.status = status
    this.code = code
  }
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

  try {
    const payload = (await response.json()) as Partial<APIErrorEnvelope>
    if (
      payload.error &&
      typeof payload.error.code === 'string' &&
      typeof payload.error.message === 'string'
    ) {
      code = payload.error.code
      message = payload.error.message
    }
  } catch {
    // The status still carries the server-side authorization decision.
  }

  return new AdminAPIError(response.status, code, message)
}
