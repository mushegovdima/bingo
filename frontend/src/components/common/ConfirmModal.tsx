import type { ReactNode } from 'react'
import { Button, Modal, useOverlayState } from '@heroui/react'

interface Props {
  title: string
  message: ReactNode
  confirmLabel?: string
  cancelLabel?: string
  variant?: 'primary' | 'danger'
  onConfirm: () => void
  onClose: () => void
}

export function ConfirmModal({
  title,
  message,
  confirmLabel = 'Подтвердить',
  cancelLabel = 'Отмена',
  variant = 'primary',
  onConfirm,
  onClose,
}: Props) {
  const state = useOverlayState({
    defaultOpen: true,
    onOpenChange: (open) => { if (!open) onClose() },
  })

  return (
    <Modal state={state}>
      <Modal.Backdrop isDismissable>
        <Modal.Container size="sm">
          <Modal.Dialog>
            <Modal.Header>
              <Modal.Heading>{title}</Modal.Heading>
              <Modal.CloseTrigger />
            </Modal.Header>
            <Modal.Body>
              <p>{message}</p>
            </Modal.Body>
            <Modal.Footer>
              <Button variant="ghost" onPress={state.close}>{cancelLabel}</Button>
              <Button
                variant={variant === 'danger' ? 'outline' : 'primary'}
                onPress={() => { onConfirm(); state.close() }}
                style={variant === 'danger' ? { color: 'var(--color-danger)', borderColor: 'var(--color-danger)' } : undefined}
              >
                {confirmLabel}
              </Button>
            </Modal.Footer>
          </Modal.Dialog>
        </Modal.Container>
      </Modal.Backdrop>
    </Modal>
  )
}
