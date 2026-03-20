const ACCESS_TOKEN_KEY = 'buglog.access_token'
const REFRESH_TOKEN_ID_KEY = 'buglog.refresh_token_id'
const REFRESH_TOKEN_KEY = 'buglog.refresh_token'

export type Tokens = {
  access_token: string
  refresh_token_id: string
  refresh_token: string
}

export type RefreshTokens = Pick<Tokens, 'refresh_token_id' | 'refresh_token'>

export function readAccessToken(): string | null {
  return localStorage.getItem(ACCESS_TOKEN_KEY)
}

export function readRefreshTokens(): RefreshTokens | null {
  const refresh_token_id = localStorage.getItem(REFRESH_TOKEN_ID_KEY)
  const refresh_token = localStorage.getItem(REFRESH_TOKEN_KEY)
  if (!refresh_token_id || !refresh_token) return null
  return { refresh_token_id, refresh_token }
}

export function writeTokens(tokens: Tokens) {
  localStorage.setItem(ACCESS_TOKEN_KEY, tokens.access_token)
  localStorage.setItem(REFRESH_TOKEN_ID_KEY, tokens.refresh_token_id)
  localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refresh_token)
}

export function clearTokens() {
  localStorage.removeItem(ACCESS_TOKEN_KEY)
  localStorage.removeItem(REFRESH_TOKEN_ID_KEY)
  localStorage.removeItem(REFRESH_TOKEN_KEY)
}

