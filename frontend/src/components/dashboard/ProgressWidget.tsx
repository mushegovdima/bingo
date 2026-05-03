

import { CardContent, CardRoot, ProgressBarRoot, ProgressBarTrack, ProgressBarFill, SkeletonRoot } from '@heroui/react'
import { THEME } from '@/lib/theme'
import { useBalance } from '@/hooks/useBalance'

interface Props {
  seasonId: number
}

export function ProgressWidget({ seasonId }: Props) {
  const { data: balance, isLoading } = useBalance(seasonId)
  const coins = balance?.balance ?? 0
  const { mainPrizeTarget, mainPrizeLabel } = THEME.progress
  const percentage = Math.min((coins / mainPrizeTarget) * 100, 100)
  const remaining = Math.max(mainPrizeTarget - coins, 0)

  return (
    <CardRoot className="bg-(--color-surface) border border-(--color-border) shadow-(--shadow-md) transition-[box-shadow,transform] duration-200 hover:shadow-(--shadow-xl) hover:-translate-y-1 cursor-default">
      <CardContent className="p-6 flex flex-col gap-4">
        <div className="flex items-center justify-between">
          <span className="text-xs font-semibold uppercase tracking-widest text-(--color-text-muted)">
            До главного приза
          </span>
          <span className="text-(--color-primary) text-sm font-semibold">
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
            <ProgressBarTrack className="h-3 w-full rounded-full bg-(--color-border)">
              <ProgressBarFill
                className="h-full rounded-full bg-(--color-primary) transition-[width] duration-300"
                style={{ width: `${percentage}%` }}
              />
            </ProgressBarTrack>
          </ProgressBarRoot>
        )}

        <div className="flex items-center justify-between">
          {isLoading ? (
            <SkeletonRoot className="h-4 w-36 rounded" />
          ) : (
            <span className="text-(--color-text-muted) text-sm">
              {coins.toLocaleString('ru-RU')} /{' '}
              {mainPrizeTarget.toLocaleString('ru-RU')} баллов
            </span>
          )}

          {!isLoading && remaining > 0 && (
            <span className="text-(--color-text-subtle) text-xs">
              ещё {remaining.toLocaleString('ru-RU')} баллов
            </span>
          )}

          {!isLoading && remaining === 0 && (
            <span className="text-(--color-success) font-semibold text-xs">
              Приз получен! 🎉
            </span>
          )}
        </div>
      </CardContent>
    </CardRoot>
  )
}
