import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { modLogin } from '@/features/moderator/api/moderatorApi'
import { Card } from '@/shared/ui/Card/Card'
import { Button } from '@/shared/ui/Button/Button'
import { Alert } from '@/shared/ui/Alert/Alert'
import styles from './ModeratorLoginPage.module.css'

export function ModeratorLoginPage() {
  const nav = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  function toUserErrorMessage(err: unknown): string {
    const code = String((err as any)?.code ?? '')
    const message = String((err as any)?.message ?? '').toLowerCase()
    const status = Number((err as any)?.status ?? 0)

    if (code === 'invalid_credentials' || message.includes('invalid credentials')) {
      return 'Неверный email или пароль. Проверьте данные и попробуйте снова.'
    }
    if (status === 401) {
      return 'Не удалось выполнить вход. Проверьте данные и попробуйте еще раз.'
    }
    if (status === 403) {
      return 'Доступ запрещен.'
    }

    return 'Не удалось выполнить вход. Попробуйте позже.'
  }

  return (
    <div className={styles.page}>
      <Card>
        <div className={styles.row}>
          <div>
            <h1 className={styles.title}>Вход для модератора</h1>
          </div>
        </div>

        <div className={styles.form}>
          <label className={styles.field}>
            <span className={styles.label}>Email</span>
            <input
              className={styles.input}
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="Введите свой email..."
              autoComplete="username"
            />
          </label>

          <label className={styles.field}>
            <span className={styles.label}>Пароль</span>
            <div className={styles.passwordWrap}>
              <input
                className={styles.input}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                type={showPassword ? 'text' : 'password'}
                placeholder="Введите свой пароль..."
                autoComplete="current-password"
              />
              <button
                type="button"
                className={styles.passwordToggle}
                aria-label={showPassword ? 'Скрыть пароль' : 'Показать пароль'}
                onClick={() => setShowPassword((v) => !v)}
              >
                {showPassword ? (
                  <svg width="32" height="32" viewBox="0 0 24 24" aria-hidden="true">
                    <path
                      d="M3 3l18 18"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    />
                    <path
                      d="M10.58 10.58a2 2 0 0 0 2.83 2.83"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    />
                    <path
                      d="M9.88 5.08A10.8 10.8 0 0 1 12 5c7 0 10 7 10 7a18.5 18.5 0 0 1-3.02 4.28"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    />
                    <path
                      d="M6.11 6.11A18.6 18.6 0 0 0 2 12s3 7 10 7c1.12 0 2.16-.2 3.11-.55"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    />
                  </svg>
                ) : (
                  <svg width="32" height="32" viewBox="0 0 24 24" aria-hidden="true">
                    <path
                      d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7S2 12 2 12z"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    />
                    <circle cx="12" cy="12" r="3" fill="none" stroke="currentColor" strokeWidth="2" />
                  </svg>
                )}
              </button>
            </div>
          </label>

          <div className={styles.actions}>
            <Button
              variant="primary"
              className={styles.submitBtn}
              disabled={submitting || email.trim() === '' || password === ''}
              onClick={async () => {
                setSubmitting(true)
                setError(null)
                try {
                  await modLogin({ email: email.trim(), password })
                  nav('/mod/reports')
                } catch (e: any) {
                  setError(toUserErrorMessage(e))
                } finally {
                  setSubmitting(false)
                }
              }}
            >
              {submitting ? 'Вход…' : 'Войти'}
            </Button>
          </div>

          {error && (
            <Alert variant="error" role="alert">
              {error}
            </Alert>
          )}
        </div>
      </Card>
    </div>
  )
}

