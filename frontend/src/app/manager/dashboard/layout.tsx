import { Outlet, useNavigate, useParams } from 'react-router-dom'
import { Chip, Popover, Spinner } from '@heroui/react'
import { ChevronDown, Settings, Users, ListChecks, CalendarDays } from 'lucide-react'
import { AppLayout } from '@/components/layout/AppLayout'
import { ProtectedPage } from '@/components/common/ProtectedPage'
import { useSeasons } from '@/hooks/useSeasons'
import { useSeasonTasks } from '@/hooks/useSeasonTasks'
import { useFullLeaderboard } from '@/hooks/useFullLeaderboard'

function ManagerSeasonPicker() {
  const { seasonId: seasonIdStr } = useParams<{ seasonId: string }>()
  const seasonId = Number(seasonIdStr)
  const navigate = useNavigate()
  const { data: seasons = [], isLoading } = useSeasons()
  const { data: tasks = [] } = useSeasonTasks(seasonId)
  const { data: members = [] } = useFullLeaderboard(seasonId)

  // мероприятия — не реализованы, заглушка
  const eventsCount = 0

  const current = seasons.find((s) => s.id === seasonId)

  return (
    <div className="flex items-center justify-between gap-3 flex-wrap">
      {/* Season picker + gear */}
      <div className="flex items-center gap-1.5">
        <Popover>
          <Popover.Trigger>
            <Chip size="lg" variant="primary">
              <Chip.Label>
                <span className="text-(--color-primary) font-semibold">
                  {current?.title ?? `Сезон #${seasonId}`}
                </span>
              </Chip.Label>
              <ChevronDown size={16} className="text-(--color-primary)" />
            </Chip>
          </Popover.Trigger>
          <Popover.Content className="min-w-56">
            <Popover.Dialog>
              {isLoading ? (
                <div className="flex justify-center p-4">
                  <Spinner size="sm" />
                </div>
              ) : seasons.length === 0 ? (
                <p className="p-3 text-sm text-(--color-text-muted)">Нет сезонов</p>
              ) : (
                <div className="flex flex-col gap-0.5 max-h-72 overflow-y-auto">
                  {seasons.map((s) => (
                    <button
                      key={s.id}
                      onClick={() => navigate(`/manager/d/${s.id}`)}
                      className={`w-full text-left px-3 py-2 rounded-lg text-sm transition-colors ${
                        s.id === seasonId
                          ? 'bg-(--color-primary)/10 text-(--color-primary) font-semibold'
                          : 'text-(--color-text) hover:bg-(--color-border)'
                      }`}
                    >
                      <span className="flex items-center justify-between gap-2">
                        {s.title}
                        {s.is_active && (
                          <span className="text-xs text-(--color-success) font-normal">активный</span>
                        )}
                      </span>
                    </button>
                  ))}
                </div>
              )}
            </Popover.Dialog>
          </Popover.Content>
        </Popover>

        <button
          onClick={() => navigate(`/admin/seasons/${seasonId}`)}
          title="Настройки сезона"
          className="p-1.5 rounded-lg text-(--color-text-muted) hover:text-(--color-text) hover:bg-(--color-border) transition-colors"
        >
          <Settings size={16} />
        </button>
      </div>

      {/* Stats chips */}
      <div className="flex items-center gap-2 flex-wrap">
        <Chip size="sm" variant="secondary">
          <Users size={12} />
          <Chip.Label>{members.length} участников</Chip.Label>
        </Chip>
        <Chip size="sm" variant="secondary">
          <ListChecks size={12} />
          <Chip.Label>{tasks.length} задач</Chip.Label>
        </Chip>
        <Chip size="sm" variant="secondary">
          <CalendarDays size={12} />
          <Chip.Label>{eventsCount} мероприятий</Chip.Label>
        </Chip>
      </div>
    </div>
  )
}

export default function ManagerDashboardLayout() {
  return (
    <ProtectedPage requireManager>
      <AppLayout>
        <div className="flex flex-col gap-4">
          <ManagerSeasonPicker />
          <Outlet />
        </div>
      </AppLayout>
    </ProtectedPage>
  )
}
