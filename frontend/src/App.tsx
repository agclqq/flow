import { Navigate, Route, Routes } from 'react-router-dom'
import AppLayout from './layouts/AppLayout'
import InstanceDetailPage from './pages/InstanceDetailPage'
import InstanceListPage from './pages/InstanceListPage'
import TemplateEditPage from './pages/TemplateEditPage'
import TemplateListPage from './pages/TemplateListPage'

export default function App() {
  return (
    <Routes>
      <Route element={<AppLayout />}>
        <Route index element={<Navigate to="/templates" replace />} />
        <Route path="templates" element={<TemplateListPage />} />
        <Route path="templates/new" element={<TemplateEditPage />} />
        <Route path="templates/:id/edit" element={<TemplateEditPage />} />
        <Route path="templates/:id/instances" element={<InstanceListPage />} />
        <Route path="templates/:templateId/instances/:instanceId" element={<InstanceDetailPage />} />
      </Route>
    </Routes>
  )
}
