

import { CardContent, CardRoot, ProgressBarRoot, ProgressBarTrack, ProgressBarFill, SkeletonRoot } from '@heroui/react'
import { THEME } from '@/lib/theme'
import s from '@/styles/widgets.module.scss'

interface Props {
  coins: number
  isLoading: boolean
}

/**
 * Прогресс-бар: сколько KC осталось до главного приза.
 */
export function ProgressWidget({ coins, isLoading }: Props) {
  const { mainPrizeTarget, mainPrizeLabel } = THEME.progress
  const percentage = Math.min((coins / mainPrizeTarget) * 100, 100)
  const remaining = Math.max(mainPrizeTarget - coins, 0)

  return (
    <CardRoot className={s.card}>
      <CardContent className="p-6 flex flex-col gap-4">
        <div className="flex items-center justify-between">
          <span className={s.label}>
            До главного приза
          </span>
          <span className={s.primaryText}>
            {mainPrizeLabel}
          </span>
        </div>

        {isLoading ? (
          <SkeletonRoot className="h-3 w-full rounded-full" />
        ) : (
          <ProgressBarRoot
            value={percentage}
            aria-label={`Прогресс до приза: ${Math.round(percentage)}%`}
            className="w-full"
          >
            <ProgressBarTrack className={s.progressTrack}>
              <ProgressBarFill
                className={s.progressFill}
                style={{ width: `${percentage}%` }}
              />
            </ProgressBarTrack>
          </ProgressBarRoot>
        )}

        <div className="flex items-center justify-between">
          {isLoading ? (
            <SkeletonRoot className="h-4 w-36 rounded" />
          ) : (
            <span className={s.progressStats}>
              {coins.toLocaleString('ru-RU')} /{' '}
              {mainPrizeTarget.toLocaleString('ru-RU')} KC
            </span>
          )}

          {!isLoading && remaining > 0 && (
            <span className={s.progressRemaining}>
              ещё {remaining.toLocaleString('ru-RU')} KC
            </span>
          )}

          {!isLoading && remaining === 0 && (
            <span className={s.progressAchieved}>
              Приз получен! 🎉
            </span>
          )}
        </div>
      </CardContent>
    </CardRoot>
  )
}
