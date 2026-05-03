
import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Chip } from '@heroui/react'
import { CheckCircle, XCircle } from 'lucide-react'
import { useSeasonTasks } from '@/hooks/useSeasonTasks'
import { useSubmissions } from '@/hooks/useSubmissions'
import { useSubmitTask } from '@/hooks/useSubmitTask'
import { useAdminSubmissions } from '@/hooks/useAdminSubmissions'
import { useApproveSubmission, useRejectSubmission } from '@/hooks/useReviewSubmission'
import { useAuth } from '@/hooks/useAuth'
import type { TaskSubmission } from '@/types'

function formatDate(iso: string) {
  return new Date(iso).toLocaleString('ru-RU')
}

const STATUS_LABEL: Record<string, string> = {
  approved: 'Принято',
  pending: 'На проверке',
  rejected: 'Отклонено',
}
const STATUS_COLOR: Record<string, 'success' | 'warning' | 'danger'> = {
  approved: 'success',
  pending: 'warning',
  rejected: 'danger',
}

function SubmissionReviewRow({ sub }: { sub: TaskSubmission }) {
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
    <div className="bg-(--color-bg) border border-(--color-border) rounded-xl p-4 flex flex-col gap-3">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="flex flex-col gap-0.5">
          <span className="text-sm font-semibold text-(--color-text)">
            {sub.user_name ?? `User #${sub.user_id}`}
          </span>
          <span className="text-xs text-(--color-text-muted)">{formatDate(sub.submitted_at)}</span>
        </div>
        <Chip size="sm" color={STATUS_COLOR[sub.status]}>{STATUS_LABEL[sub.status]}</Chip>
      </div>

      {sub.comment && (
        <div className="text-sm text-(--color-text-muted) bg-(--color-surface) rounded-lg px-3 py-2">
          <span className="font-medium text-(--color-text)">Комментарий: </span>
          {sub.comment}
        </div>
      )}

      {sub.status === 'pending' && !rejectOpen && (
        <div className="flex gap-2 flex-wrap">
          <button
            onClick={() => approve(sub.id)}
            disabled={approvePending || rejectPending}
            className="flex items-center gap-1.5 text-sm font-semibold px-3 py-1.5 rounded-lg bg-(--color-success)/10 text-(--color-success) hover:bg-(--color-success)/20 transition-colors disabled:opacity-50"
          >
            <CheckCircle size={14} /> Принять
          </button>
          <button
            onClick={() => { setRejectOpen(true); setComment(''); setErr(null) }}
            disabled={approvePending || rejectPending}
            className="flex items-center gap-1.5 text-sm font-semibold px-3 py-1.5 rounded-lg bg-(--color-danger)/10 text-(--color-danger) hover:bg-(--color-danger)/20 transition-colors disabled:opacity-50"
          >
            <XCircle size={14} /> Отклонить
          </button>
        </div>
      )}

      {sub.status === 'pending' && rejectOpen && (
        <div className="flex flex-col gap-2">
          <textarea
            value={comment}
            onChange={(e) => { setComment(e.target.value); setErr(null) }}
            placeholder="Причина отклонения..."
            rows={2}
            className="w-full rounded-lg border border-(--color-border) bg-(--color-surface) px-3 py-2 text-sm text-(--color-text) resize-none outline-none focus:border-(--color-primary) transition-colors"
          />
          {err && <span className="text-xs text-red-500">{err}</span>}
          <div className="flex gap-2">
            <button
              onClick={handleReject}
              disabled={rejectPending}
              className="text-sm font-semibold px-3 py-1.5 rounded-lg bg-(--color-danger) text-white hover:opacity-90 transition-opacity disabled:opacity-50"
            >
              Отклонить
            </button>
            <button
              onClick={() => setRejectOpen(false)}
              className="text-sm px-3 py-1.5 rounded-lg text-(--color-text-muted) hover:text-(--color-text) transition-colors"
            >
              Отмена
            </button>
          </div>
        </div>
      )}

      {sub.status !== 'pending' && sub.review_comment && (
        <div className="text-sm text-(--color-text-muted) border-t border-(--color-border) pt-2">
          <span className="font-medium text-(--color-text)">Ответ проверяющего: </span>
          {sub.review_comment}
        </div>
      )}
    </div>
  )
}

function ManagerReviewPanel({ submissions }: { submissions: TaskSubmission[] }) {
  const pending = submissions.filter((s) => s.status === 'pending')
  const reviewed = submissions.filter((s) => s.status !== 'pending')

  return (
    <div className="bg-(--color-surface) border border-(--color-border) rounded-2xl p-5 shadow-(--shadow-md) flex flex-col gap-3">
      <span className="text-xs font-semibold uppercase tracking-widest text-(--color-text-muted)">
        Проверка заявок
        {pending.length > 0 && (
          <span className="ml-2 inline-flex items-center justify-center w-5 h-5 rounded-full bg-(--color-warning)/20 text-(--color-warning) text-[10px] font-bold">
            {pending.length}
          </span>
        )}
      </span>

      {pending.length > 0 && (
        <div className="flex flex-col gap-2">
          {pending.map((sub) => <SubmissionReviewRow key={sub.id} sub={sub} />)}
        </div>
      )}

      {reviewed.length > 0 && (
        <details className="group" open={pending.length === 0}>
          <summary className="text-xs text-(--color-text-muted) cursor-pointer list-none flex items-center gap-1 select-none">
            <span className="group-open:hidden">▶</span>
            <span className="hidden group-open:inline">▼</span>
            Проверено ({reviewed.length})
          </summary>
          <div className="flex flex-col gap-2 mt-2">
            {reviewed.map((sub) => <SubmissionReviewRow key={sub.id} sub={sub} />)}
          </div>
        </details>
      )}
    </div>
  )
}

export default function TaskDetailPage() {
  const { seasonId: seasonIdStr, taskId: taskIdStr } = useParams<{
    seasonId: string
    taskId: string
  }>()
  const seasonId = Number(seasonIdStr)
  const taskId = Number(taskIdStr)
  const navigate = useNavigate()
  const { isManager } = useAuth()

  const { data: tasks = [], isLoading: tasksLoading } = useSeasonTasks(seasonId)
  const { data: submissions, isLoading: submissionsLoading } = useSubmissions(seasonId)
  const { mutateAsync: submit, isPending } = useSubmitTask()

  // Manager-only: all submissions for this season, filtered by taskId
  const { data: allSubmissions = [] } = useAdminSubmissions(isManager ? seasonId : undefined)
  const taskSubmissions = allSubmissions.filter((s) => s.task_id === taskId)

  const [comment, setComment] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitted, setSubmitted] = useState(false)

  const task = tasks.find((t) => t.id === taskId)
  const submission = submissions?.all.find((s) => s.task_id === taskId)

  const isLoading = tasksLoading || submissionsLoading

  const handleSubmit = async () => {
    setError(null)
    if (!comment.trim()) {
      setError('Комментарий обязателен')
      return
    }
    try {
      await submit({ task_id: taskId, comment: comment.trim() })
      setSubmitted(true)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Ошибка при отправке')
    }
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <span className="text-(--color-text-muted)">Загрузка...</span>
      </div>
    )
  }

  if (!task) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <span className="text-(--color-text-muted)">Задача не найдена</span>
      </div>
    )
  }

  const isRejected = submission?.status === 'rejected'
  const canSubmit = !submission || isRejected

  return (
    <div className="bg-(--color-bg) px-4 pt-6 pb-10">
      <div className="w-full max-w-lg mx-auto flex flex-col gap-4">
        <button
          onClick={() => navigate(-1)}
          className="text-sm text-(--color-text-muted) hover:text-(--color-text) transition-colors self-start"
        >
          ← Назад
        </button>

        {/* Task description card */}
        <div className="bg-(--color-surface) border border-(--color-border) rounded-2xl p-5 shadow-(--shadow-md) flex flex-col gap-2">
          <span className="text-xs font-semibold uppercase tracking-widest text-(--color-text-muted)">
            {task.category}
          </span>
          <h1 className="text-lg font-bold text-(--color-text)">{task.title}</h1>
          {task.description && (
            <p className="text-sm text-(--color-text-muted)">{task.description}</p>
          )}
          {task.reward_coins > 0 && (
            <span className="text-sm text-(--color-primary) font-semibold">
              +{task.reward_coins} баллов за выполнение
            </span>
          )}
        </div>

        {/* Fresh submit success */}
        {submitted && !submission && (
          <div className="rounded-2xl bg-yellow-50 border border-yellow-200 p-5 text-sm text-yellow-700 shadow-(--shadow-md)">
            ⏳ Задача отправлена на проверку
          </div>
        )}

        {/* Submit form card */}
        {canSubmit && !submitted && (
          <div className="bg-(--color-surface) border border-(--color-border) rounded-2xl p-5 shadow-(--shadow-md) flex flex-col gap-3">
            {isRejected && (
              <p className="text-xs text-(--color-text-muted)">Задача отклонена — вы можете отправить повторно</p>
            )}
            <label className="flex flex-col gap-1.5">
              <span className="text-sm font-medium text-(--color-text)">
                Комментарий <span className="text-red-500">*</span>
              </span>
              <textarea
                value={comment}
                onChange={(e) => setComment(e.target.value)}
                rows={3}
                placeholder="Опишите, что вы сделали..."
                className="w-full rounded-xl border border-(--color-border) bg-(--color-bg) px-3 py-2.5 text-base text-(--color-text) placeholder:text-(--color-text-subtle) focus:outline-none focus:ring-2 focus:ring-(--color-primary) resize-none"
              />
            </label>
            {error && <p className="text-sm text-red-500">{error}</p>}
            <button
              onClick={handleSubmit}
              disabled={isPending}
              className="w-full rounded-xl bg-(--color-primary) text-white font-semibold py-3.5 text-base hover:opacity-90 active:scale-[0.98] transition-all disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isPending ? 'Отправка...' : 'Выполнено'}
            </button>
          </div>
        )}

        {/* Manager review panel */}
        {isManager && taskSubmissions.length > 0 && (
          <ManagerReviewPanel submissions={taskSubmissions} />
        )}
        {isManager && taskSubmissions.length === 0 && (
          <div className="bg-(--color-surface) border border-(--color-border) rounded-2xl p-5 shadow-(--shadow-md)">
            <span className="text-xs font-semibold uppercase tracking-widest text-(--color-text-muted)">
              Проверка заявок
            </span>
            <p className="text-sm text-(--color-text-subtle) mt-2">Нет заявок по этой задаче</p>
          </div>
        )}
      </div>
    </div>
  )
}

