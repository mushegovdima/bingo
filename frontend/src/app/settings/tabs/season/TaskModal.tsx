import { useState, type FormEvent } from 'react'
import { Button, ComboBox, FieldError, Form, Input, Label, ListBox, ListBoxItem, Modal, NumberField, Switch, TextField, useOverlayState, TextArea } from '@heroui/react'
import { Task } from '@/types'
import { useCreateTask, useUpdateTask } from '@/hooks/useTaskMutations'
import type { CreateTaskRequest, UpdateTaskRequest } from '@/lib/api/tasks'
import s from '../../settings.module.scss'

interface Props {
  seasonId: number
  task?: Task
  categories?: string[]
  onClose: () => void
}

export function TaskModal({ seasonId, task, categories = [], onClose }: Props) {
  const isEdit = !!task?.id
  const [title, setTitle] = useState(task?.title ?? '')
  const [category, setCategory] = useState(task?.category ?? '')
  const [description, setDescription] = useState(task?.description ?? '')
  const [rewardCoins, setRewardCoins] = useState(task?.reward_coins ?? 0)
  const [sortOrder, setSortOrder] = useState(task?.sort_order ?? 0)
  const [isActive, setIsActive] = useState(task?.is_active ?? true)

  const { mutate: create, isPending: isCreating } = useCreateTask()
  const { mutate: update, isPending: isUpdating } = useUpdateTask(seasonId)
  const isPending = isCreating || isUpdating

  const state = useOverlayState({
    defaultOpen: true,
    onOpenChange: (open) => { if (!open) onClose() },
  })

  const handleSave = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    if (isEdit) {
      const data: UpdateTaskRequest = {
        title,
        category,
        description,
        reward_coins: rewardCoins,
        sort_order: sortOrder,
        is_active: isActive,
      }
      update({ id: task.id, data }, { onSuccess: state.close })
    } else {
      const data: CreateTaskRequest = {
        season_id: seasonId,
        title,
        category,
        description,
        reward_coins: rewardCoins,
        sort_order: sortOrder,
        is_active: isActive,
      }
      create(data, { onSuccess: state.close })
    }
  }

  return (
    <Modal state={state}>
      <Modal.Backdrop isDismissable>
        <Modal.Container size="lg">
          <Modal.Dialog>
            <Modal.Header>
              <Modal.Heading>{isEdit ? 'Редактировать задачу' : 'Новая задача'}</Modal.Heading>
              <Modal.CloseTrigger />
            </Modal.Header>
            <Form onSubmit={handleSave}>
            <Modal.Body className={s.modalBody}>
              <TextField
                isRequired
                validationBehavior="aria"
                validate={(v) => {
                  if (!v.trim()) return 'Обязательное поле'
                  if (v.length < 3) return 'Минимум 3 символа'
                  return null
                }}
                value={title}
                onChange={setTitle}
              >
                <Label>Название</Label>
                <Input placeholder="Написать пост о выходных" />
                <FieldError />
              </TextField>

              <ComboBox
                allowsCustomValue
                inputValue={category}
                onInputChange={setCategory}
              >
                <Label>Категория</Label>
                <ComboBox.InputGroup>
                  <Input placeholder="Семья, бизнес, ..." />
                  <ComboBox.Trigger />
                </ComboBox.InputGroup>
                <ComboBox.Popover>
                  <ListBox>
                    {categories.map((c) => (
                      <ListBoxItem key={c} id={c} textValue={c}>{c}</ListBoxItem>
                    ))}
                  </ListBox>
                </ComboBox.Popover>
              </ComboBox>

              <TextField value={description} onChange={setDescription}>
                <Label>Описание</Label>
                <TextArea placeholder="Подробности задачи" rows={3} />
              </TextField>

              <div className={s.fieldRowWrap}>
                <NumberField value={rewardCoins} onChange={setRewardCoins} minValue={0}>
                  <Label>Монеты</Label>
                  <NumberField.Group>
                    <NumberField.DecrementButton />
                    <NumberField.Input />
                    <NumberField.IncrementButton />
                  </NumberField.Group>
                </NumberField>

                <NumberField value={sortOrder} onChange={setSortOrder} minValue={0}>
                  <Label>Порядок</Label>
                  <NumberField.Group>
                    <NumberField.DecrementButton />
                    <NumberField.Input />
                    <NumberField.IncrementButton />
                  </NumberField.Group>
                </NumberField>
              </div>

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
