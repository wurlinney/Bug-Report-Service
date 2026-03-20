import { useEffect, useMemo, useRef, useState } from 'react'
import { createPublicReport, createUploadSession, deleteUploadFromSession } from '@/features/public-report/api/publicReportApi'
import { uploadScreenshot } from '@/features/public-report/upload/tusUpload'
import { Card } from '@/shared/ui/Card/Card'
import { Button } from '@/shared/ui/Button/Button'
import { Alert } from '@/shared/ui/Alert/Alert'
import styles from './PublicReportPage.module.css'

type UploadItem = {
  id: string
  file: File
  progress: number
  status: 'pending' | 'uploading' | 'done' | 'cancelled' | 'error'
  error?: string
  uploadId?: string
}

const ALLOWED_UPLOAD_MIME_TYPES = ['image/png', 'image/jpeg', 'image/webp', 'application/pdf'] as const

function makeUploadID(file: File): string {
  return `${file.name}-${file.size}-${file.lastModified}-${Math.random().toString(16).slice(2)}`
}

function formatBytes(n: number): string {
  if (!Number.isFinite(n) || n <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let v = n
  let i = 0
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  const digits = i === 0 ? 0 : v < 10 ? 1 : 0
  return `${v.toFixed(digits)} ${units[i]}`
}

function toUserErrorMessage(err: unknown): string {
  const raw = String((err as any)?.message ?? err ?? '').toLowerCase()

  if (!raw) return 'Не удалось выполнить запрос. Попробуйте еще раз.'
  if (raw.includes('upload_cancelled')) return 'Загрузка файла отменена.'
  if (raw.includes('unsupported_media_type')) return 'Неподдерживаемый формат файла. Разрешены: jpeg, png, webp, pdf.'
  if (raw.includes('file too large') || raw.includes('file_too_large') || raw.includes('413')) {
    return 'Файл слишком большой. Максимальный размер файла — 20 МБ.'
  }
  if (raw.includes('validation_error') || raw.includes('400')) {
    return 'Проверьте корректность заполненных полей и повторите попытку.'
  }
  if (raw.includes('401') || raw.includes('403')) {
    return 'Недостаточно прав для выполнения операции.'
  }
  if (raw.includes('404')) {
    return 'Сервис временно недоступен. Попробуйте позже.'
  }
  if (raw.includes('500') || raw.includes('502') || raw.includes('503') || raw.includes('504') || raw.includes('internal_error')) {
    return 'Сервис временно недоступен. Мы уже работаем над этим. Попробуйте позже.'
  }
  if (raw.includes('network') || raw.includes('failed to fetch') || raw.includes('timeout')) {
    return 'Проблема с сетью. Проверьте подключение и попробуйте снова.'
  }
  return 'Не удалось выполнить запрос. Попробуйте еще раз.'
}

export function PublicReportPage() {
  const [reporterName, setReporterName] = useState('')
  const [description, setDescription] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [creatingSession, setCreatingSession] = useState(false)
  const [dragActive, setDragActive] = useState(false)
  const [uploadSessionID, setUploadSessionID] = useState<string | null>(null)
  const [result, setResult] = useState<{ id: string; message?: string } | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [uploads, setUploads] = useState<UploadItem[]>([])
  const [pendingSessionDeletes, setPendingSessionDeletes] = useState(0)
  const uploadControlsRef = useRef(new Map<string, { abort: () => void }>())
  const cancelledRef = useRef(new Set<string>())
  const fileInputRef = useRef<HTMLInputElement | null>(null)
  const startUploadsRef = useRef<(files: File[]) => Promise<void>>(
    async () => {
      // replaced after function declaration
    },
  )

  const canSubmit = useMemo(
    () => reporterName.trim().length > 0 && !submitting && !creatingSession && pendingSessionDeletes === 0,
    [reporterName, submitting, creatingSession, pendingSessionDeletes],
  )

  async function ensureUploadSession(): Promise<string> {
    if (uploadSessionID) return uploadSessionID
    const created = await createUploadSession()
    setUploadSessionID(created.id)
    return created.id
  }

  async function runUpload(item: UploadItem, sessionID: string): Promise<void> {
    setUploads((prev) =>
      prev.map((current) =>
        current.id === item.id
          ? { ...current, status: 'uploading', progress: Math.max(current.progress, 0.01), error: undefined }
          : current,
      ),
    )

    try {
      const { uploadId } = await uploadScreenshot({
        uploadSessionId: sessionID,
        file: item.file,
        onControl: (c) => {
          uploadControlsRef.current.set(item.id, c)
        },
        onProgress: ({ bytesUploaded, bytesTotal }) => {
          const total = bytesTotal > 0 ? bytesTotal : item.file.size
          const progress = total > 0 ? bytesUploaded / total : 0
          setUploads((prev) =>
            prev.map((current) =>
              current.id === item.id
                ? { ...current, progress: Math.max(current.progress, progress) }
                : current,
            ),
          )
        },
      })

      setUploads((prev) =>
        prev.map((current) =>
          current.id === item.id
            ? { ...current, status: 'done', progress: 1, error: undefined, uploadId }
            : current,
        ),
      )
    } catch (e: any) {
      if (cancelledRef.current.has(item.id) || e?.message === 'upload_cancelled') {
        setUploads((prev) =>
          prev.map((current) =>
            current.id === item.id
              ? { ...current, status: 'cancelled', error: undefined }
              : current,
          ),
        )
        return
      }
      setUploads((prev) =>
        prev.map((current) =>
          current.id === item.id
            ? { ...current, status: 'error', error: toUserErrorMessage(e) }
            : current,
        ),
      )
      throw e
    }
  }

  async function startUploads(files: File[]): Promise<void> {
    if (files.length === 0) return

    const newItems: UploadItem[] = files.map((file) => ({
      id: makeUploadID(file),
      file,
      progress: 0,
      status: 'pending',
    }))

    setError(null)
    setUploads((prev) => [...prev, ...newItems])
    setCreatingSession(true)

    try {
      const sessionID = await ensureUploadSession()
      await Promise.all(newItems.map((item) => runUpload(item, sessionID)))
    } catch (e: any) {
      setError(toUserErrorMessage(e))
    } finally {
      setCreatingSession(false)
    }
  }

  // Keep latest startUploads implementation for global paste handler.
  startUploadsRef.current = startUploads

  function isProbablyAllowedUploadFile(file: File): boolean {
    if (ALLOWED_UPLOAD_MIME_TYPES.includes(file.type as (typeof ALLOWED_UPLOAD_MIME_TYPES)[number])) return true
    const name = (file.name || '').toLowerCase()
    return name.endsWith('.pdf') || name.endsWith('.png') || name.endsWith('.jpg') || name.endsWith('.jpeg') || name.endsWith('.webp')
  }

  // Allow pasting files (Ctrl+V) into the form upload area.
  // Notes:
  // - We ignore paste into inputs/textarea to not break normal text editing.
  // - We extract File objects from clipboard items and reuse the existing tus flow.
  useEffect(() => {
    const onPaste = (e: ClipboardEvent) => {
      if (submitting || creatingSession) return
      const target = e.target as HTMLElement | null
      if (target) {
        const tag = target.tagName?.toLowerCase()
        if (tag === 'textarea' || tag === 'input' || target.isContentEditable) return
      }

      const dt = e.clipboardData
      if (!dt) return

      const items = Array.from(dt.items ?? [])
      const files = items
        .filter((i) => i.kind === 'file')
        .map((i) => i.getAsFile())
        .filter((f): f is File => Boolean(f))

      const allowed = files.filter(isProbablyAllowedUploadFile)
      if (allowed.length === 0) return

      e.preventDefault()
      void startUploadsRef.current(allowed)
    }

    window.addEventListener('paste', onPaste)
    return () => window.removeEventListener('paste', onPaste)
  }, [submitting, creatingSession])

  async function cancelUpload(u: UploadItem): Promise<void> {
    cancelledRef.current.add(u.id)
    uploadControlsRef.current.get(u.id)?.abort()

    // If already uploaded and we have a session, remove from session on backend so it won't be bound to the report.
    if (u.status === 'done' && u.uploadId && uploadSessionID) {
      try {
        setPendingSessionDeletes((n) => n + 1)
        await deleteUploadFromSession({ upload_session_id: uploadSessionID, upload_id: u.uploadId })
      } catch {
        // Best-effort: if backend call fails, still remove from UI to match user's intent.
      } finally {
        setPendingSessionDeletes((n) => Math.max(0, n - 1))
      }
    }

    // Close the upload row immediately (and collapse the whole block if it was the last item).
    setUploads((prev) => prev.filter((x) => x.id !== u.id))
  }

  return (
    <div className={styles.stack}>
      <Card>
        <h1 className={styles.title}>Оставить обращение</h1>

        <div className={styles.form}>
          <label className={styles.field}>
            <span className={styles.label}>Фамилия и имя</span>
            <input
              className={styles.input}
              value={reporterName}
              onChange={(e) => setReporterName(e.target.value)}
              placeholder="Введите вашу фамилию и имя..."
              autoComplete="name"
              disabled={submitting || creatingSession}
            />
          </label>

          <label className={styles.field}>
            <span className={styles.label}>Описание</span>
            <textarea
              className={styles.textarea}
              value={description}
              onChange={(e) => setDescription(e.target.value.slice(0, 350))}
              placeholder="Опишите проблему..."
              rows={6}
              maxLength={350}
              disabled={submitting || creatingSession}
            />
            <span className={styles.counter}>{description.length}/350</span>
          </label>

          <div className={styles.field}>
            <span className={styles.label}>Прикрепите файлы (изображения или PDF)</span>
            <div
              className={dragActive ? `${styles.uploadBox} ${styles.uploadBoxActive}` : styles.uploadBox}
              onDragEnter={(e) => {
                if (submitting || creatingSession) return
                e.preventDefault()
                e.stopPropagation()
                setDragActive(true)
              }}
              onDragOver={(e) => {
                if (submitting || creatingSession) return
                e.preventDefault()
                e.stopPropagation()
                setDragActive(true)
              }}
              onDragLeave={(e) => {
                e.preventDefault()
                e.stopPropagation()
                setDragActive(false)
              }}
              onDrop={async (e) => {
                if (submitting || creatingSession) return
                e.preventDefault()
                e.stopPropagation()
                setDragActive(false)

                const dt = e.dataTransfer
                const files = Array.from(dt?.files ?? []).filter((f) => ALLOWED_UPLOAD_MIME_TYPES.includes(f.type as (typeof ALLOWED_UPLOAD_MIME_TYPES)[number]))
                await startUploads(files)
              }}
            >
              <input
                ref={fileInputRef}
                className={styles.fileInput}
                type="file"
                multiple
                accept="image/png,image/jpeg,image/webp,application/pdf,.pdf"
                disabled={submitting || creatingSession}
                onChange={async (e) => {
                  const files = Array.from(e.target.files ?? [])
                  e.currentTarget.value = ''
                  await startUploads(files)
                }}
              />
              <span
                className={styles.uploadButton}
                role="button"
                tabIndex={0}
                onClick={() => fileInputRef.current?.click()}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') fileInputRef.current?.click()
                }}
              >
                {creatingSession ? 'Загрузка файлов...' : 'Выбрать файлы'}
              </span>
              <span className={styles.uploadHint}>Можно перетащить изображения/PDF сюда или вставить из буфера обмена (Ctrl+V).</span>
            </div>
          </div>
          <span className={styles.uploadFormats}>Принимаются: jpeg, png, webp, pdf</span>

          {uploads.length > 0 && (
            <div className={styles.fileList}>
              {uploads.map((u) => (
                <div key={u.id} className={styles.fileRow}>
                  <div className={styles.fileName}>{u.file.name}</div>

                  <div className={styles.progress} aria-label={`Прогресс загрузки: ${Math.round(u.progress * 100)}%`}>
                    <div className={styles.progressBar} style={{ width: `${Math.round(u.progress * 100)}%` }} />
                  </div>

                  <div className={styles.inlineMeta}>
                    <div className={styles.percent}>{Math.round(u.progress * 100)}%</div>
                    <div className={styles.fileSizeRight}>{formatBytes(u.file.size)}</div>
                    <div
                      className={
                        u.status === 'done'
                          ? `${styles.statusPill} ${styles.statusDone}`
                          : u.status === 'cancelled'
                            ? `${styles.statusPill} ${styles.statusCancelled}`
                          : u.status === 'error'
                            ? `${styles.statusPill} ${styles.statusError}`
                            : u.status === 'uploading'
                              ? `${styles.statusPill} ${styles.statusUploading}`
                              : styles.statusPill
                      }
                    >
                      {u.status === 'done'
                        ? 'Готово'
                        : u.status === 'cancelled'
                          ? 'Отменено'
                          : u.status === 'error'
                            ? 'Ошибка'
                            : u.status === 'uploading'
                              ? 'Загрузка'
                              : 'Ожидает'}
                    </div>

                    <button type="button" className={styles.cancelX} onClick={() => void cancelUpload(u)} aria-label="Отменить">
                      ×
                    </button>
                  </div>

                  {u.status === 'error' && <div className={styles.fileError}>Ошибка: {u.error ?? 'upload failed'}</div>}
                </div>
              ))}
            </div>
          )}

          <div className={styles.actionsRow}>
            <Button
              className={styles.actionBtn}
              variant="primary"
              disabled={!canSubmit}
              onClick={async () => {
                setError(null)
                setResult(null)
                setSubmitting(true)
                try {
                  const hasAnyDoneUploads = uploads.some((u) => u.status === 'done' && Boolean(u.uploadId))
                  const resp = await createPublicReport({
                    reporter_name: reporterName.trim(),
                    description: description.trim(),
                    // Не передаём `upload_session_id`, если в UI уже нет загруженных файлов.
                    // Так мы исключаем привязку attachment-ов, которые пользователь мог отменить.
                    upload_session_id: uploadSessionID && hasAnyDoneUploads ? uploadSessionID : undefined,
                  })
                  setResult({ id: resp.id, message: resp.message })
                  setReporterName('')
                  setDescription('')
                  setUploadSessionID(null)
                  setUploads([])
                } catch (e: any) {
                  setError(toUserErrorMessage(e))
                } finally {
                  setSubmitting(false)
                }
              }}
            >
              {submitting ? 'Отправка...' : 'Отправить'}
            </Button>

            <Button
              className={styles.actionBtn}
              type="button"
              onClick={() => {
                setReporterName('')
                setDescription('')
                setUploadSessionID(null)
                setError(null)
                setResult(null)
                setUploads([])
              }}
              disabled={submitting || creatingSession}
            >
              Очистить
            </Button>
          </div>

          {error && (
            <Alert variant="error" role="alert">
              {error}
            </Alert>
          )}

          {result && (
            <Alert variant="ok" role="status">
              <div className={styles.resultStack}>
                <div>{result.message ?? 'Обращение создано'}</div>
                <div>
                  ID: <code className={styles.mono}>{result.id}</code>
                </div>
              </div>
            </Alert>
          )}
        </div>
      </Card>
    </div>
  )
}
