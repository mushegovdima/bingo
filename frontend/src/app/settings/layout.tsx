import { Outlet, useMatch, useNavigate } from 'react-router-dom'
import { Tabs } from '@heroui/react'
import { AppLayout } from '@/components/layout/AppLayout'
import { ProtectedPage } from '@/components/common/ProtectedPage'

const TABS = { USERS: 'users', TEMPLATES: 'templates', SEASONS: 'seasons' }

/**
 * Shared layout for all /admin/* routes.
 * Renders the heading + tab navigation above each child's Outlet.
 */
export default function AdminLayout() {
  const navigate = useNavigate()
  const matchTemplates = useMatch('/admin/templates')
  const matchSeasons = useMatch('/admin/seasons') || useMatch('/admin/seasons/:seasonId')

  const selectedTab = matchTemplates ? TABS.TEMPLATES : matchSeasons ? TABS.SEASONS : TABS.USERS

  return (
    <ProtectedPage requireManager>
      <AppLayout>
        <div className="flex flex-col gap-4">
          <Tabs
            variant="primary"
            selectedKey={selectedTab}
            onSelectionChange={(key) => navigate(`/admin/${key}`)}
          >
            <div className="flex gap-2 flex-wrap">
              <h1 className="m-0 text-2xl font-bold text-(--color-text)">Настройки</h1>
              <Tabs.ListContainer>
                <Tabs.List>
                  <Tabs.Tab id={TABS.USERS}>Пользователи</Tabs.Tab>
                  <Tabs.Tab id={TABS.SEASONS}>Сезоны</Tabs.Tab>
                  <Tabs.Tab id={TABS.TEMPLATES}>Шаблоны уведомлений</Tabs.Tab>
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
