import { requestJson } from '@/shared/api/httpClient'
import { writeTokens, type Tokens } from '@/shared/auth/tokenStorage'
import type { AttachmentItem, ReportItem } from '@/shared/api/types'

export type ModLoginReq = { email: string; password: string }

export async function modLogin(payload: ModLoginReq): Promise<void> {
  const tokens = await requestJson<Tokens>('/api/v1/mod/auth/login', {
    method: 'POST',
    json: payload,
    auth: false,
    retryOnUnauthorized: false,
  })
  writeTokens(tokens)
}

export async function modMe(): Promise<{ id: string; name: string; email: string; role: string }> {
  return await requestJson('/api/v1/mod/me', { auth: true })
}

export async function listReports(params: {
  limit?: number
  offset?: number
  status?: string
  q?: string
  reporter_name?: string
  sort_by?: 'created_at' | 'updated_at' | ''
  sort_desc?: boolean
}): Promise<{ items: ReportItem[]; total: number }> {
  const qp = new URLSearchParams()
  if (params.limit != null) qp.set('limit', String(params.limit))
  if (params.offset != null) qp.set('offset', String(params.offset))
  if (params.status) qp.set('status', params.status)
  if (params.q) qp.set('q', params.q)
  if (params.reporter_name) qp.set('reporter_name', params.reporter_name)
  if (params.sort_by) qp.set('sort_by', params.sort_by)
  if (params.sort_desc) qp.set('sort_desc', '1')

  const qs = qp.toString()
  return await requestJson(`/api/v1/mod/reports${qs ? `?${qs}` : ''}`, { auth: true })
}

export async function getReport(id: string): Promise<ReportItem> {
  return await requestJson(`/api/v1/mod/reports/${encodeURIComponent(id)}`, { auth: true })
}

export async function changeReportStatus(id: string, status: string): Promise<void> {
  await requestJson(`/api/v1/mod/reports/${encodeURIComponent(id)}/status`, {
    method: 'PATCH',
    auth: true,
    json: { status },
  })
}

export async function changeReportMeta(id: string, status: string, priority: string, influence: string): Promise<void> {
  await requestJson(`/api/v1/mod/reports/${encodeURIComponent(id)}/status`, {
    method: 'PATCH',
    auth: true,
    json: { status, priority, influence },
  })
}

export async function listReportAttachments(reportId: string): Promise<{ items: AttachmentItem[] }> {
  return await requestJson(`/api/v1/mod/reports/${encodeURIComponent(reportId)}/attachments`, { auth: true })
}
