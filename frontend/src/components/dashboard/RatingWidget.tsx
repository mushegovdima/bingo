

import { CardContent, CardRoot, SkeletonRoot } from '@heroui/react'
import s from '@/styles/widgets.module.scss'

interface Props {
  /** Rank position, or undefined if not available yet */
  position?: number
  isLoading: boolean
}

/**
 * Место в рейтинге.
 * TODO: бэкенд не имеет endpoint-а /leaderboard.
 * Когда он будет добавлен — передавайте position из соответствующего хука.
 */
export function RatingWidget({ position, isLoading }: Props) {
  return (
    <CardRoot className={s.card}>
      <CardContent className="p-6 flex flex-col gap-2">
        <span className={s.label}>
          Место в рейтинге
        </span>

        {isLoading ? (
          <SkeletonRoot className="h-14 w-24 rounded-lg" />
        ) : (
          <div className="flex items-baseline gap-1">
            {position !== undefined ? (
              <span className={`text-5xl font-extrabold tabular-nums ${s.ratingValue}`}>
                #{position}
              </span>
            ) : (
              <span className={`text-4xl font-bold ${s.ratingEmpty}`}>—</span>
            )}
          </div>
        )}

        <span className={s.footer}>
          {position !== undefined ? 'общий рейтинг' : 'рейтинг скоро появится'}
        </span>
      </CardContent>
    </CardRoot>
  )
}
