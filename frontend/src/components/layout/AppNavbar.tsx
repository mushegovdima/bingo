import { Button, Avatar, AvatarImage, AvatarFallback } from '@heroui/react'
import { Link, useLocation } from 'react-router-dom'
import { useAuth } from '@/hooks/useAuth'

export function AppNavbar() {
  const { pathname } = useLocation()
  const { user, isManager, logout } = useAuth()

  return (
    <nav className="bg-(--color-surface) border-b border-(--color-border) shadow-(--shadow-sm) px-4 h-16 flex items-center relative">
      <div className="max-w-7xl mx-auto w-full flex items-center justify-between gap-4">

        {/* Brand + Desktop nav */}
        <div className="flex items-center gap-8">
          <Link
            className="text-(--color-primary) font-bold text-xl tracking-tight select-none no-underline"
            to="/"
          >
            BINGO
          </Link>
          <div className="flex items-center gap-6">
            {isManager && (
              <Link
                className={`text-sm no-underline transition-colors duration-150 ${
                  pathname.startsWith('/admin')
                    ? 'text-(--color-primary) font-semibold'
                    : 'text-(--color-text-muted) hover:text-(--color-text)'
                }`}
                to="/admin"
              >
                Настройки
              </Link>
            )}
          </div>
        </div>

        {/* Right: user info + logout (desktop) */}
        <div className="flex items-center gap-3">
          {user && (
            <>
              <div className="hidden sm:flex flex-col items-end">
                <span className="text-(--color-text-secondary) text-sm font-medium leading-tight">
                  {user.name}
                </span>
              </div>

              <Avatar size="sm" color="accent">
                {user.photo_url && <AvatarImage src={user.photo_url} />}
                <AvatarFallback>{user.name[0] ?? '?'}</AvatarFallback>
              </Avatar>

              <Button
                size="sm"
                className="bg-(--color-surface)! border! border-(--color-border)! text-(--color-text-muted)! transition-[background-color,color] duration-150 hover:bg-slate-50! hover:text-(--color-text)!"
                onPress={() => logout()}
              >
                Выйти
              </Button>
            </>
          )}
        </div>
      </div>
    </nav>
  )
}

