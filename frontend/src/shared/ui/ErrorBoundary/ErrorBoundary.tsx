import type { PropsWithChildren } from 'react'
import React from 'react'
import { Card } from '@/shared/ui/Card/Card'

type State = { error: unknown }

export class ErrorBoundary extends React.Component<PropsWithChildren, State> {
  state: State = { error: null }

  static getDerivedStateFromError(error: unknown): State {
    return { error }
  }

  override componentDidCatch(error: unknown) {
    // eslint-disable-next-line no-console
    console.error('UI crashed:', error)
  }

  override render() {
    if (!this.state.error) return this.props.children

    const message =
      this.state.error instanceof Error
        ? `${this.state.error.name}: ${this.state.error.message}`
        : String(this.state.error)

    return (
      <Card>
        <h1 style={{ margin: 0, fontSize: 18 }}>Ошибка в UI</h1>
        <p style={{ opacity: 0.8 }}>Открой DevTools → Console, там будет подробность.</p>
        <pre style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>{message}</pre>
      </Card>
    )
  }
}

