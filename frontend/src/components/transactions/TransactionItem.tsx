

import { Chip } from '@heroui/react'
import { Transaction } from '@/types'

const REASON_LABEL: Record<Transaction['reason'], string> = {
  event: 'Мероприятие',
  task: 'Задача БИНГО',
  manual: 'Ручная корректировка',
  reward: 'Приз',
}

interface Props {
  transaction: Transaction
}

/**
 * Одна строка в истории операций.
 */
export function TransactionItem({ transaction }: Props) {
  const isCredit = transaction.amount > 0
  const formattedDate = new Date(transaction.created_at).toLocaleDateString(
    'ru-RU',
    { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' },
  )

  return (
    <div className="flex items-center gap-4 py-3 border-b border-(--color-border) last:border-b-0">
      {/* Date */}
      <span className="text-(--color-text-muted) text-xs tabular-nums w-32 shrink-0">
        {formattedDate}
      </span>

      {/* Type badge */}
      <Chip
        size="sm"
        color={isCredit ? 'success' : 'danger'}
        variant="soft"
        className="shrink-0"
      >
        {isCredit ? 'Начисление' : 'Списание'}
      </Chip>

      {/* Description */}
      <span className="text-(--color-text-secondary) text-sm flex-1 overflow-hidden text-ellipsis whitespace-nowrap">
        {transaction.ref_title || REASON_LABEL[transaction.reason]}
      </span>

      {/* Amount */}
      <span className={`text-sm font-bold tabular-nums shrink-0 ${isCredit ? 'text-(--color-success)' : 'text-(--color-danger)'}`}>
        {isCredit ? '+' : ''}
        {transaction.amount.toLocaleString('ru-RU')} C
      </span>
    </div>
  )
}
