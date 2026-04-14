import { useMemo, useState } from 'react'
import { Button, Chip, Skeleton, Table, TableLayout, Virtualizer } from '@heroui/react'
import { Copy, Pencil, Plus, Trash2 } from 'lucide-react'
import { Task } from '@/types'
import { useSeasonTasks } from '@/hooks/useSeasonTasks'
import { useDeleteTask } from '@/hooks/useTaskMutations'
import { TaskModal } from './TaskModal'
import { ConfirmModal } from '@/components/common/ConfirmModal'
import s from '../../settings.module.scss'

interface Props {
  seasonId: number
}

export function TasksPanel({ seasonId }: Props) {
  const { data: tasks, isLoading } = useSeasonTasks(seasonId)
  const { mutate: deleteTask } = useDeleteTask(seasonId)
  const [modalTask, setModalTask] = useState<Task | null>(null)
  const [deletingId, setDeletingId] = useState<number | null>(null)

  const layout = useMemo(() => new TableLayout({ rowHeight: 56 }), [])

  const categories = useMemo(
    () => [...new Set((tasks ?? []).map((t) => t.category).filter(Boolean))],
    [tasks],
  )

  const sorted = useMemo(
    () => [...(tasks ?? [])].sort((a, b) => a.sort_order - b.sort_order),
    [tasks],
  )

  const copyTask = (task: Task) => {
    const data = {
      ...task,
      id: 0,
      season_id: seasonId,
      title: `${task.title} (копия)`,
      sort_order: task.sort_order + 1,
      is_active: false
    } as Task
    setModalTask(data)
  }

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
      <div className={s.tableToolbar}>
        <span className={s.countLabel}>{sorted.length} задач</span>
        <Button variant="primary" size="sm" onPress={() => setModalTask({} as Task)}>
          <Plus size={15} />
          Добавить задачу
        </Button>
      </div>

      <Table className={s.usersTable}>
        <Table.ResizableContainer>
          <Virtualizer layout={layout}>
            <Table.Content aria-label="Задачи">
              <Table.Header>
                <Table.Column id="id" defaultWidth={30} isRowHeader>
                  Id <Table.ColumnResizer />
                </Table.Column>
                <Table.Column id="title" minWidth={150}>
                  Название <Table.ColumnResizer />
                </Table.Column>
                <Table.Column id="category" defaultWidth={130}>
                  Категория <Table.ColumnResizer />
                </Table.Column>
                <Table.Column id="reward_coins" defaultWidth={100}>
                  Монеты <Table.ColumnResizer />
                </Table.Column>
                <Table.Column id="status" defaultWidth={110}>
                  Статус <Table.ColumnResizer />
                </Table.Column>
                <Table.Column id="actions" defaultWidth={90}>Действия</Table.Column>
              </Table.Header>
              <Table.Body items={sorted}>
                {(task) => (
                  <Table.Row id={task.id}>
                    <Table.Cell>
                      <span className={s.muted}>{task.id}</span>
                    </Table.Cell>
                    <Table.Cell>
                      <div>
                        <div className={s.userName}>{task.title}</div>
                        {task.description && (
                          <div className={`${s.userUsername} truncate`} title={task.description}>{task.description}</div>
                        )}
                      </div>
                    </Table.Cell>
                    <Table.Cell>
                      <Chip size="sm" color="default">{task.category || '—'}</Chip>
                    </Table.Cell>
                    <Table.Cell>
                      <span style={{ fontWeight: 600 }}>{task.reward_coins} 🪙</span>
                    </Table.Cell>
                    <Table.Cell>
                      <Chip size="sm" color={task.is_active ? 'success' : 'default'}>
                        {task.is_active ? 'Активна' : 'Неактивна'}
                      </Chip>
                    </Table.Cell>
                    <Table.Cell>
                      <div className={s.actions}>
                        <Button
                          size="sm"
                          variant="ghost"
                          aria-label="Редактировать"
                          isIconOnly
                          onPress={() => setModalTask(task)}
                        >
                          <Pencil size={14} />
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          aria-label="Копировать"
                          isIconOnly
                          onPress={() => copyTask(task)}
                        >
                          <Copy size={14} />
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          aria-label="Удалить"
                          isIconOnly
                          onPress={() => setDeletingId(task.id)}
                        >
                          <Trash2 size={14} />
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

      {modalTask !== null && (
        <TaskModal
          seasonId={seasonId}
          task={modalTask}
          categories={categories}
          onClose={() => setModalTask(null)}
        />
      )}
      {deletingId !== null && (
        <ConfirmModal
          title="Удалить задачу"
          message="Вы уверены? Это действие нельзя отменить."
          confirmLabel="Удалить"
          variant="danger"
          onConfirm={() => deleteTask(deletingId)}
          onClose={() => setDeletingId(null)}
        />
      )}
    </>
  )
}
