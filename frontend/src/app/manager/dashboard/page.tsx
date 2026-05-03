import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Button, CardContent, CardRoot, Chip, Skeleton } from '@heroui/react'
import { CheckCircle, XCircle } from 'lucide-react'
import { useAdminSubmissions } from '@/hooks/useAdminSubmissions'
import { useSeasonTasks } from '@/hooks/useSeasonTasks'
import { useApproveSubmission, useRejectSubmission } from '@/hooks/useReviewSubmission'
import { useFullLeaderboard } from '@/hooks/useFullLeaderboard'
import type { LeaderboardEntry, TaskSubmission, Task } from '@/types'

// ─── Leaderboard ──────────────────────────────────────────────────────────────

function LeaderboardRow({ entry }: { entry: LeaderboardEntry }) {
  const medal = entry.position === 1 ? '🥇' : entry.position === 2 ? '🥈' : entry.position === 3 ? '🥉' : null
  return (
    <div className="flex items-center gap-3 rounded-lg px-2 py-1.5">
      <span className="w-7 text-center text-sm font-bold tabular-nums shrink-0 text-(--color-text-muted)">
        {medal ?? `#${entry.position}`}
      </span>
      <div className="w-7 h-7 rounded-full bg-(--color-primary)/15 flex items-center justify-center text-xs font-bold text-(--color-primary) shrink-0">
        {entry.name.charAt(0).toUpperCase()}
      </div>
      <span className="flex-1 truncate text-sm text-(--color-text)">{entry.name}</span>
      <span className="text-sm font-bold tabular-nums text-(--color-coin) shrink-0">
        {entry.balance.toLocaleString('ru-RU')}
      </span>
    </div>
  )
}

function ManagerLeaderboard({ seasonId }: { seasonId: number }) {
  const { data: entries = [], isLoading } = useFullLeaderboard(seasonId)

  return (
    <CardRoot className="bg-(--color-surface) border border-(--color-border) shadow-(--shadow-md)">
      <CardContent className="p-4 flex flex-col gap-2">
        <div className="flex items-center justify-between mb-1">
          <span className="text-xs font-semibold uppercase tracking-widest text-(--color-text-muted)">
            Рейтинг участников
          </span>
          <span className="text-xs text-(--color-text-subtle)">{entries.length} чел.</span>
        </div>
        {isLoading ? (
          <div className="flex flex-col gap-1">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-8 w-full rounded-lg" />
            ))}
          </div>
        ) : entries.length === 0 ? (
          <span className="text-sm text-(--color-text-subtle) py-2">Участников пока нет</span>
        ) : (
          <div className="flex flex-col gap-0.5 max-h-[480px] overflow-y-auto overscroll-contain">
            {entries.map((e) => (
              <LeaderboardRow key={e.user_id} entry={e} />
            ))}
          </div>
        )}
      </CardContent>
    </CardRoot>
  )
}

// ─── Pending submissions ──────────────────────────────────────────────────────

function formatDate(iso: string) {
  return new Date(iso).toLocaleString('ru-RU', {
    day: '2-digit', month: '2-digit', year: 'numeric',
    hour: '2-digit', minute: '2-digit',
  })
}

function SubmissionCard({
  sub,
  task,
}: {
  sub: TaskSubmission
  task: Task | undefined
}) {
  const navigate = useNavigate()
  const { mutate: approve, isPending: approvePending } = useApproveSubmission()
  const { mutate: reject, isPending: rejectPending } = useRejectSubmission()

  const [rejectOpen, setRejectOpen] = useState(false)
  const [comment, setComment] = useState('')
  const [err, setErr] = useState<string | null>(null)

  const handleReject = () => {
    if (!comment.trim()) { setErr('Укажите причину'); return }
    reject(
      { id: sub.id, comment: comment.trim() },
      { onSuccess: () => { setRejectOpen(false); setComment(''); setErr(null) } },
    )
  }

  return (
    <div className="bg-(--color-surface) border border-(--color-border) rounded-xl p-4 flex flex-col gap-3 shadow-(--shadow-sm)">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="flex flex-col gap-0.5 min-w-0">
          <button
            className="font-semibold text-sm text-(--color-text) truncate text-left hover:text-(--color-primary) transition-colors"
            onClick={() => task && navigate(`/manager/seasons/${task.season_id}`)}
          >
            {task?.title ?? `Задача #${sub.task_id}`}
          </button>
          <span className="text-xs text-(--color-text-muted)">
            {sub.user_name ?? `User #${sub.user_id}`} · {formatDate(sub.submitted_at)}
          </span>
        </div>
        <Chip size="sm" color="warning">На проверке</Chip>
      </div>

      {sub.comment && (
        <div className="bg-(--color-bg) rounded-lg px-3 py-2 text-sm text-(--color-text-muted)">
          <span className="font-medium text-(--color-text)">Комментарий: </span>
          {sub.comment}
        </div>
      )}

      {!rejectOpen && (
        <div className="flex gap-2 flex-wrap">
          <Button
            size="sm"
            variant="primary"
            onPress={() => approve(sub.id)}
            isDisabled={approvePending || rejectPending}
          >
            <CheckCircle size={14} />
            Принять
          </Button>
          <Button
            size="sm"
            variant="ghost"
            onPress={() => { setRejectOpen(true); setComment(''); setErr(null) }}
            isDisabled={approvePending || rejectPending}
          >
            <XCircle size={14} />
            Отклонить
          </Button>
        </div>
      )}

      {rejectOpen && (
        <div className="flex flex-col gap-2">
          <label className="flex flex-col gap-1">
            <span className="text-xs font-medium text-(--color-text)">
              Причина <span className="text-red-500">*</span>
            </span>
            <textarea
              value={comment}
              onChange={(e) => { setComment(e.target.value); setErr(null) }}
              placeholder="Укажите, что нужно исправить..."
              rows={2}
              className="w-full rounded-lg border border-(--color-border) bg-(--color-bg) px-3 py-2 text-sm text-(--color-text) resize-none outline-none focus:border-(--color-primary) transition-colors"
            />
          </label>
          {err && <span className="text-xs text-red-500">{err}</span>}
          <div className="flex gap-2">
            <Button size="sm" variant="danger" onPress={handleReject} isDisabled={rejectPending}>
              Отклонить
            </Button>
            <Button size="sm" variant="ghost" onPress={() => setRejectOpen(false)}>
              Отмена
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}

function PendingSubmissions({ seasonId }: { seasonId: number }) {
  const { data: submissions, isLoading } = useAdminSubmissions(seasonId)
  const { data: tasks = [] } = useSeasonTasks(seasonId)

  const taskMap = new Map(tasks.map((t) => [t.id, t]))
  const pending = submissions?.filter((s) => s.status === 'pending') ?? []

  return (
    <CardRoot className="bg-(--color-surface) border border-(--color-border) shadow-(--shadow-md)">
      <CardContent className="p-4 flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <span className="text-xs font-semibold uppercase tracking-widest text-(--color-text-muted)">
            Требуют проверки
          </span>
          {pending.length > 0 && (
            <Chip size="sm" color="warning">{pending.length}</Chip>
          )}
        </div>

        {isLoading ? (
          <div className="flex flex-col gap-2">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-20 w-full rounded-xl" />
            ))}
          </div>
        ) : pending.length === 0 ? (
          <p className="text-sm text-(--color-text-subtle) py-2">Новых заявок нет</p>
        ) : (
          <div className="flex flex-col gap-2">
            {pending.map((sub) => (
              <SubmissionCard key={sub.id} sub={sub} task={taskMap.get(sub.task_id)} />
            ))}
          </div>
        )}
      </CardContent>
    </CardRoot>
  )
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export default function ManagerDashboardPage() {
  const { seasonId: seasonIdStr } = useParams<{ seasonId: string }>()
  const seasonId = Number(seasonIdStr)

  return (
    <div className="grid grid-cols-1 lg:grid-cols-[1fr_320px] gap-4 items-start">
      <PendingSubmissions seasonId={seasonId} />
      <ManagerLeaderboard seasonId={seasonId} />
    </div>
  )
}
