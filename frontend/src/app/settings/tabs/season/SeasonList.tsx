import { useMemo, useState } from 'react'
import type { SortDescriptor } from '@heroui/react'
import { Button, Chip, Input, Skeleton, Table, TableLayout, Virtualizer } from '@heroui/react'
import { Pencil, Plus, Trash2 } from 'lucide-react'
import { Season } from '@/types'
import { useSeasons } from '@/hooks/useSeasons'
import { useDeleteSeason } from '@/hooks/useSeasonMutations'
import { SeasonModal } from './SeasonModal'
import { ConfirmModal } from '@/components/common/ConfirmModal'
import s from '../../settings.module.scss'

interface Props {
  selectedId: number | null
  onSelect: (id: number) => void
}

function fmt(iso: string) {
  return new Date(iso).toLocaleDateString('ru-RU')
}

export function SeasonList({ selectedId, onSelect }: Props) {
  const { data: seasons, isLoading } = useSeasons()
  const { mutate: deleteSeason } = useDeleteSeason()
  const [modalSeason, setModalSeason] = useState<Season | 'new' | null>(null)
  const [deletingId, setDeletingId] = useState<number | null>(null)
  const [search, setSearch] = useState('')
  const [sortDescriptor, setSortDescriptor] = useState<SortDescriptor>({
    column: 'id',
    direction: 'descending',
  })

  const layout = useMemo(() => new TableLayout({ }), [])

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase()
    if (!q) return seasons ?? []
    return (seasons ?? []).filter((c) => c.title.toLowerCase().includes(q))
  }, [seasons, search])

  const sorted = useMemo(() => {
    const dir = sortDescriptor.direction === 'descending' ? -1 : 1
    return [...filtered].sort((a, b) => {
      switch (sortDescriptor.column) {
        case 'title': return dir * a.title.localeCompare(b.title)
        case 'start_date': return dir * (new Date(a.start_date).getTime() - new Date(b.start_date).getTime())
        case 'end_date': return dir * (new Date(a.end_date).getTime() - new Date(b.end_date).getTime())
        case 'status': return dir * (Number(a.is_active) - Number(b.is_active))
        default: return dir * (a.id - b.id)
      }
    })
  }, [filtered, sortDescriptor])

  if (isLoading) {
    return (
      <div className={s.skeletonList}>
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className={s.skeletonRow} />
        ))}
      </div>
    )
  }

  return (
    <>
      <div className={s.tableCard}>
      <div className={s.tableToolbar}>
        <Input
          placeholder="Поиск по названию…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className={s.searchInput}
        />
        <span className={s.countLabel}>{filtered.length} / {(seasons ?? []).length}</span>
        <div className='flex-1' />
        <Button variant="primary" size="sm" onPress={() => setModalSeason('new')}>
          <Plus size={15} />
          Создать
        </Button>
      </div>

      <Table className={s.usersTable} variant="primary">
        <Table.ResizableContainer>
          <Virtualizer layout={layout}>
            <Table.Content
              aria-label="Кампании"
              sortDescriptor={sortDescriptor}
              onSortChange={setSortDescriptor}
              onRowAction={(key) => onSelect(Number(key))}
            >
              <Table.Header>
                <Table.Column id="title" isRowHeader minWidth={150} allowsSorting>
                  Название <Table.ColumnResizer />
                </Table.Column>
                <Table.Column id="start_date" defaultWidth={130} allowsSorting>
                  Начало <Table.ColumnResizer />
                </Table.Column>
                <Table.Column id="end_date" defaultWidth={130} allowsSorting>
                  Конец <Table.ColumnResizer />
                </Table.Column>
                <Table.Column id="status" defaultWidth={120} allowsSorting>
                  Статус <Table.ColumnResizer />
                </Table.Column>
                <Table.Column id="actions" defaultWidth={110}>Действия</Table.Column>
              </Table.Header>
              <Table.Body items={sorted}>
                {(season) => (
                  <Table.Row
                    id={season.id}
                    style={{ cursor: 'pointer', background: selectedId === season.id ? 'var(--color-surface-2, rgba(0,0,0,.04))' : undefined }}
                  >
                    <Table.Cell>
                      <span style={{ fontWeight: 500 }}>{season.title}</span>
                    </Table.Cell>
                    <Table.Cell>{fmt(season.start_date)}</Table.Cell>
                    <Table.Cell>{fmt(season.end_date)}</Table.Cell>
                    <Table.Cell>
                      <Chip size="sm" color={season.is_active ? 'success' : 'default'}>
                        {season.is_active ? 'Активна' : 'Неактивна'}
                      </Chip>
                    </Table.Cell>
                    <Table.Cell>
                      <div className={s.actions}>
                        <Button
                          size="sm"
                          variant="ghost"
                          aria-label="Редактировать"
                          isIconOnly
                          onPress={() => setModalSeason(season)}
                        >
                          <Pencil size={14}/>
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          aria-label="Удалить"
                          isIconOnly
                          onPress={() => setDeletingId(season.id)}
                        >
                          <Trash2 size={14}/>
                        </Button>
                      </div>
                    </Table.Cell>
                  </Table.Row>
                )}
              </Table.Body>
            </Table.Content>
          </Virtualizer>
        </Table.ResizableContainer>
      </Table>
      </div>

      {modalSeason !== null && (
        <SeasonModal
          season={modalSeason === 'new' ? undefined : modalSeason}
          onClose={() => setModalSeason(null)}
        />
      )}
      {deletingId !== null && (
        <ConfirmModal
          title="Удалить кампанию"
          message="Вы уверены? Это действие нельзя отменить."
          confirmLabel="Удалить"
          variant="danger"
          onConfirm={() => deleteSeason(deletingId)}
          onClose={() => setDeletingId(null)}
        />
      )}
    </>
  )
}
