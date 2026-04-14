import { useState } from 'react'
import { Badge, Button, Chip, Tabs } from '@heroui/react'
import { ArrowLeft, Pencil } from 'lucide-react'
import { useSeasons } from '@/hooks/useSeasons'
import { SeasonModal } from './SeasonModal'
import { TasksPanel } from './TasksPanel'
import s from '../../settings.module.scss'

interface Props {
  seasonId: number
}

export function SeasonDetail({ seasonId }: Props) {
  const { data: seasons } = useSeasons()
  const season = seasons?.find((c) => c.id === seasonId)
  const [editOpen, setEditOpen] = useState(false)

  return (
    <div className={s.detailSection}>
      <div className={s.detailHeader}>
        {season && (
          <div className={s.detailMeta}>
            <span className={s.detailTitle}>{season.title}</span>
            <Chip size="sm" color={season.is_active ? 'success' : 'default'}>
              {season.is_active ? 'Активна' : 'Неактивна'}
            </Chip>
            <span className={s.muted}>
              {new Date(season.start_date).toLocaleDateString('ru-RU')}
              {' — '}
              {new Date(season.end_date).toLocaleDateString('ru-RU')}
            </span>
            <Button size="sm" variant="ghost" isIconOnly aria-label="Редактировать" onPress={() => setEditOpen(true)}>
              <Pencil size={14} />
            </Button>
          </div>
        )}
      </div>

      {editOpen && season && (
        <SeasonModal season={season} onClose={() => setEditOpen(false)} />
      )}

      <Tabs defaultSelectedKey="tasks">
        <Tabs.ListContainer>
          <Tabs.List>
            <Tabs.Tab id="tasks">Задачи</Tabs.Tab>
            <Tabs.Tab id="events" isDisabled>Мероприятия</Tabs.Tab>
          </Tabs.List>
        </Tabs.ListContainer>

        <Tabs.Panel id="tasks" className={s.tabPanel}>
          <TasksPanel seasonId={seasonId} />
        </Tabs.Panel>
        <Tabs.Panel id="events" className={s.tabPanel}>
          <p className={s.muted}>Мероприятия будут доступны позже.</p>
        </Tabs.Panel>
      </Tabs>
    </div>
  )
}
