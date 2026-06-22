import { useState, useEffect } from 'react'
import {
  Alert, Box, Typography, Avatar, Grid, Card, CardActionArea, CardContent,
  ToggleButtonGroup, ToggleButton, Skeleton, TextField, InputAdornment, IconButton, Tooltip, Stack,
  LinearProgress, Paper,
} from '@mui/material'
import { alpha, useTheme } from '@mui/material/styles'
import { createColumnHelper, type ColumnDef } from '@tanstack/react-table'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { format, parseISO } from 'date-fns'
import { TableSimple24Regular, Grid24Regular, Search20Regular, ArrowSync24Regular } from '@fluentui/react-icons'
import PageHeader from '@/shared/components/PageHeader/PageHeader'
import DataTable, { type FilterDef } from '@/shared/components/DataTable/DataTable'
import api from '@/lib/axios'
import type { UserStats } from '@/shared/types/user'
import { formatWatchTime } from '@/shared/utils/formatWatchTime'
import { getDateLocale } from '@/lib/dateLocale'

const col = createColumnHelper<UserStats>()

type ViewMode = 'table' | 'cards'

function UserCard({ user, onClick }: { user: UserStats; onClick: () => void }) {
  return (
    <Card sx={{ height: '100%' }}>
      <CardActionArea onClick={onClick} sx={{ height: '100%' }}>
        <CardContent sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', textAlign: 'center', py: 3, gap: 1.5 }}>
          <Avatar
            src={`/proxy/Users/Images/Primary/?id=${user.UserId}&fillWidth=200&quality=90`}
            sx={{ width: 96, height: 96, bgcolor: 'primary.main', fontSize: 36, fontWeight: 700 }}
          >
            {user.UserName.charAt(0).toUpperCase()}
          </Avatar>
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
  const theme = useTheme()
  const [viewMode, setViewMode] = useState<ViewMode>('table')
  const [search, setSearch] = useState('')

  const [users, setUsers] = useState<UserStats[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const [diversity, setDiversity] = useState<any[]>([])
  const [diversityLoading, setDiversityLoading] = useState(true)

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

  useEffect(() => {
    setDiversityLoading(true)
    api.get('/stats/getViewingDiversity?days=30')
      .then(r => setDiversity(r.data?.users ?? []))
      .catch(() => setDiversity([]))
      .finally(() => setDiversityLoading(false))
  }, [])

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const columns: ColumnDef<UserStats, any>[] = [
    col.accessor('UserName', {
      header: t('users.user'),
      cell: (i) => (
        <Box
          sx={{ display: 'flex', alignItems: 'center', gap: 1, cursor: 'pointer' }}
          onClick={() => navigate(`/users/${i.row.original.UserId}?view=history`)}
        >
          <Avatar
            src={`/proxy/Users/Images/Primary/?id=${i.row.original.UserId}&fillWidth=56&quality=90`}
            sx={{ width: 28, height: 28, bgcolor: 'primary.main', fontSize: 12, fontWeight: 700 }}
          >
            {(i.getValue() as string).charAt(0).toUpperCase()}
          </Avatar>
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
      cell: (i) => {
        const v = i.getValue() as string | undefined
        if (!v) return '—'
        try { return format(parseISO(v), 'dd/MM/yyyy', { locale: getDateLocale() }) } catch { return v }
      },
    }),
    col.accessor('LastSeen', {
      header: t('users.lastSeen'),
      cell: (i) => {
        const v = i.getValue() as string | undefined
        if (!v) return '—'
        try { return format(parseISO(v), 'dd/MM/yyyy HH:mm', { locale: getDateLocale() }) } catch { return v }
      },
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

      {/* Viewing Diversity */}
      <Paper sx={{ mt: 3, p: 3 }}>
        <Typography variant="h6" sx={{ fontWeight: 700, mb: 2 }}>
          {t('insights.viewingDiversity', 'Viewing Diversity')}
        </Typography>

        {diversityLoading ? (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} variant="rounded" height={40} />
            ))}
          </Box>
        ) : diversity.length === 0 ? (
          <Typography variant="body2" color="text.secondary">
            {t('common.noData', 'No data available')}
          </Typography>
        ) : (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            {/* Header */}
            <Box
              sx={{
                display: 'grid',
                gridTemplateColumns: '40px 1fr 2fr 80px 80px',
                alignItems: 'center',
                gap: 1.5,
                px: 1.5,
                py: 0.5,
              }}
            >
              <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 700 }}>#</Typography>
              <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 700 }}>{t('users.user')}</Typography>
              <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 700 }}>{t('insights.score', 'Score')}</Typography>
              <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 700, textAlign: 'right' }}>{t('insights.genres', 'Genres')}</Typography>
              <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 700, textAlign: 'right' }}>{t('stats.totalPlays', 'Plays')}</Typography>
            </Box>

            {/* Rows */}
            {diversity.map((u, idx) => (
              <Box
                key={u.userId}
                sx={{
                  display: 'grid',
                  gridTemplateColumns: '40px 1fr 2fr 80px 80px',
                  alignItems: 'center',
                  gap: 1.5,
                  px: 1.5,
                  py: 1,
                  borderRadius: 1,
                  transition: 'background-color 0.15s',
                  '&:hover': {
                    backgroundColor: alpha(theme.palette.primary.main, 0.06),
                  },
                }}
              >
                <Typography variant="body2" sx={{ fontWeight: 700 }} color="text.secondary">
                  {idx + 1}
                </Typography>
                <Typography variant="body2" sx={{ fontWeight: 500 }} noWrap>
                  {u.userName}
                </Typography>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
                  <LinearProgress
                    variant="determinate"
                    value={Math.round((u.diversityScore ?? 0) * 100)}
                    sx={{
                      flex: 1,
                      height: 8,
                      borderRadius: 4,
                      backgroundColor: alpha(theme.palette.primary.main, 0.12),
                      '& .MuiLinearProgress-bar': { borderRadius: 4 },
                    }}
                  />
                  <Typography variant="caption" sx={{ fontWeight: 700, minWidth: 36, textAlign: 'right' }}>
                    {Math.round((u.diversityScore ?? 0) * 100)}%
                  </Typography>
                </Box>
                <Typography variant="body2" sx={{ textAlign: 'right' }}>
                  {u.uniqueGenres ?? 0}
                </Typography>
                <Typography variant="body2" sx={{ textAlign: 'right' }}>
                  {u.totalPlays ?? 0}
                </Typography>
              </Box>
            ))}
          </Box>
        )}
      </Paper>
    </>
  )
}
