import { Route, Routes } from 'react-router-dom'
import { PublicReportPage } from '@/pages/PublicReportPage/PublicReportPage'
import { ModeratorLoginPage } from '@/pages/ModeratorLoginPage/ModeratorLoginPage'
import { ModeratorReportsPage } from '@/pages/ModeratorReportsPage/ModeratorReportsPage'
import { ModeratorReportDetailPage } from '@/pages/ModeratorReportDetailPage/ModeratorReportDetailPage'
import { Card } from '@/shared/ui/Card/Card'

export function AppRouter() {
  return (
    <Routes>
      <Route path="/" element={<PublicReportPage />} />
      <Route path="/mod/login" element={<ModeratorLoginPage />} />
      <Route path="/mod/reports" element={<ModeratorReportsPage />} />
      <Route path="/mod/reports/:id" element={<ModeratorReportDetailPage />} />
      <Route path="*" element={<Card>Not found</Card>} />
    </Routes>
  )
}

