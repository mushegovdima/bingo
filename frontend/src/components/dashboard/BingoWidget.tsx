

import { CardContent, CardRoot, SkeletonRoot } from '@heroui/react'
import s from '@/styles/widgets.module.scss'

interface Props {
  approvedCount: number
  totalCount: number
  isLoading: boolean
}

/**
 * Счётчик выполненных задач БИНГО (submissions со статусом approved).
 */
export function BingoWidget({ approvedCount, totalCount, isLoading }: Props) {
  return (
    <CardRoot className={s.card}>
      <CardContent className="p-6 flex flex-col gap-2">
        <span className={s.label}>
          Закрыто БИНГО
        </span>

        {isLoading ? (
          <div className="flex gap-1 items-end">
            <SkeletonRoot className="h-14 w-16 rounded-lg" />
            <SkeletonRoot className="h-8 w-12 rounded-lg" />
          </div>
        ) : (
          <div className="flex items-baseline gap-1">
            <span className={`text-5xl font-extrabold tabular-nums ${s.bingoValue}`}>
              {approvedCount}
            </span>
          <span className="text-2xl font-semibold text-slate-400">
              /{totalCount}
            </span>
          </div>
        )}

        <span className={s.footer}>задач выполнено</span>
      </CardContent>
    </CardRoot>
  )
}
