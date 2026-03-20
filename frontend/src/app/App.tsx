import { AppLayout } from './layout/AppLayout'
import { AppRouter } from './router'
import { ErrorBoundary } from '@/shared/ui/ErrorBoundary/ErrorBoundary'

export function App() {
  return (
    <AppLayout>
      <ErrorBoundary>
        <AppRouter />
      </ErrorBoundary>
    </AppLayout>
  )
}

