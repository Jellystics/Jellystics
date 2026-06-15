import { createBrowserRouter } from 'react-router-dom'
import AppShell from '@/shared/components/Layout/AppShell'
import AuthGuard from '@/features/auth/AuthGuard'
import PublicGuard from '@/features/auth/PublicGuard'
import LoginPage from '@/features/auth/LoginPage'
import SetupPage from '@/features/auth/SetupPage'
import DashboardPage from '@/features/dashboard/DashboardPage'
import ActivityPage from '@/features/activity/ActivityPage'
import TimelinePage from '@/features/activity/TimelinePage'
import LibrariesPage from '@/features/libraries/LibrariesPage'
import LibraryDetailPage from '@/features/libraries/LibraryDetailPage'
import ItemDetailPage from '@/features/libraries/ItemDetailPage'
import UsersPage from '@/features/users/UsersPage'
import UserDetailPage from '@/features/users/UserDetailPage'
import StatisticsPage from '@/features/statistics/StatisticsPage'
import SettingsPage from '@/features/settings/SettingsPage'

export const router = createBrowserRouter([
  {
    element: <PublicGuard />,
    children: [
      { path: '/login', element: <LoginPage /> },
      { path: '/setup', element: <SetupPage /> },
    ],
  },
  {
    element: <AuthGuard />,
    children: [
      {
        path: '/',
        element: <AppShell />,
        children: [
          { index: true, element: <DashboardPage /> },
          { path: 'activity', element: <ActivityPage /> },
          { path: 'activity/timeline', element: <TimelinePage /> },
          { path: 'libraries', element: <LibrariesPage /> },
          { path: 'libraries/:id', element: <LibraryDetailPage /> },
          { path: 'libraries/:libraryId/items/:itemId', element: <ItemDetailPage /> },
          { path: 'users', element: <UsersPage /> },
          { path: 'users/:id', element: <UserDetailPage /> },
          { path: 'statistics', element: <StatisticsPage /> },
          { path: 'settings', element: <SettingsPage /> },
        ],
      },
    ],
  },
])
