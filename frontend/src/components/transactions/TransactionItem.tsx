

import { Chip } from '@heroui/react'
import { Transaction } from '@/types'
import s from './TransactionItem.module.scss'

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
    <div className={s.item}>
      {/* Date */}
      <span className={s.date}>
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
      <span className={s.description}>
        {transaction.ref_title || REASON_LABEL[transaction.reason]}
      </span>

      {/* Amount */}
      <span
        className={`${s.amount} ${isCredit ? s.credit : s.debit}`}
      >
        {isCredit ? '+' : ''}
        {transaction.amount.toLocaleString('ru-RU')} KC
      </span>
    </div>
  )
}
