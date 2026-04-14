import { Navigate, type RouteObject } from 'react-router-dom'
import LoginPage from '@/app/page'
import DashboardLayout from '@/app/dashboard/layout'
import DashboardPage from '@/app/dashboard/page'
import TransactionsWidgetPage from '@/app/transactions/widget'
import AdminLayout from '@/app/settings/layout'
import AdminUsersPage from '@/app/settings/page'
import AdminSeasonsPage from '@/app/settings/seasons'
import AdminSeasonDetailPage from '@/app/settings/season-detail'
import SeasonsPage from '@/app/seasons/page'
import RootRedirect from '@/app/root-redirect'

/**
 * /                                     — smart redirect
 * /seasons                            — season picker
 * /d/:seasonId                        — dashboard
 * /d/:seasonId/balance                — transaction history
 * /login                                — Telegram login
 * /admin/users                          — users management
 * /admin/seasons                      — seasons list
 * /admin/seasons/:seasonId          — season detail
 */
export const routes: RouteObject[] = [
  { path: '/',             element: <RootRedirect /> },
  { path: '/seasons',    element: <SeasonsPage /> },
  {
    path: '/d/:seasonId',
    element: <DashboardLayout />,
    children: [
      { index: true,     element: <DashboardPage /> },
      { path: 'balance', element: <TransactionsWidgetPage /> },
    ],
  },
  { path: '/login',        element: <LoginPage /> },
  {
    path: '/admin',
    element: <AdminLayout />,
    children: [
      { index: true,                   element: <Navigate to="users" replace /> },
      { path: 'users',                 element: <AdminUsersPage /> },
      { path: 'seasons',             element: <AdminSeasonsPage /> },
      { path: 'seasons/:seasonId', element: <AdminSeasonDetailPage /> },
    ],
  },
  { path: '*',             element: <Navigate to="/" replace /> },
]

