import { useState, useEffect } from 'react'
import { Alert, Avatar, Box, Typography } from '@mui/material'
import { createColumnHelper, type ColumnDef } from '@tanstack/react-table'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { format, parseISO } from 'date-fns'
import PageHeader from '@/shared/components/PageHeader/PageHeader'
import DataTable from '@/shared/components/DataTable/DataTable'
import api from '@/lib/axios'
import type { UserStats } from '@/shared/types/user'
import { formatWatchTime } from '@/shared/utils/formatWatchTime'

const col = createColumnHelper<UserStats>()

export default function UsersPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [users, setUsers] = useState<UserStats[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api.get('/stats/getUserStats').then((r) => setUsers(r.data ?? [])).catch(() => setError(t('common.loadError'))).finally(() => setLoading(false))
  }, [t])

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const columns: ColumnDef<UserStats, any>[] = [
    col.accessor('UserName', {
      header: t('users.user'),
      cell: (i) => (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, cursor: 'pointer' }} onClick={() => navigate(`/users/${i.row.original.UserId}`)}>
          <Avatar
            src={`/proxy/Users/Images/Primary/?id=${i.row.original.UserId}&fillWidth=56&quality=90`}
            sx={{ width: 28, height: 28, bgcolor: 'primary.main', fontSize: 12, fontWeight: 700 }}
          >
            {(i.getValue() as string).charAt(0).toUpperCase()}
          </Avatar>
          <Typography variant="body2" sx={{ fontWeight: 500 }} color="primary.main">{i.getValue() as string}</Typography>
        </Box>
      ),
    }),
    col.accessor('TotalPlays', { header: t('stats.totalPlays') }),
    col.accessor('TotalWatchTime', { header: t('stats.watchTime'), cell: (i) => formatWatchTime(i.getValue() as number) }),
    col.accessor('LastSeen', {
      header: t('users.lastSeen'),
      cell: (i) => {
        const v = i.getValue() as string | undefined
        if (!v) return '—'
        try { return format(parseISO(v), 'dd/MM/yyyy HH:mm') } catch { return v }
      },
    }),
    col.accessor('FavoriteGenre', { header: t('users.favoriteGenre'), cell: (i) => (i.getValue() as string | undefined) ?? '—' }),
  ]

  return (
    <>
      <PageHeader title={t('nav.users')} />
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
      <DataTable data={users} columns={columns} loading={loading} searchPlaceholder={t('users.search')} />
    </>
  )
}
