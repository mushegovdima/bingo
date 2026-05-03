import { useState } from 'react'
import { Button, Chip, Skeleton } from '@heroui/react'
import { CheckCircle, XCircle, ChevronDown, ChevronUp } from 'lucide-react'
import { useAdminSubmissions } from '@/hooks/useAdminSubmissions'
import { useSeasonTasks } from '@/hooks/useSeasonTasks'
import { useApproveSubmission, useRejectSubmission } from '@/hooks/useReviewSubmission'
import { TaskSubmissionStatus } from '@/types'
import s from '../../settings.module.scss'

const statusChip: Record<TaskSubmissionStatus, { label: string; color: 'success' | 'warning' | 'danger' }> = {
  approved: { label: 'Принято',     color: 'success' },
  pending:  { label: 'На проверке', color: 'warning' },
  rejected: { label: 'Отклонено',   color: 'danger'  },
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleString('ru-RU', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

interface Props {
  seasonId: number
}

export function SubmissionsPanel({ seasonId }: Props) {
  const { data: submissions, isLoading } = useAdminSubmissions(seasonId)
  const { data: tasks = [] } = useSeasonTasks(seasonId)
  const { mutate: approve, isPending: approvePending } = useApproveSubmission()
  const { mutate: reject,  isPending: rejectPending  } = useRejectSubmission()

  const [rejectingId, setRejectingId] = useState<number | null>(null)
  const [rejectComment, setRejectComment] = useState('')
  const [rejectError, setRejectError] = useState<string | null>(null)

  const taskMap = new Map(tasks.map((t) => [t.id, t]))

  const handleApprove = (id: number) => {
    approve(id)
  }

  const handleRejectSubmit = (id: number) => {
    if (!rejectComment.trim()) {
      setRejectError('Укажите причину отклонения')
      return
    }
    reject(
      { id, comment: rejectComment.trim() },
      {
        onSuccess: () => {
          setRejectingId(null)
          setRejectComment('')
          setRejectError(null)
        },
      },
    )
  }

  if (isLoading) {
    return (
      <div className={s.skeletonList}>
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className={s.skeletonRow} />
        ))}
      </div>
    )
  }

  if (!submissions || submissions.length === 0) {
    return <p className={s.muted}>Нет заявок на проверку</p>
  }

  const pending  = submissions.filter((s) => s.status === 'pending')
  const reviewed = submissions.filter((s) => s.status !== 'pending')

  return (
    <div className="flex flex-col gap-3">
      {pending.length === 0 && (
        <p className={s.muted}>Нет новых заявок на проверку</p>
      )}

      {pending.length > 0 && (
        <div className="flex flex-col gap-2">
          <span className="text-xs font-semibold uppercase tracking-wider text-(--color-text-muted)">
            Ожидают проверки ({pending.length})
          </span>
          {pending.map((sub) => {
            const task = taskMap.get(sub.task_id)
            const isRejectOpen = rejectingId === sub.id
            return (
              <div
                key={sub.id}
                className="bg-(--color-surface) border border-(--color-border) rounded-xl p-4 flex flex-col gap-3 shadow-(--shadow-sm)"
              >
                {/* Header row */}
                <div className="flex flex-wrap items-start gap-2 justify-between">
                  <div className="flex flex-col gap-0.5 min-w-0">
                    <span className="font-semibold text-sm text-(--color-text) truncate">
                      {task?.title ?? `Задача #${sub.task_id}`}
                    </span>
                    <span className="text-xs text-(--color-text-muted)">
                      {sub.user_name ?? `User #${sub.user_id}`} · {formatDate(sub.submitted_at)}
                    </span>
                  </div>
                  <Chip size="sm" color="warning">{statusChip.pending.label}</Chip>
                </div>

                {/* User comment */}
                {sub.comment && (
                  <div className="bg-(--color-bg) rounded-lg px-3 py-2 text-sm text-(--color-text-muted)">
                    <span className="font-medium text-(--color-text)">Комментарий: </span>
                    {sub.comment}
                  </div>
                )}

                {/* Action buttons */}
                {!isRejectOpen && (
                  <div className="flex gap-2 flex-wrap">
                    <Button
                      size="sm"
                      variant="primary"
                      onPress={() => handleApprove(sub.id)}
                      isDisabled={approvePending || rejectPending}
                    >
                      <CheckCircle size={14} />
                      Принять
                    </Button>
                    <Button
                      size="sm"
                      variant="ghost"
                      onPress={() => {
                        setRejectingId(sub.id)
                        setRejectComment('')
                        setRejectError(null)
                      }}
                      isDisabled={approvePending || rejectPending}
                    >
                      <XCircle size={14} />
                      Отклонить
                    </Button>
                  </div>
                )}

                {/* Reject form */}
                {isRejectOpen && (
                  <div className="flex flex-col gap-2">
                    <label className="flex flex-col gap-1">
                      <span className="text-xs font-medium text-(--color-text)">
                        Причина отклонения <span className="text-red-500">*</span>
                      </span>
                      <textarea
                        value={rejectComment}
                        onChange={(e) => {
                          setRejectComment(e.target.value)
                          setRejectError(null)
                        }}
                        placeholder="Укажите, что нужно исправить..."
                        rows={2}
                        className="w-full rounded-lg border border-(--color-border) bg-(--color-bg) px-3 py-2 text-sm text-(--color-text) resize-none outline-none focus:border-(--color-primary) transition-colors"
                      />
                    </label>
                    {rejectError && (
                      <span className="text-xs text-red-500">{rejectError}</span>
                    )}
                    <div className="flex gap-2">
                      <Button
                        size="sm"
                        variant="danger"
                        onPress={() => handleRejectSubmit(sub.id)}
                        isDisabled={rejectPending}
                      >
                        Отклонить
                      </Button>
                      <Button
                        size="sm"
                        variant="ghost"
                        onPress={() => {
                          setRejectingId(null)
                          setRejectComment('')
                          setRejectError(null)
                        }}
                      >
                        Отмена
                      </Button>
                    </div>
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}

      {reviewed.length > 0 && (
        <ReviewedList items={reviewed} taskMap={taskMap} />
      )}
    </div>
  )
}

function ReviewedList({
  items,
  taskMap,
}: {
  items: ReturnType<typeof useAdminSubmissions>['data'] & object[]
  taskMap: Map<number, { title: string }>
}) {
  const [open, setOpen] = useState(false)

  return (
    <div className="flex flex-col gap-2">
      <button
        className="flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wider text-(--color-text-muted) hover:text-(--color-text) transition-colors self-start"
        onClick={() => setOpen((v) => !v)}
      >
        {open ? <ChevronUp size={13} /> : <ChevronDown size={13} />}
        Проверенные ({items.length})
      </button>

      {open && (
        <div className="flex flex-col gap-2">
          {items.map((sub) => {
            const task = taskMap.get(sub.task_id)
            const chip = statusChip[sub.status as TaskSubmissionStatus]
            return (
              <div
                key={sub.id}
                className="bg-(--color-surface) border border-(--color-border) rounded-xl p-4 flex flex-col gap-2 opacity-80"
              >
                <div className="flex flex-wrap items-start gap-2 justify-between">
                  <div className="flex flex-col gap-0.5 min-w-0">
                    <span className="font-semibold text-sm text-(--color-text) truncate">
                      {task?.title ?? `Задача #${sub.task_id}`}
                    </span>
                    <span className="text-xs text-(--color-text-muted)">
                      {sub.user_name ?? `User #${sub.user_id}`} · {formatDate(sub.submitted_at)}
                    </span>
                  </div>
                  <Chip size="sm" color={chip.color}>{chip.label}</Chip>
                </div>
                {sub.review_comment && (
                  <div className="text-xs text-(--color-text-muted)">
                    <span className="font-medium">Комментарий: </span>{sub.review_comment}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
