import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import { Tabs } from '@heroui/react'
import { AppLayout } from '@/components/layout/AppLayout'
import { ProtectedPage } from '@/components/common/ProtectedPage'
import s from './settings.module.scss'

const TABS = { USERS: 'users', SEASONS: 'seasons' }

/**
 * Shared layout for all /admin/* routes.
 * Renders the heading + tab navigation above each child's Outlet.
 */
export default function AdminLayout() {
  const navigate = useNavigate()
  const { pathname } = useLocation()
  const selectedTab = pathname.includes('/seasons') ? TABS.SEASONS : TABS.USERS

  return (
    <ProtectedPage requireManager>
      <AppLayout>
        <div className={s.page}>
          <Tabs
            variant="primary"
            selectedKey={selectedTab}
            onSelectionChange={(key) => navigate(`/admin/${key}`)}
          >
            <div className="flex gap-2 flex-wrap">
              <h1 className={s.heading}>Настройки</h1>
              <Tabs.ListContainer>
                <Tabs.List>
                  <Tabs.Tab id={TABS.USERS}>Пользователи</Tabs.Tab>
                  <Tabs.Tab id={TABS.SEASONS}>Кампании</Tabs.Tab>
                </Tabs.List>
              </Tabs.ListContainer>
            </div>
          </Tabs>

          <Outlet />
        </div>
      </AppLayout>
    </ProtectedPage>
  )
}
