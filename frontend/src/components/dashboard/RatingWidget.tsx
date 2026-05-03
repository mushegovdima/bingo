
import { CardContent, CardRoot, SkeletonRoot } from '@heroui/react'
import { LeaderboardEntry } from '@/types'
import { useLeaderboard } from '@/hooks/useLeaderboard'

interface Props {
  seasonId: number
}

function EntryRow({ entry, isCurrent }: { entry: LeaderboardEntry; isCurrent: boolean }) {
  return (
    <div
      className={
        'flex items-center gap-2 rounded-lg px-2 py-1.5 transition-colors ' +
        (isCurrent
          ? 'bg-(--color-primary)/10 ring-1 ring-(--color-primary)/30'
          : 'opacity-50')
      }
    >
      <span
        className={
          'w-6 text-center text-xs font-bold tabular-nums shrink-0 ' +
          (isCurrent ? 'text-(--color-secondary)' : 'text-(--color-text-muted)')
        }
      >
        #{entry.position}
      </span>

      <div
        className={
          'w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold shrink-0 ' +
          (isCurrent
            ? 'bg-(--color-primary) text-white'
            : 'bg-(--color-border) text-(--color-text-muted)')
        }
      >
        {entry.name.charAt(0).toUpperCase()}
      </div>

      <span
        className={
          'flex-1 truncate text-xs ' +
          (isCurrent ? 'font-semibold text-(--color-text)' : 'text-(--color-text-muted)')
        }
      >
        {entry.name}
      </span>

      <span
        className={
          'text-xs font-bold tabular-nums shrink-0 ' +
          (isCurrent ? 'text-(--color-coin)' : 'text-(--color-text-subtle)')
        }
      >
        {entry.balance.toLocaleString('ru-RU')} баллов
      </span>
    </div>
  )
}

export function RatingWidget({ seasonId }: Props) {
  const { data: entries = [], isLoading } = useLeaderboard(seasonId)
  const current = entries.find((e) => e.is_current)

  return (
    <CardRoot className="bg-(--color-surface) border border-(--color-border) shadow-(--shadow-md) transition-[box-shadow,transform] duration-200 hover:shadow-(--shadow-xl) hover:-translate-y-1 cursor-default">
      <CardContent className="p-4 flex flex-col gap-2">
        <div className="flex items-center justify-between">
          <span className="text-xs font-semibold uppercase tracking-widest text-(--color-text-muted)">
            Место в рейтинге
          </span>
          {current && (
            <span className="text-lg font-extrabold tabular-nums text-(--color-secondary)">
              #{current.position}
            </span>
          )}
        </div>

        {isLoading ? (
          <div className="flex flex-col gap-1">
            <SkeletonRoot className="h-8 w-full rounded-lg" />
            <SkeletonRoot className="h-8 w-full rounded-lg" />
            <SkeletonRoot className="h-8 w-full rounded-lg" />
          </div>
        ) : entries.length === 0 ? (
          <span className="text-xs text-(--color-text-subtle) py-1">рейтинг скоро появится</span>
        ) : (
          <div className="flex flex-col gap-0.5">
            {entries.map((entry) => (
              <EntryRow key={entry.user_id} entry={entry} isCurrent={entry.is_current} />
            ))}
          </div>
        )}
      </CardContent>
    </CardRoot>
  )
}
