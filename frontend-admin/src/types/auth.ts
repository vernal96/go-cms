export interface LoginCredentials {
  identifier: string
  password: string
}

export interface LoginResponse {
  access_token: string
  token_type: 'Bearer'
  expires_at: string
}

export interface StoredSession {
  accessToken: string
  expiresAt: string
}

export interface AdminUser {
  id: number
  login: string
  email: string
  display_name: string
  color_scheme: ColorScheme
  accent_color: AccentColor
  has_avatar: boolean
  avatar_updated_at: string | null
}

export type ColorScheme = 'light' | 'dark' | 'system'
export type AccentColor = 'blue' | 'violet' | 'indigo' | 'emerald' | 'amber' | 'rose'

export interface AdminSessionResponse {
  user: AdminUser
  permissions: string[]
}

export interface APIErrorEnvelope {
  error: {
    code: string
    message: string
    details?: {
      fields?: FieldValidationError[]
    }
  }
}

export interface FieldValidationError {
  key: string
  rule: string
  param: string
}

export type AuthStatus =
  | 'checking'
  | 'anonymous'
  | 'authenticating'
  | 'forbidden'
  | 'authorized'
