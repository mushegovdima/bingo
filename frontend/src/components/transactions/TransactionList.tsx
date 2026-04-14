

import { CardRoot, CardContent, Spinner } from '@heroui/react'
import { Transaction } from '@/types'
import { TransactionItem } from './TransactionItem'
import { TransactionFiltersPanel } from './TransactionFiltersPanel'
import { TransactionFilters } from '@/hooks/useTransactions'
import s from './TransactionList.module.scss'

interface Props {
  transactions: Transaction[] | undefined
  isLoading: boolean
  filters: TransactionFilters
  onFilterChange: (updated: Partial<TransactionFilters>) => void
}

/**
 * Полный блок истории: фильтры + список транзакций.
 */
export function TransactionList({
  transactions,
  isLoading,
  filters,
  onFilterChange,
}: Props) {
  return (
    <div className="flex flex-col gap-4">
      <TransactionFiltersPanel filters={filters} onChange={onFilterChange} />

      <CardRoot className={s.card}>
        <CardContent className="p-0 sm:p-2">
          {isLoading && (
            <div className="flex items-center justify-center py-12">
              <Spinner className="text-primary" />
            </div>
          )}

          {!isLoading && (!transactions || transactions.length === 0) && (
            <p className={s.empty}>
              Операций не найдено
            </p>
          )}

          {!isLoading && transactions && transactions.length > 0 && (
            <div className="px-4">
              {transactions.map((t) => (
                <TransactionItem key={t.id} transaction={t} />
              ))}
            </div>
          )}
        </CardContent>
      </CardRoot>
    </div>
  )
}
