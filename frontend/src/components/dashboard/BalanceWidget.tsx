

import { CardContent, CardRoot, SkeletonRoot } from '@heroui/react'
import s from '@/styles/widgets.module.scss'
import { useNavigate } from 'react-router-dom'

interface Props {
  coins: number
  isLoading: boolean
  seasonId: number
}

/**
 * Большой виджет с текущим балансом Bingo Coin.
 */
export function BalanceWidget({ coins, isLoading, seasonId }: Props) {
  const navigate = useNavigate()

  return (
    <CardRoot className={s.card} onClick={() => navigate(`/d/${seasonId}/balance`)}>
      <CardContent className="p-6 flex flex-col gap-2 cursor-pointer">
        <span className={s.label}>
          Bingo Coin
        </span>

        {isLoading ? (
          <div className="flex gap-2 items-end">
            <SkeletonRoot className="h-16 w-40 rounded-lg" />
            <SkeletonRoot className="h-10 w-12 rounded-lg" />
          </div>
        ) : (
          <div className="flex items-baseline gap-2">
            <span
              className={`text-6xl font-extrabold tabular-nums ${s.coinValue}`}
            >
              {coins.toLocaleString('ru-RU')}
            </span>
            <span
              className={`text-3xl font-bold ${s.coinValue}`}
              aria-hidden
            >
              KC
            </span>
          </div>
        )}

        <span className={s.footer}>текущий баланс</span>
      </CardContent>
    </CardRoot>
  )
}
