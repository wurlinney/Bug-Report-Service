import type { PropsWithChildren } from 'react'
import { Link, NavLink, useLocation, useNavigate } from 'react-router-dom'
import { clearTokens, readAccessToken } from '@/shared/auth/tokenStorage'
import styles from './AppLayout.module.css'

export function AppLayout(props: PropsWithChildren) {
  const nav = useNavigate()
  const location = useLocation()
  const isAuthed = typeof window !== 'undefined' && Boolean(readAccessToken())
  const onModerReports = location.pathname.startsWith('/mod/reports')

  return (
    <div className={styles.shell}>
      <header className={styles.header}>
        <div className={styles.headerRow}>
          <Link to="/" className={styles.brand}>
            Фасад: форма обратной связи
          </Link>
          <nav className={styles.nav}>
            <NavLink
              to="/"
              end
              className={({ isActive }) => (isActive ? `${styles.navLink} ${styles.navLinkActive}` : styles.navLink)}
            >
              Обращение
            </NavLink>
            {isAuthed ? (
              <>
                <button
                  type="button"
                  className={onModerReports ? `${styles.navButton} ${styles.navButtonActive}` : styles.navButton}
                  onClick={() => {
                    nav('/mod/reports')
                  }}
                >
                  Список обращений
                </button>
                <button
                  type="button"
                  className={styles.navButton}
                  onClick={() => {
                    clearTokens()
                    nav('/mod/login')
                  }}
                >
                  Выйти
                </button>
              </>
            ) : (
              <NavLink
                to="/mod/login"
                className={({ isActive }) => (isActive ? `${styles.navLink} ${styles.navLinkActive}` : styles.navLink)}
              >
                Модератор
              </NavLink>
            )}
          </nav>
        </div>
      </header>
      <main className={styles.main}>{props.children}</main>
    </div>
  )
}

