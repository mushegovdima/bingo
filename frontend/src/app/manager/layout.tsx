import { Outlet } from 'react-router-dom'
import { AppLayout } from '@/components/layout/AppLayout'
import { ProtectedPage } from '@/components/common/ProtectedPage'

export default function ManagerLayout() {
  return (
    <ProtectedPage requireManager>
      <AppLayout>
        <Outlet />
      </AppLayout>
    </ProtectedPage>
  )
}
