import { useCallback, useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { changeReportStatus, listReports } from '@/features/moderator/api/moderatorApi'
import type { ReportItem } from '@/shared/api/types'
import { readAccessToken } from '@/shared/auth/tokenStorage'
import { formatUnixSeconds } from '@/shared/lib/date'
import { Card } from '@/shared/ui/Card/Card'
import { Button } from '@/shared/ui/Button/Button'
import { Alert } from '@/shared/ui/Alert/Alert'
import styles from './ModeratorReportsPage.module.css'

const STATUS_OPTIONS: { value: string; label: string }[] = [
  { value: 'new', label: 'Новые' },
  { value: 'in_review', label: 'В работе' },
  { value: 'resolved', label: 'Решено' },
]

const PRIORITY_OPTIONS = ['Высокий', 'Средний', 'Низкий'] as const
const INFLUENCE_OPTIONS = ['Крит/блокер', 'Высокий', 'Средний', 'Низкий', 'Не баг а фича'] as const
const UNSET_PRIORITY = '__unset_priority__'
const UNSET_INFLUENCE = '__unset_influence__'

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

export function ModeratorReportsPage() {
  const nav = useNavigate()
  const [itemsByStatus, setItemsByStatus] = useState<Record<string, ReportItem[]>>({})
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [draggingId, setDraggingId] = useState<string | null>(null)
  const [dragOverStatus, setDragOverStatus] = useState<string | null>(null)
  const [updatingStatus, setUpdatingStatus] = useState(false)
  const isDragRef = useRef(false)

  const [q, setQ] = useState('')
  const [sortDesc, setSortDesc] = useState(true)
  const [priorityFilter, setPriorityFilter] = useState('')
  const [influenceFilter, setInfluenceFilter] = useState('')

  const hasToken = Boolean(readAccessToken())

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const qTrim = q.trim() || undefined

      // Загружаем отдельно по каждому статусу, чтобы таблицы были "честными":
      // один запрос -> одна секция.
      const results = await Promise.all(
        STATUS_OPTIONS.map(async (s) => {
          const resp = await listReports({
            limit: 20,
            offset: 0,
            q: qTrim,
            status: s.value,
            sort_by: 'updated_at',
            sort_desc: sortDesc,
          })
          return { status: s.value, items: resp.items, total: resp.total }
        }),
      )

      const nextItemsByStatus: Record<string, ReportItem[]> = {}
      for (const r of results) {
        nextItemsByStatus[r.status] = r.items
      }

      setItemsByStatus(nextItemsByStatus)
    } catch (e: any) {
      setError(e?.message ?? 'Ошибка запроса')
    } finally {
      setLoading(false)
    }
  }, [q, sortDesc])

  useEffect(() => {
    void load()
  }, [load])

  const openDetails = (reportId: string) => {
    nav(`/mod/reports/${encodeURIComponent(reportId)}`)
  }

  return (
    <div className={styles.stack}>
      <Card>
        <div className={styles.row}>
          <div>
            <h1 className={styles.title}>Обращения</h1>
          </div>
          <div className={styles.actions}>
            {!hasToken && (
              <Button variant="primary" type="button" onClick={() => nav('/mod/login')}>
                Войти
              </Button>
            )}
            <Button type="button" onClick={() => void load()} disabled={loading}>
              {loading ? 'Загрузка…' : 'Обновить'}
            </Button>
            <Button variant="primary" type="button" onClick={() => void load()} disabled={loading}>
              Применить
            </Button>
          </div>
        </div>

        <div className={styles.filters}>
          <label className={styles.field}>
            <span className={styles.label}>Поиск</span>
            <input className={styles.input} value={q} onChange={(e) => setQ(e.target.value)} placeholder="Поиск по имени или описанию..." />
          </label>
          <label className={styles.field}>
            <span className={styles.label}>Сортировать по обновлению</span>
            <select className={styles.select} value={sortDesc ? '1' : '0'} onChange={(e) => setSortDesc(e.target.value === '1')}>
              <option value="1">по убыванию</option>
              <option value="0">по возрастанию</option>
            </select>
          </label>
          <label className={styles.field}>
            <span className={styles.label}>Приоритет</span>
            <select className={styles.select} value={priorityFilter} onChange={(e) => setPriorityFilter(e.target.value)}>
              <option value="">Все</option>
              {PRIORITY_OPTIONS.map((x) => (
                <option key={x} value={x}>
                  {x}
                </option>
              ))}
              <option value={UNSET_PRIORITY}>Не задан</option>
            </select>
          </label>
          <label className={styles.field}>
            <span className={styles.label}>Влияние</span>
            <select className={styles.select} value={influenceFilter} onChange={(e) => setInfluenceFilter(e.target.value)}>
              <option value="">Все</option>
              {INFLUENCE_OPTIONS.map((x) => (
                <option key={x} value={x}>
                  {x}
                </option>
              ))}
              <option value={UNSET_INFLUENCE}>Не задано</option>
            </select>
          </label>
        </div>

        {error && (
          <div className={styles.errorBlock}>
            <Alert variant="error" role="alert">
              {error}
            </Alert>
          </div>
        )}
      </Card>

      <Card>
        <div className={styles.sections}>
          {STATUS_OPTIONS.map((s) => {
            const sectionItemsRaw = itemsByStatus[s.value] ?? []
            const sectionItems = sectionItemsRaw.filter((it) => {
              const priorityValue = (it.priority ?? '').trim()
              const influenceValue = (it.influence ?? '').trim()

              if (priorityFilter) {
                if (priorityFilter === UNSET_PRIORITY) {
                  if (priorityValue !== '') return false
                } else if (priorityValue !== priorityFilter) {
                  return false
                }
              }

              if (influenceFilter) {
                if (influenceFilter === UNSET_INFLUENCE) {
                  if (influenceValue !== '') return false
                } else if (influenceValue.toLowerCase() !== influenceFilter.toLowerCase()) {
                  return false
                }
              }

              return true
            })
            const sectionTotal = sectionItems.length

            return (
              <div
                key={s.value}
                className={`${styles.statusSection} ${dragOverStatus === s.value ? styles.statusSectionDragOver : ''}`}
                onDragOver={(e) => {
                  e.preventDefault()
                  if (updatingStatus || loading) return
                  setDragOverStatus(s.value)
                  e.dataTransfer.dropEffect = 'move'
                }}
                onDragEnter={() => {
                  if (updatingStatus || loading) return
                  setDragOverStatus(s.value)
                }}
                onDragLeave={() => {
                  setDragOverStatus((cur) => (cur === s.value ? null : cur))
                }}
                onDrop={async (e) => {
                  e.preventDefault()
                  if (updatingStatus || loading) return

                  const droppedId = e.dataTransfer.getData('text/plain')
                  const fromStatus = e.dataTransfer.getData('fromStatus')
                  if (!droppedId || !fromStatus || fromStatus === s.value) {
                    setDragOverStatus(null)
                    return
                  }

                  setUpdatingStatus(true)
                  setError(null)
                  try {
                    await changeReportStatus(droppedId, s.value)
                    await load()
                  } catch (err: any) {
                    setError(err?.message ?? 'Ошибка запроса')
                  } finally {
                    isDragRef.current = false
                    setDraggingId(null)
                    setDragOverStatus(null)
                    setUpdatingStatus(false)
                  }
                }}
              >
                <div className={styles.sectionHeader}>
                  <div className={styles.sectionTitle}>{s.label}</div>
                  <div className={styles.muted}>
                    Всего: <b>{sectionTotal}</b>
                  </div>
                </div>

                <div className={styles.table}>
                  <div className={styles.thead}>
                    <div>ID</div>
                    <div>Отправитель</div>
                    <div>Статус</div>
                    <div>Влияние</div>
                    <div>Приоритет</div>
                    <div>Обновлено</div>
                  </div>

                  {sectionItems.map((it) => {
                    const statusLabel = STATUS_OPTIONS.find((x) => x.value === it.status)?.label ?? it.status
                    const influenceLabel = (it.influence ?? '').trim() || 'Не задано'
                    const priorityLabel = (it.priority ?? '').trim() || 'Не задан'
                    const influenceClass =
                      influenceLabel === 'Крит/блокер' || influenceLabel === 'Крит/Блокер'
                        ? `${styles.badge} ${styles.influenceBlocker}`
                        : influenceLabel === 'Высокий'
                          ? `${styles.badge} ${styles.influenceHigh}`
                          : influenceLabel === 'Средний'
                            ? `${styles.badge} ${styles.influenceMedium}`
                            : influenceLabel === 'Низкий'
                              ? `${styles.badge} ${styles.influenceLow}`
                              : influenceLabel === 'Не баг а фича'
                                ? `${styles.badge} ${styles.influenceFeature}`
                                : styles.badge
                    return (
                      <div
                        key={it.id}
                        className={`${styles.trow} ${draggingId === it.id ? styles.trowDragging : ''}`}
                        draggable={!updatingStatus && !loading}
                        onDragStart={(e) => {
                          if (updatingStatus || loading) return
                          isDragRef.current = true
                          setDraggingId(it.id)
                          setDragOverStatus(s.value)
                          e.dataTransfer.setData('text/plain', it.id)
                          e.dataTransfer.setData('fromStatus', s.value)
                          e.dataTransfer.effectAllowed = 'move'
                        }}
                        onDragEnd={() => {
                          isDragRef.current = false
                          setDraggingId(null)
                          setDragOverStatus(null)
                        }}
                        onClick={(e) => {
                          if (isDragRef.current) {
                            e.preventDefault()
                            e.stopPropagation()
                            return
                          }
                          openDetails(it.id)
                        }}
                        role="button"
                        tabIndex={0}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter') openDetails(it.id)
                        }}
                      >
                        <div className={styles.mono}>{it.id}</div>
                        <div>{it.reporter_name}</div>
                        <div>
                          <span className={getStatusBadgeClass(it.status)}>{statusLabel}</span>
                        </div>
                        <div>
                          <span className={influenceClass}>{influenceLabel}</span>
                        </div>
                        <div>
                          <span
                            className={
                              priorityLabel === 'Высокий'
                                ? `${styles.badge} ${styles.priorityHigh}`
                                : priorityLabel === 'Средний'
                                  ? `${styles.badge} ${styles.priorityMedium}`
                                  : priorityLabel === 'Низкий'
                                    ? `${styles.badge} ${styles.priorityLow}`
                                    : styles.badge
                            }
                          >
                            {priorityLabel}
                          </span>
                        </div>
                        <div className={styles.muted}>{formatUnixSeconds(it.updated_at)}</div>
                      </div>
                    )
                  })}

                  {sectionItems.length === 0 && !loading && <div className={`${styles.muted} ${styles.emptyState}`}>Нет данных</div>}
                </div>
              </div>
            )
          })}
        </div>
      </Card>
    </div>
  )
}

