export type ReportItem = {
  id: string
  reporter_name: string
  description: string
  status: string
  influence: string
  priority: string
  created_at: number
  updated_at: number
}

export type AttachmentItem = {
  id: number
  report_id: string
  file_name: string
  content_type: string
  file_size: number
  storage_key: string
  created_at: number
  download_url: string
}
