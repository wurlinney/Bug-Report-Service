import type { PropsWithChildren } from 'react'
import styles from './Alert.module.css'

export type AlertVariant = 'error' | 'ok'

export function Alert(props: PropsWithChildren<{ variant: AlertVariant; role?: 'alert' | 'status' }>) {
  const { variant, role, children } = props
  const cls = [styles.alert, variant === 'error' ? styles.error : styles.ok].join(' ')
  return (
    <div className={cls} role={role}>
      {children}
    </div>
  )
}

