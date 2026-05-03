import { Navigate, type RouteObject } from 'react-router-dom'
import LoginPage from '@/app/page'
import DashboardLayout from '@/app/dashboard/layout'
import DashboardPage from '@/app/dashboard/page'
import TaskDetailPage from '@/app/dashboard/task'
import TransactionsWidgetPage from '@/app/transactions/widget'
import AdminLayout from '@/app/settings/layout'
import AdminUsersPage from '@/app/settings/page'
import AdminTemplatesPage from '@/app/settings/templates'
import ManagerLayout from '@/app/manager/layout'
import ManagerSeasonsPage from '@/app/settings/seasons'
import ManagerSeasonDetailPage from '@/app/settings/season-detail'
import ManagerDashboardLayout from '@/app/manager/dashboard/layout'
import ManagerDashboardPage from '@/app/manager/dashboard/page'
import SeasonsPage from '@/app/seasons/page'
import RootRedirect from '@/app/root-redirect'

/**
 * /                                     — smart redirect
 * /seasons                            — season picker
 * /d/:seasonId                        — dashboard (resident)
 * /d/:seasonId/balance                — transaction history
 * /login                                — Telegram login
 * /admin/users                          — users management
 * /admin/templates                      — notification templates
 * /manager/d/:seasonId                — manager dashboard [manager]
 * /manager/seasons                   — seasons management [manager]
 * /manager/seasons/:seasonId       — season detail [manager]
 */
export const routes: RouteObject[] = [
  { path: '/',             element: <RootRedirect /> },
  { path: '/seasons',    element: <SeasonsPage /> },
  {
    path: '/d/:seasonId',
    element: <DashboardLayout />,
    children: [
      { index: true,           element: <DashboardPage /> },
      { path: 'balance',       element: <TransactionsWidgetPage /> },
      { path: 'task/:taskId',  element: <TaskDetailPage /> },
    ],
  },
  { path: '/login',        element: <LoginPage /> },
  {
    path: '/admin',
    element: <AdminLayout />,
    children: [
      { index: true,                   element: <Navigate to="users" replace /> },
      { path: 'users',                 element: <AdminUsersPage /> },
      { path: 'seasons',               element: <ManagerSeasonsPage /> },
      { path: 'seasons/:seasonId',     element: <ManagerSeasonDetailPage /> },
      { path: 'templates',             element: <AdminTemplatesPage /> },
    ],
  },
  {
    path: '/manager',
    children: [
      {
        path: 'd/:seasonId',
        element: <ManagerDashboardLayout />,
        children: [
          { index: true, element: <ManagerDashboardPage /> },
        ],
      },
      {
        element: <ManagerLayout />,
        children: [
          { path: 'seasons',              element: <ManagerSeasonsPage /> },
          { path: 'seasons/:seasonId',    element: <ManagerSeasonDetailPage /> },
        ],
      },
    ],
  },
  { path: '*',             element: <Navigate to="/" replace /> },
]

