

import { CardContent, CardRoot, SkeletonRoot } from '@heroui/react'
import { useNavigate } from 'react-router-dom'
import { useBalance } from '@/hooks/useBalance'

interface Props {
  seasonId: number
}

export function BalanceWidget({ seasonId }: Props) {
  const navigate = useNavigate()
  const { data: balance, isLoading } = useBalance(seasonId)
  const coins = balance?.balance ?? 0

  return (
    <CardRoot
      className="bg-(--color-surface) border border-(--color-border) shadow-(--shadow-md) transition-[box-shadow,transform] duration-200 hover:shadow-(--shadow-xl) hover:-translate-y-1 cursor-default"
      onClick={() => navigate(`/d/${seasonId}/balance`)}
    >
      <CardContent className="p-6 flex flex-col gap-2 cursor-pointer">
        <span className="text-xs font-semibold uppercase tracking-widest text-(--color-text-muted)">
          Текущий баланс
        </span>

        {isLoading ? (
          <div className="flex gap-2 items-end">
            <SkeletonRoot className="h-16 w-40 rounded-lg" />
            <SkeletonRoot className="h-10 w-12 rounded-lg" />
          </div>
        ) : (
          <div className="flex items-baseline gap-2">
            <span className="text-4xl sm:text-6xl font-extrabold tabular-nums text-(--color-coin)">
              {coins.toLocaleString('ru-RU')}
            </span>
            <span className="text-2xl sm:text-3xl font-bold text-(--color-coin)" aria-hidden>
              баллов
            </span>
          </div>
        )}

        <span className="text-xs text-(--color-text-subtle)">текущий баланс</span>
      </CardContent>
    </CardRoot>
  )
}
