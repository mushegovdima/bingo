

import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { balanceApi } from '@/lib/api/balance'
import { Transaction } from '@/types'

export type TransactionTypeFilter = 'all' | 'credit' | 'debit'

export interface TransactionFilters {
  type: TransactionTypeFilter
  dateFrom: string
  dateTo: string
}

const DEFAULT_FILTERS: TransactionFilters = {
  type: 'all',
  dateFrom: '',
  dateTo: '',
}

function applyFilters(
  transactions: Transaction[],
  filters: TransactionFilters,
): Transaction[] {
  return transactions.filter((t) => {
    if (filters.type === 'credit' && t.amount <= 0) return false
    if (filters.type === 'debit' && t.amount >= 0) return false

    if (filters.dateFrom) {
      if (new Date(t.created_at) < new Date(filters.dateFrom)) return false
    }

    if (filters.dateTo) {
      const end = new Date(filters.dateTo)
      end.setHours(23, 59, 59, 999)
      if (new Date(t.created_at) > end) return false
    }

    return true
  })
}

export function useTransactions(seasonId: number | undefined) {
  const [filters, setFilters] = useState<TransactionFilters>(DEFAULT_FILTERS)

  // Fetch raw list once. Filtering is purely client-side via useMemo.
  const { data: raw, isLoading, error } = useQuery({
    queryKey: ['transactions', seasonId],
    queryFn: () => balanceApi.getTransactions(seasonId!),
    enabled: !!seasonId,
  })

  const data = useMemo(
    () => (raw ? applyFilters(raw, filters) : undefined),
    [raw, filters],
  )

  return { data, isLoading, error, filters, setFilters }
}
