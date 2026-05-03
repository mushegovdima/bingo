import { useRef, useState, useCallback } from 'react'
import { Button, Chip, Skeleton } from '@heroui/react'
import { Check, ChevronRight, Clock, Loader2, RotateCcw } from 'lucide-react'
import { useTemplates, useTemplateHistory } from '@/hooks/useTemplates'
import { useUpdateTemplate } from '@/hooks/useUpdateTemplate'
import { Template } from '@/types'
import s from '../settings.module.scss'

const CODENAME_LABELS: Record<string, string> = {
  season_available: 'Новый сезон',
  task_approved: 'Задача выполнена',
  claim_submitted: 'Заявка принята',
  claim_completed: 'Приз получен',
  claim_cancelled: 'Заявка отменена',
}

// ─── History panel ────────────────────────────────────────────────────────────

interface HistoryPanelProps {
  codename: string
  onRestore: (body: string) => void
}

function HistoryPanel({ codename, onRestore }: HistoryPanelProps) {
  const { data: history, isLoading } = useTemplateHistory(codename)

  if (isLoading) return <Skeleton className={s.skeletonRow} />
  if (!history?.length) return <p className={s.emptyHint}>История изменений пуста</p>

  return (
    <div className={s.historyList}>
      {history.map((h) => (
        <div key={h.id} className={s.historyItem}>
          <div className={s.historyMeta}>
            <Clock size={12} />
            {new Date(h.changed_at).toLocaleString('ru-RU')}
          </div>
          <pre className={s.historyBody}>{h.body}</pre>
          <Button
            size="sm"
            variant="ghost"
            onPress={() => onRestore(h.body)}
          >
            <RotateCcw size={13} /> Восстановить
          </Button>
        </div>
      ))}
    </div>
  )
}

// ─── Editor ───────────────────────────────────────────────────────────────────

interface EditorProps {
  template: Template
}

function Editor({ template }: EditorProps) {
  const [body, setBody] = useState(template.body)
  const [saved, setSaved] = useState(false)
  const [showHistory, setShowHistory] = useState(false)
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const { mutate: save, isPending } = useUpdateTemplate()

  const vars = template.vars ?? []

  // Insert {{key}} at current cursor position
  const insertVar = useCallback((key: string) => {
    const el = textareaRef.current
    if (!el) return

    const start = el.selectionStart
    const end = el.selectionEnd
    const placeholder = `{{${key}}}`
    const next = body.slice(0, start) + placeholder + body.slice(end)
    setBody(next)
    setSaved(false)

    // Restore focus and move caret after inserted text
    requestAnimationFrame(() => {
      el.focus()
      const pos = start + placeholder.length
      el.setSelectionRange(pos, pos)
    })
  }, [body])

  const handleSave = () => {
    save({ codename: template.codename, body }, {
      onSuccess: () => {
        setSaved(true)
        setTimeout(() => setSaved(false), 2000)
      },
    })
  }

  const handleRestore = (restoredBody: string) => {
    setBody(restoredBody)
    setSaved(false)
    setShowHistory(false)
  }

  const isDirty = body !== template.body

  return (
    <div className={s.editor}>
      <div className={s.editorHeader}>
        <div>
          <h2 className={s.editorTitle}>
            {CODENAME_LABELS[template.codename] ?? template.codename}
          </h2>
          <code className={s.editorCodename}>{template.codename}</code>
        </div>
        <div className={s.editorActions}>
          <Button
            size="sm"
            variant="ghost"
            onPress={() => setShowHistory((v) => !v)}
          >
            <Clock size={14} />
            История
          </Button>
          <Button
            size="sm"
            isDisabled={!isDirty || isPending}
            onPress={handleSave}
          >
            {isPending
              ? <><Loader2 size={14} className={s.spin} /> Сохранение…</>
              : saved
                ? <><Check size={14} /> Сохранено</>
                : 'Сохранить'
            }
          </Button>
        </div>
      </div>

      {vars.length > 0 && (
        <div className={s.varsRow}>
          <span className={s.varsLabel}>Переменные:</span>
          {vars.map((v) => (
            <button
              key={v.key}
              className={s.varChipBtn}
              type="button"
              title={v.label}
              onClick={() => insertVar(v.key)}
            >
              <Chip size="sm" className={s.varChip}>
                {`{{${v.key}}}`}
              </Chip>
            </button>
          ))}
        </div>
      )}

      <textarea
        ref={textareaRef}
        className={s.bodyTextarea}
        value={body}
        onChange={(e) => { setBody(e.target.value); setSaved(false) }}
        rows={12}
        spellCheck={false}
      />

      {showHistory && (
        <div className={s.historyPanel}>
          <h3 className={s.historyHeading}>История изменений</h3>
          <HistoryPanel codename={template.codename} onRestore={handleRestore} />
        </div>
      )}
    </div>
  )
}

// ─── TemplatesTab ──────────────────────────────────────────────────────────────

export function TemplatesTab() {
  const { data: templates, isLoading } = useTemplates()
  const [selected, setSelected] = useState<string | null>(null)

  const selectedTemplate = templates?.find((t) => t.codename === selected)
    ?? (templates?.[0] ?? null)

  if (isLoading) {
    return (
      <div className={s.skeletonList}>
        {Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={i} className={s.skeletonRow} />
        ))}
      </div>
    )
  }

  return (
    <div className={s.templatesLayout}>
      {/* Sidebar */}
      <nav className={s.templatesSidebar}>
        {(templates ?? []).map((t) => (
          <button
            key={t.codename}
            className={[
              s.templatesSidebarItem,
              (selected ?? templates?.[0]?.codename) === t.codename
                ? s.templatesSidebarItemActive
                : '',
            ].join(' ')}
            onClick={() => setSelected(t.codename)}
          >
            <span>{CODENAME_LABELS[t.codename] ?? t.codename}</span>
            <ChevronRight size={14} className={s.templatesSidebarChevron} />
          </button>
        ))}
      </nav>

      {/* Editor */}
      <div className={s.templatesMain}>
        {selectedTemplate
          ? <Editor key={selectedTemplate.codename} template={selectedTemplate} />
          : <p className={s.emptyHint}>Шаблоны не найдены</p>
        }
      </div>
    </div>
  )
}
