import { useMemo, useState } from 'react'
import type { SortDescriptor } from '@heroui/react'
import { LogIn, Pencil } from 'lucide-react'
import {
  Avatar,
  AvatarFallback,
  AvatarImage,
  Button,
  Checkbox,
  Chip,
  Input,
  Label,
  Modal,
  Skeleton,
  Switch,
  Table,
  TableLayout,
  TextField,
  Tooltip,
  Virtualizer,
  useOverlayState,
} from '@heroui/react'
import { useUsers } from '@/hooks/useUsers'
import { useUpdateUser } from '@/hooks/useUpdateUser'
import { useImpersonate } from '@/hooks/useImpersonate'
import { User, UserRole } from '@/types'
import type { UpdateUserRequest } from '@/lib/api/users'
import { ConfirmModal } from '@/components/common/ConfirmModal'
import s from '../settings.module.scss'

// ─── Edit Modal ───────────────────────────────────────────────────────────────

interface EditModalProps {
  user: User
  onClose: () => void
}

function EditModal({ user, onClose }: EditModalProps) {
  const [name, setName] = useState(user.name)
  const [roles, setRoles] = useState<UserRole[]>(user.roles)
  const [isBlocked, setIsBlocked] = useState(user.is_blocked)
  const { mutate: updateUser, isPending } = useUpdateUser()
  const state = useOverlayState({
    defaultOpen: true,
    onOpenChange: (open) => {
      if (!open) onClose()
    },
  })

  const toggleRole = (role: UserRole) => {
    setRoles((prev) =>
      prev.includes(role) ? prev.filter((r) => r !== role) : [...prev, role],
    )
  }

  const handleSave = () => {
    const data: UpdateUserRequest = { name, roles, is_blocked: isBlocked }
    updateUser({ id: user.id, data }, { onSuccess: state.close })
  }

  return (
    <Modal state={state}>
      <Modal.Backdrop isDismissable>
        <Modal.Container size="sm">
          <Modal.Dialog>
            <Modal.Header>
              <Modal.Heading>Редактировать пользователя</Modal.Heading>
            <Modal.CloseTrigger />
          </Modal.Header>
          <Modal.Body className={s.modalBody}>
            <TextField value={name} onChange={setName}>
              <Label>Имя</Label>
              <Input />
            </TextField>

            <div className={s.field}>
              <Label>Роли</Label>
              <div className={s.rolesGroup}>
                {(['manager', 'resident'] as UserRole[]).map((role) => (
                  <Checkbox
                    key={role}
                    isSelected={roles.includes(role)}
                    onChange={() => toggleRole(role)}
                  >
                    <Checkbox.Control>
                      <Checkbox.Indicator />
                    </Checkbox.Control>
                    <Checkbox.Content>{role}</Checkbox.Content>
                  </Checkbox>
                ))}
              </div>
            </div>

            <div className={s.field}>
              <Label>Статус</Label>
              <Switch isSelected={isBlocked} onChange={setIsBlocked}>
                <Switch.Control>
                  <Switch.Thumb />
                </Switch.Control>
                <Switch.Content>
                  {isBlocked ? 'Заблокирован' : 'Активен'}
                </Switch.Content>
              </Switch>
            </div>

            <TextField isReadOnly value={String(user.telegram_id)}>
              <Label>Telegram ID</Label>
              <Input />
            </TextField>
            <TextField isReadOnly value={user.username || '—'}>
              <Label>Username</Label>
              <Input />
            </TextField>
            <TextField isReadOnly value={new Date(user.created_at).toLocaleDateString('ru-RU')}>
              <Label>Зарегистрирован</Label>
              <Input />
            </TextField>
          </Modal.Body>
          <Modal.Footer>
            <Button variant="ghost" onPress={state.close}>
              Отмена
            </Button>
            <Button variant="primary" onPress={handleSave} isDisabled={isPending}>
              {isPending ? 'Сохранение…' : 'Сохранить'}
            </Button>
            </Modal.Footer>
          </Modal.Dialog>
        </Modal.Container>
      </Modal.Backdrop>
    </Modal>
  )
}

// ─── Users Tab ────────────────────────────────────────────────────────────────

export function UsersTab() {
  const { data: users, isLoading } = useUsers()
  const {
    mutate: impersonate,
    isPending: isImpersonating,
    variables: impersonatingId,
  } = useImpersonate()
  const [editingUser, setEditingUser] = useState<User | null>(null)
  const [impersonatingConfirm, setImpersonatingConfirm] = useState<User | null>(null)
  const [search, setSearch] = useState('')
  const [sortDescriptor, setSortDescriptor] = useState<SortDescriptor>({
    column: 'id',
    direction: 'ascending',
  })

  const layout = useMemo(() => new TableLayout({  }), [])

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase()
    if (!q) return users ?? []
    return (users ?? []).filter(
      (u) =>
        u.name.toLowerCase().includes(q) ||
        (u.username ?? '').toLowerCase().includes(q) ||
        String(u.id).includes(q) ||
        String(u.telegram_id).includes(q),
    )
  }, [users, search])

  const sorted = useMemo(() => {
    const col = sortDescriptor.column as string
    const dir = sortDescriptor.direction === 'descending' ? -1 : 1
    return [...filtered].sort((a, b) => {
      switch (col) {
        case 'user':   return dir * a.name.localeCompare(b.name)
        case 'status': return dir * (Number(a.is_blocked) - Number(b.is_blocked))
        case 'id':     return dir * (a.id - b.id)
        default:       return 0
      }
    })
  }, [filtered, sortDescriptor])

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
    <>
      <div className={s.tableCard}>
      <div className={s.tableToolbar}>
        <Input
          placeholder="Поиск по имени, @username, ID…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className={s.searchInput}
        />
        <span className={s.countLabel}>{filtered.length} / {(users ?? []).length}</span>
      </div>

      <Table className={s.usersTable} variant="primary">
        <Table.ResizableContainer>
          <Virtualizer layout={layout}>
            <Table.Content
              aria-label="Пользователи"
              sortDescriptor={sortDescriptor}
              onSortChange={setSortDescriptor}
            >
            <Table.Header>
              <Table.Column id="user" minWidth={200} isRowHeader allowsSorting>
                Пользователь
                <Table.ColumnResizer />
              </Table.Column>
              <Table.Column id="roles" minWidth={150}>
                Роли
                <Table.ColumnResizer />
              </Table.Column>
              <Table.Column id="status" minWidth={130} allowsSorting>
                Статус
                <Table.ColumnResizer />
              </Table.Column>
              <Table.Column id="id" defaultWidth={80} allowsSorting>
                ID
                <Table.ColumnResizer />
              </Table.Column>
              <Table.Column id="actions" defaultWidth={100}>Действия</Table.Column>
            </Table.Header>
            <Table.Body items={sorted}>
              {(user) => (
                <Table.Row id={user.id}>
                  <Table.Cell>
                    <div className={s.userInfo}>
                      <Avatar size="sm">
                        {user.photo_url ? (
                          <AvatarImage src={user.photo_url} alt={user.name} />
                        ) : (
                          <AvatarFallback>{user.name[0] ?? '?'}</AvatarFallback>
                        )}
                      </Avatar>
                      <div>
                        <div className={s.userName}>{user.name}</div>
                        {user.username && (
                          <div className={s.userUsername}>@{user.username}</div>
                        )}
                      </div>
                    </div>
                  </Table.Cell>
                  <Table.Cell>
                    <div className={s.chipsRow}>
                      {user.roles.map((role) => (
                        <Chip
                          key={role}
                          size="sm"
                          color={role === 'manager' ? 'accent' : 'default'}
                        >
                          {role}
                        </Chip>
                      ))}
                    </div>
                  </Table.Cell>
                  <Table.Cell>
                    <Chip
                      size="sm"
                      color={user.is_blocked ? 'danger' : 'success'}
                    >
                      {user.is_blocked ? 'Заблокирован' : 'Активен'}
                    </Chip>
                  </Table.Cell>
                  <Table.Cell>
                    <span className={s.muted}>{user.id}</span>
                  </Table.Cell>
                  <Table.Cell>
                    <div className={s.actions}>
                      <Button
                        size="sm"
                        variant="ghost"
                        aria-label="Редактировать"
                        isIconOnly
                        onPress={() => setEditingUser(user)}
                      >
                        <Pencil size={14} />
                      </Button>
                      <Tooltip>
                        <Tooltip.Trigger>
                          <Button
                            size="sm"
                            variant="ghost"
                            aria-label="Войти под пользователем"
                            isIconOnly
                            isDisabled={isImpersonating && impersonatingId === user.id}
                            onPress={() => setImpersonatingConfirm(user)}
                          >
                            <LogIn size={14} />
                          </Button>
                        </Tooltip.Trigger>

                        <Tooltip.Content>
                          Войти под этим пользователем
                        </Tooltip.Content>
                      </Tooltip>
                    </div>
                  </Table.Cell>
                </Table.Row>
              )}
            </Table.Body>
            </Table.Content>
          </Virtualizer>
        </Table.ResizableContainer>
      </Table>
      </div>

      {editingUser && (
        <EditModal user={editingUser} onClose={() => setEditingUser(null)} />
      )}
      {impersonatingConfirm && (
        <ConfirmModal
          title="Войти под пользователем"
          message={<>Вы уверены, что хотите войти под <strong>{impersonatingConfirm.name}</strong>{impersonatingConfirm.username ? ` (@${impersonatingConfirm.username})` : ''}?</>}
          confirmLabel="Войти"
          onConfirm={() => impersonate(impersonatingConfirm.id)}
          onClose={() => setImpersonatingConfirm(null)}
        />
      )}
    </>
  )
}
