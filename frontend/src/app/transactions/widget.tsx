import { useNavigate, useParams } from 'react-router-dom'
import { Button, Chip, SkeletonRoot } from '@heroui/react'
import { ArrowLeft } from 'lucide-react'
import { TransactionList } from '@/components/transactions/TransactionList'
import { useTransactions } from '@/hooks/useTransactions'
import { useBalance } from '@/hooks/useBalance'
import s from './transactions.module.scss'

export default function TransactionsWidgetPage() {
  const { seasonId: seasonIdStr } = useParams<{ seasonId: string }>()
  const seasonId = Number(seasonIdStr)
  const navigate = useNavigate()
  const { data: transactions, isLoading, filters, setFilters } = useTransactions(seasonId)
  const { data: balance, isLoading: balanceLoading } = useBalance(seasonId)

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-3">
        <Button
          variant="ghost"
          size="sm"
          onPress={() => navigate(`/d/${seasonId}`)}
        >
          <ArrowLeft size={14} />
          Назад
        </Button>
        <h1 className={s.title}>Баланс</h1>
        {balanceLoading
          ? <SkeletonRoot className="h-6 w-20 rounded-full" />
          : <Chip size="lg" color="accent" variant="soft">{(balance?.balance ?? 0).toLocaleString('ru-RU')} KC</Chip>
        }
      </div>

      <TransactionList
        transactions={transactions}
        isLoading={isLoading}
        filters={filters}
        onFilterChange={(partial) => setFilters((prev) => ({ ...prev, ...partial }))}
      />
    </div>
  )
}
