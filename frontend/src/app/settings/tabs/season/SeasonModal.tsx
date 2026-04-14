import { type FormEvent, useState } from 'react'
import { Button, FieldError, Form, Input, Label, Modal, Switch, TextField, useOverlayState } from '@heroui/react'
import { Season } from '@/types'
import { useCreateSeason, useUpdateSeason } from '@/hooks/useSeasonMutations'
import type { CreateSeasonRequest, UpdateSeasonRequest } from '@/lib/api/seasons'
import s from '../../settings.module.scss'

interface Props {
  season?: Season
  onClose: () => void
}

function toInputDate(iso: string): string {
  return iso ? iso.slice(0, 10) : ''
}

function toRFC3339(dateStr: string): string {
  return dateStr ? `${dateStr}T00:00:00Z` : ''
}

export function SeasonModal({ season, onClose }: Props) {
  const isEdit = !!season
  const [title, setTitle] = useState(season?.title ?? '')
  const [startDate, setStartDate] = useState(toInputDate(season?.start_date ?? ''))
  const [endDate, setEndDate] = useState(toInputDate(season?.end_date ?? ''))
  const [isActive, setIsActive] = useState(season?.is_active ?? false)

  const { mutate: create, isPending: isCreating } = useCreateSeason()
  const { mutate: update, isPending: isUpdating } = useUpdateSeason()
  const isPending = isCreating || isUpdating

  const state = useOverlayState({
    defaultOpen: true,
    onOpenChange: (open) => { if (!open) onClose() },
  })

  const handleSave = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    if (isEdit) {
      const data: UpdateSeasonRequest = { title, start_date: toRFC3339(startDate), end_date: toRFC3339(endDate), is_active: isActive }
      update({ id: season.id, data }, { onSuccess: state.close })
    } else {
      const data: CreateSeasonRequest = { title, start_date: toRFC3339(startDate), end_date: toRFC3339(endDate), is_active: isActive }
      create(data, { onSuccess: state.close })
    }
  }

  return (
    <Modal state={state}>
      <Modal.Backdrop isDismissable>
        <Modal.Container size="lg">
          <Modal.Dialog>
            <Modal.Header>
              <Modal.Heading>{isEdit ? 'Редактировать кампанию' : 'Новая кампания'}</Modal.Heading>
              <Modal.CloseTrigger />
            </Modal.Header>

            <Form onSubmit={handleSave}>
              <Modal.Body className={s.modalBody}>
                <TextField
                  isRequired
                  validate={(v) => {
                    if (!v.trim()) return 'Обязательное поле'
                    if (v.length < 3) return 'Минимум 3 символа'
                    return null
                  }}
                  value={title}
                  onChange={setTitle}
                >
                  <Label>Название</Label>
                  <Input placeholder="Марафон: Весна 2026" />
                  <FieldError />
                </TextField>

                <TextField
                  isRequired
                  validate={(v) => !v ? 'Обязательное поле' : null}
                  value={startDate}
                  onChange={setStartDate}
                >
                  <Label>Дата начала</Label>
                  <Input type="date" />
                  <FieldError />
                </TextField>

                <TextField
                  isRequired
                  validate={(v) => {
                    if (!v) return 'Обязательное поле'
                    if (startDate && v <= startDate) return 'Должна быть позже даты начала'
                    return null
                  }}
                  value={endDate}
                  onChange={setEndDate}
                >
                  <Label>Дата окончания</Label>
                  <Input type="date" />
                  <FieldError />
                </TextField>

                <Switch isSelected={isActive} onChange={setIsActive}>
                  <Switch.Control><Switch.Thumb /></Switch.Control>
                  <Switch.Content>{isActive ? 'Активна' : 'Неактивна'}</Switch.Content>
                </Switch>

              </Modal.Body>
              <Modal.Footer>
                <Button variant="ghost" onPress={state.close}>Отмена</Button>
                <Button variant="primary" type="submit" isDisabled={isPending}>
                  {isPending ? 'Сохранение…' : 'Сохранить'}
                </Button>
              </Modal.Footer>
            </Form>
          </Modal.Dialog>
        </Modal.Container>
      </Modal.Backdrop>
    </Modal>
  )
}
