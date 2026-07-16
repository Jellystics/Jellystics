import { useState, useEffect } from 'react'
import {
  Alert, Box, Typography, Grid, Card, CardActionArea, CardContent,
  ToggleButtonGroup, ToggleButton, Skeleton, TextField, InputAdornment, IconButton, Tooltip, Stack,
} from '@mui/material'
import { createColumnHelper, type ColumnDef } from '@tanstack/react-table'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { TableSimple24Regular, Grid24Regular, Search20Regular, ArrowSync24Regular } from '@fluentui/react-icons'
import PageHeader from '@/shared/components/PageHeader/PageHeader'
import DataTable, { type FilterDef } from '@/shared/components/DataTable/DataTable'
import api from '@/lib/axios'
import type { UserStats } from '@/shared/types/user'
import { formatWatchTime } from '@/shared/utils/formatWatchTime'
import { formatDateOnly, formatDateTime } from '@/shared/utils/formatDate'
import UserAvatar from '@/shared/components/UserAvatar/UserAvatar'

const col = createColumnHelper<UserStats>()

type ViewMode = 'table' | 'cards'

function UserCard({ user, onClick }: { user: UserStats; onClick: () => void }) {
  return (
    <Card sx={{ height: '100%' }}>
      <CardActionArea onClick={onClick} sx={{ height: '100%' }}>
        <CardContent sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', textAlign: 'center', py: 3, gap: 1.5 }}>
          <UserAvatar userId={user.UserId} userName={user.UserName} size={96} />
          <Typography variant="body1" noWrap sx={{ maxWidth: '100%', fontWeight: 700 }}>
            {user.UserName}
          </Typography>
        </CardContent>
      </CardActionArea>
    </Card>
  )
}

export default function UsersPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [viewMode, setViewMode] = useState<ViewMode>('table')
  const [search, setSearch] = useState('')

  const [users, setUsers] = useState<UserStats[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)


  const load = (showLoading = true) => {
    if (showLoading) setLoading(true)
    api.get('/stats/getUserStats')
      .then((r) => setUsers(r.data ?? []))
      .catch(() => setError(t('common.loadError')))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    load()
    const interval = window.setInterval(() => load(false), 15000)
    return () => window.clearInterval(interval)
  }, [t])


  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const columns: ColumnDef<UserStats, any>[] = [
    col.accessor('UserName', {
      header: t('users.user'),
      cell: (i) => (
        <Box
          sx={{ display: 'flex', alignItems: 'center', gap: 1, cursor: 'pointer' }}
          onClick={() => navigate(`/users/${i.row.original.UserId}?view=history`)}
        >
          <UserAvatar userId={i.row.original.UserId} userName={i.getValue() as string} size={28} />
          <Typography variant="body2" sx={{ fontWeight: 500 }} color="primary.main">
            {i.getValue() as string}
          </Typography>
        </Box>
      ),
    }),
    col.accessor('TotalPlays', { header: t('stats.totalPlays') }),
    col.accessor('TotalWatchTime', {
      header: t('stats.watchTime'),
      cell: (i) => formatWatchTime(i.getValue() as number),
    }),
    col.accessor('UniqueItems', {
      header: t('users.uniqueItems', 'Contenus uniques'),
      cell: (i) => (i.getValue() as number) || '—',
    }),
    col.accessor('MostUsedClient', {
      header: t('users.favoriteClient', 'Client favori'),
      cell: (i) => (i.getValue() as string | undefined) ?? '—',
    }),
    col.accessor('MostUsedDevice', {
      header: t('users.favoriteDevice', 'Appareil favori'),
      cell: (i) => (i.getValue() as string | undefined) ?? '—',
    }),
    col.accessor('FirstSeen', {
      header: t('users.firstSeen', 'Premier visionnage'),
      cell: (i) => formatDateOnly(i.getValue() as string | undefined),
    }),
    col.accessor('LastSeen', {
      header: t('users.lastSeen'),
      cell: (i) => formatDateTime(i.getValue() as string | undefined),
    }),
  ]

  const filterDefs: FilterDef[] = []

  const filteredUsers = users.filter((u) =>
    u.UserName.toLowerCase().includes(search.toLowerCase())
  )

  const viewToggle = (
    <ToggleButtonGroup
      exclusive
      size="small"
      value={viewMode}
      onChange={(_, v) => { if (v) setViewMode(v) }}
      sx={{ height: 32, '& .MuiToggleButton-root': { px: 1.25 } }}
    >
      <ToggleButton value="cards"><Grid24Regular style={{ fontSize: 16 }} /></ToggleButton>
      <ToggleButton value="table"><TableSimple24Regular style={{ fontSize: 16 }} /></ToggleButton>
    </ToggleButtonGroup>
  )

  return (
    <>
      <PageHeader title={t('nav.users')} actions={viewToggle} />
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      {viewMode === 'cards' ? (
        <>
          <Stack direction="row" spacing={1} sx={{ mb: 2, alignItems: 'center' }}>
            <TextField
              size="small"
              placeholder={t('users.search')}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              sx={{ width: 240 }}
              slotProps={{
                input: {
                  startAdornment: (
                    <InputAdornment position="start">
                      <Search20Regular style={{ fontSize: 16, color: 'var(--mui-palette-text-secondary)' }} />
                    </InputAdornment>
                  ),
                },
              }}
            />
            <Box sx={{ flexGrow: 1 }} />
            <Tooltip title={t('common.refresh')}>
              <span>
                <IconButton size="small" onClick={() => load()} disabled={loading} sx={{ color: 'text.secondary' }}>
                  <ArrowSync24Regular style={{ fontSize: 20 }} />
                </IconButton>
              </span>
            </Tooltip>
          </Stack>
          <Grid container spacing={2}>
            {loading
              ? Array.from({ length: 8 }).map((_, i) => (
                  <Grid key={i} size={{ xs: 6, sm: 4, md: 3, lg: 2 }}>
                    <Card>
                      <CardContent sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 1.5, py: 3 }}>
                        <Skeleton variant="circular" width={96} height={96} />
                        <Skeleton variant="text" width="60%" />
                      </CardContent>
                    </Card>
                  </Grid>
                ))
              : filteredUsers.map((user) => (
                  <Grid key={user.UserId} size={{ xs: 6, sm: 4, md: 3, lg: 2 }}>
                    <UserCard
                      user={user}
                      onClick={() => navigate(`/users/${user.UserId}?view=history`)}
                    />
                  </Grid>
                ))
            }
          </Grid>
        </>
      ) : (
        <DataTable
          data={users}
          columns={columns}
          loading={loading}
          searchPlaceholder={t('users.search')}
          onRefresh={load}
          filterDefs={filterDefs}
          initialColumnVisibility={{
            UniqueItems: false,
            MostUsedClient: false,
            MostUsedDevice: false,
            FirstSeen: false,
          }}
        />
      )}

    </>
  )
}
