import * as tus from 'tus-js-client'

function apiBase(): string {
  const raw = (import.meta as any).env?.VITE_API_BASE ?? (import.meta as any).env?.VITE_API_TARGET ?? ''
  return typeof raw === 'string' ? raw.replace(/\/+$/, '') : ''
}

function uuid(): string {
  // Works in modern browsers; fallback is good enough for idempotency key.
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) return crypto.randomUUID()
  return String(Date.now()) + '-' + Math.random().toString(16).slice(2)
}

export type UploadProgress = {
  bytesUploaded: number
  bytesTotal: number
}

function normalizedContentType(file: File): string {
  const type = (file.type || '').trim().toLowerCase()
  const name = (file.name || '').trim().toLowerCase()

  if (type === 'image/jpg') return 'image/jpeg'
  if (type === 'application/x-pdf') return 'application/pdf'
  if (type) return type

  if (name.endsWith('.pdf')) return 'application/pdf'
  if (name.endsWith('.jpg') || name.endsWith('.jpeg')) return 'image/jpeg'
  if (name.endsWith('.png')) return 'image/png'
  if (name.endsWith('.webp')) return 'image/webp'
  return ''
}

export async function uploadScreenshot(params: {
  uploadSessionId: string
  file: File
  endpoint?: string
  onProgress?: (p: UploadProgress) => void
  onControl?: (c: { abort: () => void }) => void
}): Promise<{ uploadId: string }> {
  const base = apiBase()
  const defaultEndpoint = base
    ? `${base}/api/v1/uploads`
    : typeof window !== 'undefined'
      ? `${window.location.origin}/api/v1/uploads`
      : '/api/v1/uploads'
  const { uploadSessionId, file, endpoint = defaultEndpoint, onProgress, onControl } = params

  const contentType = normalizedContentType(file)
  if (!contentType) throw new Error('Unknown file type')

  return await new Promise((resolve, reject) => {
    let aborted = false

    const expectedOrigin = (() => {
      if (typeof window === 'undefined') return ''
      try {
        // endpoint should typically be absolute. If it's not, assume same-origin.
        return new URL(endpoint).origin
      } catch {
        return window.location.origin
      }
    })()

    const normalizeUploadUrl = (url: string): string => {
      if (typeof window === 'undefined') return url
      try {
        const u = new URL(url)
        // Only rewrite cross-origin URLs when we're intentionally using same-origin proxy.
        if (expectedOrigin && expectedOrigin !== window.location.origin) return url
        if (u.origin === window.location.origin) return url
        return `${window.location.origin}${u.pathname}${u.search}`
      } catch {
        return url
      }
    }

    const upload = new tus.Upload(file, {
      endpoint,
      retryDelays: [0, 1000, 3000, 5000],
      onChunkComplete: (chunkSize, bytesAccepted, bytesTotal) => {
        // Some environments report bytesTotal=0 in onProgress; chunk callback is more reliable.
        const total = bytesTotal > 0 ? bytesTotal : file.size
        const uploaded = bytesAccepted > 0 ? bytesAccepted : Math.min(file.size, chunkSize)
        onProgress?.({ bytesUploaded: uploaded, bytesTotal: total })
      },
      onBeforeRequest: () => {
        // Safety net: ensure we never PATCH cross-origin when running behind a dev proxy.
        if (upload.url && typeof window !== 'undefined') {
          upload.url = normalizeUploadUrl(upload.url)
        }
      },
      onAfterResponse: (_req, res) => {
        // Robust progress fallback: trust server-reported offset when available.
        const off = res.getHeader('upload-offset') ?? res.getHeader('Upload-Offset')
        if (off) {
          const n = Number(off)
          if (Number.isFinite(n) && n >= 0) {
            onProgress?.({ bytesUploaded: n, bytesTotal: file.size })
          }
        }

        // When running behind a dev proxy, backend may respond with Location pointing to :8080.
        // tus-js-client will PATCH to that URL, which becomes cross-origin and can be blocked.
        const loc = res.getHeader('location') ?? res.getHeader('Location')
        if (!loc) return
        try {
          const u = new URL(loc)
          if (
            typeof window !== 'undefined' &&
            // Only rewrite in proxy scenario; for direct API base keep cross-origin as-is.
            expectedOrigin === window.location.origin &&
            u.origin !== window.location.origin
          ) {
            // Use microtask to run after internal response handling (some stacks set upload.url later).
            queueMicrotask(() => {
              upload.url = `${window.location.origin}${u.pathname}${u.search}`
            })
          }
        } catch {
          // ignore non-absolute locations
        }
      },
      metadata: {
        upload_session_id: uploadSessionId,
        filename: file.name,
        content_type: contentType,
        idempotency_key: uuid(),
      },
      onError: (error) => {
        if (aborted) reject(new Error('upload_cancelled'))
        else reject(error)
      },
      onProgress: (bytesUploaded, bytesTotal) => {
        const total = bytesTotal > 0 ? bytesTotal : file.size
        onProgress?.({ bytesUploaded, bytesTotal: total })
      },
      onSuccess: () => {
        const url = upload.url ?? ''
        const id = url.split('/').filter(Boolean).pop() ?? ''
        resolve({ uploadId: id })
      },
    })

    onControl?.({
      abort: () => {
        aborted = true
        upload.abort(true)
      },
    })

    upload.start()
  })
}
