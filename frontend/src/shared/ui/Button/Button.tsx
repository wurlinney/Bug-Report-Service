import type { ButtonHTMLAttributes } from 'react'
import styles from './Button.module.css'

export type ButtonVariant = 'default' | 'primary'

export function Button(props: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: ButtonVariant }) {
  const { className, variant = 'default', ...rest } = props
  const variantClass = variant === 'primary' ? styles.buttonPrimary : ''
  const cls = [styles.button, variantClass, className].filter(Boolean).join(' ')
  return <button {...rest} className={cls} />
}

