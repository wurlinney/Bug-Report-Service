import { clearTokens, readAccessToken, readRefreshTokens, writeTokens, type Tokens } from '@/shared/auth/tokenStorage'

export type ApiError = {
  status: number
  code?: string
  message?: string
}

type Json = Record<string, unknown> | unknown[] | string | number | boolean | null

function apiBase(): string {
  // In dev we typically rely on Vite proxy (relative /api/*).
  // In preview/static hosting we may need an absolute base URL.
  const raw = (import.meta as any).env?.VITE_API_BASE ?? (import.meta as any).env?.VITE_API_TARGET ?? ''
  return typeof raw === 'string' ? raw.replace(/\/+$/, '') : ''
}

function resolveURL(path: string): string {
  // Keep absolute URLs as-is.
  if (/^https?:\/\//i.test(path)) return path
  const base = apiBase()
  if (!base) return path
  return `${base}${path.startsWith('/') ? '' : '/'}${path}`
}

async function safeReadJson(res: Response): Promise<any> {
  const text = await res.text()
  if (!text) return null
  try {
    return JSON.parse(text)
  } catch {
    return null
  }
}

function toApiError(res: Response, body: any): ApiError {
  const nested = body?.error
  const code =
    typeof body?.code === 'string'
      ? body.code
      : typeof nested?.code === 'string'
        ? nested.code
        : undefined
  const message =
    typeof body?.message === 'string'
      ? body.message
      : typeof nested?.message === 'string'
        ? nested.message
        : typeof body?.error === 'string'
          ? body.error
          : res.statusText
  return {
    status: res.status,
    code,
    message,
  }
}

let refreshInFlight: Promise<void> | null = null

async function refreshTokens(): Promise<void> {
  if (refreshInFlight) return await refreshInFlight

  refreshInFlight = (async () => {
    const rt = readRefreshTokens()
    if (!rt) throw { status: 401, message: 'missing refresh token' } satisfies ApiError

    const res = await fetch(resolveURL('/api/v1/mod/auth/refresh'), {
      method: 'POST',
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(rt),
    })

    if (!res.ok) {
      const body = await safeReadJson(res)
      throw toApiError(res, body)
    }

    const body = (await safeReadJson(res)) as Tokens
    writeTokens(body)
  })()
    .catch((e) => {
      clearTokens()
      throw e
    })
    .finally(() => {
      refreshInFlight = null
    })

  return await refreshInFlight
}

export type RequestOptions = Omit<RequestInit, 'body'> & {
  json?: Json
  auth?: boolean
  retryOnUnauthorized?: boolean
}

export async function requestJson<T>(path: string, options?: RequestOptions): Promise<T> {
  const headers = new Headers(options?.headers)
  headers.set('Accept', 'application/json')

  if (options?.json !== undefined) {
    headers.set('Content-Type', 'application/json')
  }

  if (options?.auth) {
    const token = readAccessToken()
    if (token) headers.set('Authorization', `Bearer ${token}`)
  }

  const url = resolveURL(path)
  const res = await fetch(url, {
    ...options,
    headers,
    body: options?.json !== undefined ? JSON.stringify(options.json) : undefined,
  })

  if (res.ok) return (await safeReadJson(res)) as T

  const body = await safeReadJson(res)
  const err = toApiError(res, body)

  const shouldRetry =
    options?.auth &&
    (options?.retryOnUnauthorized ?? true) &&
    err.status === 401

  if (!shouldRetry) throw err

  await refreshTokens()

  const token = readAccessToken()
  const headers2 = new Headers(options?.headers)
  headers2.set('Accept', 'application/json')
  if (options?.json !== undefined) headers2.set('Content-Type', 'application/json')
  if (token) headers2.set('Authorization', `Bearer ${token}`)

  const res2 = await fetch(url, {
    ...options,
    headers: headers2,
    body: options?.json !== undefined ? JSON.stringify(options.json) : undefined,
  })
  if (res2.ok) return (await safeReadJson(res2)) as T

  const body2 = await safeReadJson(res2)
  throw toApiError(res2, body2)
}

