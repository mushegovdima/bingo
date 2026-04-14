

import { TransactionFilters, TransactionTypeFilter } from '@/hooks/useTransactions'
import s from './TransactionFiltersPanel.module.scss'

const TYPE_OPTIONS: { value: TransactionTypeFilter; label: string }[] = [
  { value: 'all', label: 'Все операции' },
  { value: 'credit', label: 'Начисления' },
  { value: 'debit', label: 'Списания' },
]

const inputCls = s.input

interface Props {
  filters: TransactionFilters
  onChange: (updated: Partial<TransactionFilters>) => void
}

/**
 * Панель фильтров для истории транзакций: тип и период.
 */
export function TransactionFiltersPanel({ filters, onChange }: Props) {
  return (
    <div className="flex flex-col sm:flex-row gap-3">
      <div className="flex flex-col gap-1 sm:w-48">
        <label className={s.label}>Тип операции</label>
        <select
          value={filters.type}
          onChange={(e) => onChange({ type: e.target.value as TransactionTypeFilter })}
          className={inputCls}
        >
          {TYPE_OPTIONS.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
      </div>

      <div className="flex flex-col gap-1 sm:w-44">
        <label className={s.label}>С</label>
        <input
          type="date"
          value={filters.dateFrom}
          onChange={(e) => onChange({ dateFrom: e.target.value })}
          className={inputCls}
        />
      </div>

      <div className="flex flex-col gap-1 sm:w-44">
        <label className={s.label}>По</label>
        <input
          type="date"
          value={filters.dateTo}
          onChange={(e) => onChange({ dateTo: e.target.value })}
          className={inputCls}
        />
      </div>
    </div>
  )
}
