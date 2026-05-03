

import { CardContent, CardRoot, SkeletonRoot } from '@heroui/react'
import { useSubmissions } from '@/hooks/useSubmissions'
import { useSeasonTasks } from '@/hooks/useSeasonTasks'

interface Props {
  seasonId: number
}

export function BingoWidget({ seasonId }: Props) {
  const { data: submissions, isLoading: submissionsLoading } = useSubmissions(seasonId)
  const { data: tasks = [], isLoading: tasksLoading } = useSeasonTasks(seasonId)
  const isLoading = submissionsLoading || tasksLoading
  const approvedCount = submissions?.approvedCount ?? 0
  const totalCount = tasks.filter((t) => t.is_active).length
  return (
    <CardRoot className="bg-(--color-surface) border border-(--color-border) shadow-(--shadow-md) transition-[box-shadow,transform] duration-200 hover:shadow-(--shadow-xl) hover:-translate-y-1 cursor-default">
      <CardContent className="p-6 flex flex-col gap-2">
        <span className="text-xs font-semibold uppercase tracking-widest text-(--color-text-muted)">
          Закрыто БИНГО
        </span>

        {isLoading ? (
          <div className="flex gap-1 items-end">
            <SkeletonRoot className="h-14 w-16 rounded-lg" />
            <SkeletonRoot className="h-8 w-12 rounded-lg" />
          </div>
        ) : (
          <div className="flex items-baseline gap-1">
            <span className="text-5xl font-extrabold tabular-nums text-(--color-success)">
              {approvedCount}
            </span>
            <span className="text-2xl font-semibold text-slate-400">
              /{totalCount}
            </span>
          </div>
        )}

        <span className="text-xs text-(--color-text-subtle)">задач выполнено</span>
      </CardContent>
    </CardRoot>
  )
}
