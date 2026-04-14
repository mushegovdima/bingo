import { Button, Avatar, AvatarImage, AvatarFallback } from '@heroui/react'
import { Link, useLocation } from 'react-router-dom'
import { useAuth } from '@/hooks/useAuth'
import s from './AppNavbar.module.scss'

export function AppNavbar() {
  const { pathname } = useLocation()
  const { user, isManager, logout } = useAuth()

  return (
    <nav className={s.navbar}>
      <div className="max-w-7xl mx-auto w-full flex items-center justify-between gap-4">

        {/* Brand + Desktop nav */}
        <div className="flex items-center gap-8">
          <Link className={s.brand} to="/">BINGO</Link>
          <div className="flex items-center gap-6">
            {isManager && (
              <Link
                className={`${s.link} ${pathname === '/admin' ? s.linkActive : ''}`}
                to="/admin"
              >
                Админ панель
              </Link>
            )}
          </div>
        </div>

        {/* Right: user info + logout (desktop) */}
        <div className="flex items-center gap-3">
          {user && (
            <>
              <div className="flex flex-col items-end">
                <span className={s.userName}>
                  {user.name}
                </span>
              </div>

              <Avatar size="sm" color="accent">
                {user.photo_url && <AvatarImage src={user.photo_url} />}
                <AvatarFallback>{user.name[0] ?? '?'}</AvatarFallback>
              </Avatar>

              <Button size="sm" className={s.logoutBtn} onPress={() => logout()}>
                Выйти
              </Button>
            </>
          )}
        </div>
      </div>
    </nav>
  )
}

