import type { PropsWithChildren } from 'react'
import styles from './Card.module.css'

export function Card(props: PropsWithChildren<{ className?: string }>) {
  const { className, children } = props
  return <section className={className ? `${styles.card} ${className}` : styles.card}>{children}</section>
}

