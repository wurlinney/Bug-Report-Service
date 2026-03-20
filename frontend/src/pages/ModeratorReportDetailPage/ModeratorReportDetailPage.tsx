import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { changeReportMeta, getReport, listReportAttachments } from '@/features/moderator/api/moderatorApi'
import type { AttachmentItem, ReportItem } from '@/shared/api/types'
import { formatUnixSeconds } from '@/shared/lib/date'
import { Card } from '@/shared/ui/Card/Card'
import { Button } from '@/shared/ui/Button/Button'
import { Alert } from '@/shared/ui/Alert/Alert'
import styles from './ModeratorReportDetailPage.module.css'

const STATUS_OPTIONS: { value: string; label: string }[] = [
  { value: 'new', label: 'Новое' },
  { value: 'in_review', label: 'В работе' },
  { value: 'resolved', label: 'Решено' },
]

const PRIORITY_OPTIONS: { value: string; label: string }[] = [
  { value: 'Высокий', label: 'Высокий' },
  { value: 'Средний', label: 'Средний' },
  { value: 'Низкий', label: 'Низкий' },
  { value: 'Не задан', label: 'Не задан' },
]

const INFLUENCE_OPTIONS: { value: string; label: string }[] = [
  { value: 'Крит/блокер', label: 'Крит/блокер' },
  { value: 'Высокий', label: 'Высокий' },
  { value: 'Средний', label: 'Средний' },
  { value: 'Низкий', label: 'Низкий' },
  { value: 'Не баг а фича', label: 'Не баг а фича' },
  { value: 'Не задано', label: 'Не задано' },
]

function getStatusLabel(value: string): string {
  const found = STATUS_OPTIONS.find((s) => s.value === value)
  return found?.label ?? value
}

function getStatusBadgeClass(value: string): string {
  switch (value) {
    case 'new':
      return `${styles.badge} ${styles.statusNew}`
    case 'in_review':
      return `${styles.badge} ${styles.statusInReview}`
    case 'resolved':
      return `${styles.badge} ${styles.statusResolved}`
    default:
      return styles.badge
  }
}

function normalizeInfluence(value: string): string {
  return value === 'Крит/Блокер' ? 'Крит/блокер' : value
}

function getPriorityBadgeClass(value: string): string {
  switch (value) {
    case 'Высокий':
      return `${styles.badge} ${styles.priorityHigh}`
    case 'Средний':
      return `${styles.badge} ${styles.priorityMedium}`
    case 'Низкий':
      return `${styles.badge} ${styles.priorityLow}`
    default:
      return styles.badge
  }
}

function getInfluenceBadgeClass(value: string): string {
  switch (normalizeInfluence(value)) {
    case 'Крит/блокер':
      return `${styles.badge} ${styles.influenceBlocker}`
    case 'Высокий':
      return `${styles.badge} ${styles.influenceHigh}`
    case 'Средний':
      return `${styles.badge} ${styles.influenceMedium}`
    case 'Низкий':
      return `${styles.badge} ${styles.influenceLow}`
    case 'Не баг а фича':
      return `${styles.badge} ${styles.influenceFeature}`
    default:
      return styles.badge
  }
}

function isImageAttachment(contentType: string): boolean {
  return contentType.startsWith('image/')
}

export function ModeratorReportDetailPage() {
  const { id } = useParams()
  const [item, setItem] = useState<ReportItem | null>(null)
  const [attachments, setAttachments] = useState<AttachmentItem[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [newStatus, setNewStatus] = useState('')
  const [newPriority, setNewPriority] = useState('Не задан')
  const [newInfluence, setNewInfluence] = useState('Не задано')
  const [saving, setSaving] = useState(false)

  const load = useCallback(async (reportId: string) => {
    setLoading(true)
    setError(null)
    try {
      const [resp, atts] = await Promise.all([getReport(reportId), listReportAttachments(reportId)])
      setItem(resp)
      setNewStatus(resp.status)
      setNewPriority(resp.priority || 'Не задан')
      setNewInfluence(normalizeInfluence(resp.influence || 'Не задано'))
      setAttachments(atts.items ?? [])
    } catch (e: any) {
      setError(e?.message ?? 'Ошибка запроса')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (!id) return
    void load(id)
  }, [id, load])

  if (!id) return <Card>Missing id</Card>

  return (
    <div className={styles.page}>
      <Card className={styles.card}>
        <div className={`${styles.row} ${styles.rowSpaceBetween} ${styles.headerRow}`}>
          <div className={styles.headerSideLeft}>
            <Link to="/mod/reports" style={{ textDecoration: 'none' }}>
              <Button type="button">← Назад</Button>
            </Link>
            <h1 className={styles.title}>Обращение</h1>
          </div>
          <div className={styles.headerSideRight}>
            <Button type="button" onClick={() => void load(id)} disabled={loading}>
              {loading ? 'Загрузка…' : 'Обновить'}
            </Button>
            <div className={styles.muted}>
              ID: <code className={styles.mono}>{id}</code>
            </div>
          </div>
        </div>

        {error && (
          <div className={styles.errorBlock}>
            <Alert variant="error" role="alert">
              {error}
            </Alert>
          </div>
        )}

        {item && (
          <div className={`${styles.stack} ${styles.contentBlock}`}>
            {(() => {
              const currentStatus = item.status
              const currentPriority = item.priority || 'Не задан'
              const currentInfluence = normalizeInfluence(item.influence || 'Не задано')
              const hasChanges =
                newStatus.trim() !== currentStatus ||
                newPriority.trim() !== currentPriority ||
                normalizeInfluence(newInfluence.trim()) !== currentInfluence

              return (
                <>
            <div className={styles.topRow}>
              <div className={styles.kvBlock}>
                <div className={styles.kvKey}>Имя отправителя</div>
                <div className={styles.kvValue}>{item.reporter_name}</div>
              </div>
              <div className={styles.kvBlockCenter}>
                <div className={styles.kvKey}>Создано</div>
                <div className={styles.kvValue}>{formatUnixSeconds(item.created_at)}</div>
              </div>
              <div className={styles.kvBlockRight}>
                <div className={styles.kvKey}>Обновлено</div>
                <div className={styles.kvValue}>{formatUnixSeconds(item.updated_at)}</div>
              </div>
            </div>

            <div className={styles.metaRow}>
              <div className={styles.kvBlockRight}>
                <div className={styles.kvKey}>Текущий статус</div>
                <div className={styles.kvValueStatus}>
                  <span className={getStatusBadgeClass(item.status)}>{getStatusLabel(item.status)}</span>
                </div>
              </div>
              <div className={styles.kvBlockRight}>
                <div className={styles.kvKey}>Текущий приоритет</div>
                <div className={styles.kvValueStatus}>
                  <span className={getPriorityBadgeClass(item.priority || 'Не задан')}>{item.priority || 'Не задан'}</span>
                </div>
              </div>
              <div className={styles.kvBlockRight}>
                <div className={styles.kvKey}>Текущее влияние</div>
                <div className={styles.kvValueStatus}>
                  <span className={getInfluenceBadgeClass(item.influence || 'Не задано')}>
                    {normalizeInfluence(item.influence || 'Не задано')}
                  </span>
                </div>
              </div>
            </div>

            <div className={styles.descriptionBlock}>
              <div className={styles.kvKey}>Описание</div>
              <div className={`${styles.kvValue} ${styles.pre}`}>
                {item.description || <span className={styles.muted}>—</span>}
              </div>
            </div>

            <div>
              <div className={styles.kvKey}>Вложения</div>
              {attachments.length === 0 ? (
                <div className={`${styles.kvValue} ${styles.muted}`}>—</div>
              ) : (
                <div className={styles.attachGrid}>
                  {attachments.map((a) => (
                    <a
                      key={a.id}
                      className={styles.attachItem}
                      href={a.download_url}
                      target="_blank"
                      rel="noreferrer"
                      title={`${a.file_name} (${Math.round(a.file_size / 1024)} KB)`}
                    >
                      {isImageAttachment(a.content_type) ? (
                        <img
                          className={styles.attachImg}
                          src={a.download_url}
                          alt={a.file_name}
                          loading="lazy"
                          onError={() => {
                            // Если объект в S3 отсутствует (например, отменили на клиенте), убираем attachment из UI.
                            setAttachments((prev) => prev.filter((x) => x.id !== a.id))
                          }}
                        />
                      ) : (
                        <div className={styles.attachNonImage}>PDF</div>
                      )}
                      <div className={styles.attachMeta}>
                        <div className={styles.attachName}>{a.file_name}</div>
                        <div className={styles.muted}>{Math.round(a.file_size / 1024)} KB</div>
                      </div>
                    </a>
                  ))}
                </div>
              )}
            </div>

            <div className={styles.controlsRow}>
              <div className={styles.controlField}>
                <label className={styles.statusLabel}>
                  <span className={styles.kvKey}>Изменить статус</span>
                </label>
                <div className={styles.radioList}>
                  {STATUS_OPTIONS.map((opt) => (
                    <label key={opt.value} className={styles.radioOption}>
                      <input
                        type="radio"
                        name="report_status"
                        value={opt.value}
                        checked={newStatus === opt.value}
                        onChange={() => setNewStatus(opt.value)}
                        disabled={saving}
                      />
                      <span className={styles.radioText}>{opt.label}</span>
                    </label>
                  ))}
                </div>
              </div>

              <div className={styles.controlField}>
                <label className={styles.statusLabel}>
                  <span className={styles.kvKey}>Изменить приоритет</span>
                </label>
                <div className={styles.radioList}>
                  {PRIORITY_OPTIONS.map((opt) => (
                    <label key={opt.value} className={styles.radioOption}>
                      <input
                        type="radio"
                        name="report_priority"
                        value={opt.value}
                        checked={newPriority === opt.value}
                        onChange={() => setNewPriority(opt.value)}
                        disabled={saving}
                      />
                      <span className={styles.radioText}>{opt.label}</span>
                    </label>
                  ))}
                </div>
              </div>

              <div className={styles.controlField}>
                <label className={styles.statusLabel}>
                  <span className={styles.kvKey}>Изменить влияние</span>
                </label>
                <div className={styles.radioList}>
                  {INFLUENCE_OPTIONS.map((opt) => (
                    <label key={opt.value} className={styles.radioOption}>
                      <input
                        type="radio"
                        name="report_influence"
                        value={opt.value}
                        checked={newInfluence === opt.value}
                        onChange={() => setNewInfluence(opt.value)}
                        disabled={saving}
                      />
                      <span className={styles.radioText}>{opt.label}</span>
                    </label>
                  ))}
                </div>
              </div>
            </div>

            <div className={styles.saveRow}>
              <Button
                variant="primary"
                type="button"
                disabled={saving || !hasChanges || newStatus.trim() === '' || newPriority.trim() === '' || newInfluence.trim() === ''}
                onClick={async () => {
                  setSaving(true)
                  setError(null)
                  try {
                    await changeReportMeta(id, newStatus.trim(), newPriority.trim(), normalizeInfluence(newInfluence.trim()))
                    await load(id)
                  } catch (e: any) {
                    setError(e?.message ?? 'Ошибка запроса')
                  } finally {
                    setSaving(false)
                  }
                }}
              >
                {saving ? 'Сохранение…' : 'Сохранить'}
              </Button>
            </div>
                </>
              )
            })()}
          </div>
        )}
      </Card>
    </div>
  )
}

