import { requestJson } from '@/shared/api/httpClient'
import type { ReportItem } from '@/shared/api/types'

export type PublicCreateReportReq = {
  reporter_name: string
  description: string
  upload_session_id?: string
}

export type PublicCreateReportResp = ReportItem & { message?: string }

export type UploadSessionResp = {
  id: string
  created_at: number
}

export async function createUploadSession(): Promise<UploadSessionResp> {
  return await requestJson('/api/v1/public/upload-sessions', {
    method: 'POST',
  })
}

export async function createPublicReport(payload: PublicCreateReportReq): Promise<PublicCreateReportResp> {
  return await requestJson('/api/v1/public/reports', {
    method: 'POST',
    json: payload,
  })
}

export async function deleteUploadFromSession(params: { upload_session_id: string; upload_id: string }): Promise<void> {
  await requestJson(
    `/api/v1/public/upload-sessions/${encodeURIComponent(params.upload_session_id)}/uploads/${encodeURIComponent(params.upload_id)}`,
    { method: 'DELETE' },
  )
}
