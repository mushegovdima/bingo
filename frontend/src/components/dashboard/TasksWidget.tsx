
import { useNavigate } from 'react-router-dom'
import { CardContent, CardRoot, SkeletonRoot } from '@heroui/react'
import { useSeasonTasks } from '@/hooks/useSeasonTasks'
import { useSubmissions } from '@/hooks/useSubmissions'
import { TaskSubmissionStatus } from '@/types'

interface Props {
  seasonId: number
}

const statusBadge: Record<TaskSubmissionStatus | 'none', { label: string; className: string }> = {
  approved: { label: 'Выполнено', className: 'bg-green-100 text-green-700' },
  pending:  { label: 'На проверке', className: 'bg-yellow-100 text-yellow-700' },
  rejected: { label: 'Отклонено', className: 'bg-red-100 text-red-700' },
  none:     { label: 'Не выполнено', className: 'bg-(--color-border) text-(--color-text-muted)' },
}

export function TasksWidget({ seasonId }: Props) {
  const navigate = useNavigate()
  const { data: tasks = [], isLoading: tasksLoading } = useSeasonTasks(seasonId)
  const { data: submissions, isLoading: submissionsLoading } = useSubmissions(seasonId)
  const isLoading = tasksLoading || submissionsLoading

  const submissionByTaskId = new Map(
    (submissions?.all ?? []).map((s) => [s.task_id, s]),
  )

  const activeTasks = tasks.filter((t) => t.is_active)
  const approvedCount = submissions?.approvedCount ?? 0

  return (
    <CardRoot className="bg-(--color-surface) border border-(--color-border) shadow-(--shadow-md)">
      <CardContent className="p-6 flex flex-col gap-4">
        {/* Bingo counter — one line */}
        <div className="flex items-center justify-between">
          <span className="text-xs font-semibold uppercase tracking-widest text-(--color-text-muted)">
            Закрыто БИНГО
          </span>
          {isLoading ? (
            <SkeletonRoot className="h-6 w-20 rounded-lg" />
          ) : (
            <div className="flex items-baseline gap-1">
              <span className="text-2xl font-extrabold tabular-nums text-(--color-success)">{approvedCount}</span>
              <span className="text-base font-semibold text-slate-400">/{activeTasks.length}</span>
              <span className="text-xs text-(--color-text-subtle) ml-1">задач</span>
            </div>
          )}
        </div>

        {isLoading ? (
          <div className="flex flex-col gap-3">
            {[1, 2, 3].map((i) => (
              <SkeletonRoot key={i} className="h-12 w-full rounded-xl" />
            ))}
          </div>
        ) : activeTasks.length === 0 ? (
          <p className="text-(--color-text-subtle) text-sm">Задач пока нет</p>
        ) : (
          <ul className="flex flex-col gap-2">
            {activeTasks.map((task) => {
              const sub = submissionByTaskId.get(task.id)
              const statusKey = (sub?.status ?? 'none') as TaskSubmissionStatus | 'none'
              const badge = statusBadge[statusKey]

              return (
                <li key={task.id}>
                  <button
                    onClick={() => navigate(`/d/${seasonId}/task/${task.id}`)}
                    className="w-full text-left flex items-center justify-between gap-3 px-4 py-3 rounded-xl border border-(--color-border) transition-colors duration-150 cursor-pointer hover:bg-(--color-bg) active:scale-[0.99]"
                  >
                    <div className="flex flex-col gap-0.5 min-w-0">
                      <span className="font-medium text-(--color-text) text-sm truncate">
                        {task.title}
                      </span>
                      {task.reward_coins > 0 && (
                        <span className="text-xs text-(--color-text-muted)">
                          +{task.reward_coins} баллов
                        </span>
                      )}
                    </div>
                    <span
                      className={`shrink-0 text-xs font-medium px-2 py-1 rounded-full ${badge.className}`}
                    >
                      {badge.label}
                    </span>
                  </button>
                </li>
              )
            })}
          </ul>
        )}
      </CardContent>
    </CardRoot>
  )
}
