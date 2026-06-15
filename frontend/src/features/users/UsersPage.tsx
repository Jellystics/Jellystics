import { useState, useEffect } from 'react'
import { Alert, Avatar, Box, Card, CardActionArea, CardContent, Grid, Skeleton, Typography } from '@mui/material'
import { useTheme } from '@mui/material/styles'
import { createColumnHelper, type ColumnDef } from '@tanstack/react-table'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { format, parseISO } from 'date-fns'
import { BarChart } from '@mui/x-charts/BarChart'
import PageHeader from '@/shared/components/PageHeader/PageHeader'
import DataTable from '@/shared/components/DataTable/DataTable'
import ChartCard from '@/shared/components/ChartCard/ChartCard'
import MetricToggle, { type ActivityMetric } from '@/shared/components/MetricToggle/MetricToggle'
import api from '@/lib/axios'
import type { UserStats } from '@/shared/types/user'
import { formatWatchTime } from '@/shared/utils/formatWatchTime'
import { getDateLocale } from '@/lib/dateLocale'

const col = createColumnHelper<UserStats>()

const COLORS = ['#a78bfa', '#7c3aed', '#6d28d9', '#5b21b6', '#4c1d95', '#8b5cf6', '#c4b5fd', '#ede9fe']

interface LeaderboardUser {
  UserId: string
  UserName: string
  TotalPlays: number
  TotalWatchTime: number
}

const PODIUM_STYLES: Record<number, { border: string; bg: string; labelColor: string; label: string; avatarSize: number; order: number }> = {
  0: {
    border: '#FFD700',
    bg: 'rgba(255, 215, 0, 0.08)',
    labelColor: '#FFD700',
    label: '1st',
    avatarSize: 64,
    order: 2,
  },
  1: {
    border: '#C0C0C0',
    bg: 'rgba(192, 192, 192, 0.07)',
    labelColor: '#C0C0C0',
    label: '2nd',
    avatarSize: 52,
    order: 1,
  },
  2: {
    border: '#CD7F32',
    bg: 'rgba(205, 127, 50, 0.07)',
    labelColor: '#CD7F32',
    label: '3rd',
    avatarSize: 52,
    order: 3,
  },
}

export default function UsersPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const theme = useTheme()

  // --- Existing table state ---
  const [users, setUsers] = useState<UserStats[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = (showLoading = true) => {
    if (showLoading) setLoading(true)
    api.get('/stats/getUserStats').then((r) => setUsers(r.data ?? [])).catch(() => setError(t('common.loadError'))).finally(() => setLoading(false))
  }

  useEffect(() => {
    load()
    const interval = window.setInterval(() => load(false), 15000)
    return () => window.clearInterval(interval)
  }, [t])

  // --- Leaderboard state ---
  const [leaderboard, setLeaderboard] = useState<LeaderboardUser[]>([])
  const [leaderboardLoading, setLeaderboardLoading] = useState(true)
  const [metric, setMetric] = useState<ActivityMetric>('count')

  useEffect(() => {
    setLeaderboardLoading(true)
    api
      .get('/stats/getMostActiveUsers', { params: { limit: 10 } })
      .then((r) => setLeaderboard(r.data ?? []))
      .catch(() => setLeaderboard([]))
      .finally(() => setLeaderboardLoading(false))
  }, [])

  const top3 = leaderboard.slice(0, 3)
  const chartUsers = [...leaderboard].reverse()
  const chartNames = chartUsers.map((u) => u.UserName)
  const chartValues = metric === 'count'
    ? chartUsers.map((u) => u.TotalPlays)
    : chartUsers.map((u) => u.TotalWatchTime)

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
        try { return format(parseISO(v), 'dd/MM/yyyy HH:mm', { locale: getDateLocale() }) } catch { return v }
      },
    }),
    col.accessor('FavoriteGenre', { header: t('users.favoriteGenre'), cell: (i) => (i.getValue() as string | undefined) ?? '—' }),
  ]

  return (
    <>
      <PageHeader title={t('nav.users')} />
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      {/* Leaderboard section */}
      <Box sx={{ mb: 3 }}>
        {/* Top 3 Podium */}
        <Grid container spacing={2} sx={{ mb: 2 }} alignItems="flex-end">
          {leaderboardLoading
            ? [0, 1, 2].map((i) => (
                <Grid key={i} size={{ xs: 12, sm: 4 }} sx={{ order: i === 0 ? 2 : i === 1 ? 1 : 3 }}>
                  <Skeleton variant="rectangular" height={160} sx={{ borderRadius: 2 }} />
                </Grid>
              ))
            : top3.map((user, idx) => {
                const style = PODIUM_STYLES[idx]
                return (
                  <Grid key={user.UserId} size={{ xs: 12, sm: 4 }} sx={{ order: style.order }}>
                    <Card
                      sx={{
                        border: `1.5px solid ${style.border}`,
                        background: style.bg,
                        boxShadow: 'none',
                        height: idx === 0 ? 185 : 165,
                      }}
                    >
                      <CardActionArea
                        sx={{ height: '100%' }}
                        onClick={() => navigate(`/users/${user.UserId}`)}
                      >
                        <CardContent
                          sx={{
                            display: 'flex',
                            flexDirection: 'column',
                            alignItems: 'center',
                            justifyContent: 'center',
                            gap: 0.75,
                            height: '100%',
                            py: 2,
                          }}
                        >
                          <Typography
                            variant="caption"
                            sx={{
                              fontWeight: 700,
                              color: style.labelColor,
                              letterSpacing: 1,
                              textTransform: 'uppercase',
                              fontSize: 11,
                            }}
                          >
                            {style.label}
                          </Typography>
                          <Avatar
                            src={`/proxy/Users/Images/Primary/?id=${user.UserId}&fillWidth=128&quality=90`}
                            sx={{
                              width: style.avatarSize,
                              height: style.avatarSize,
                              border: `2px solid ${style.border}`,
                              bgcolor: 'primary.main',
                              fontSize: style.avatarSize * 0.4,
                              fontWeight: 700,
                            }}
                          >
                            {user.UserName.charAt(0).toUpperCase()}
                          </Avatar>
                          <Typography variant="body2" sx={{ fontWeight: 700, textAlign: 'center', lineHeight: 1.2 }}>
                            {user.UserName}
                          </Typography>
                          <Box sx={{ display: 'flex', gap: 1.5, flexWrap: 'wrap', justifyContent: 'center' }}>
                            <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                              {t('stats.totalPlays', 'Plays')}: <strong style={{ color: theme.palette.text.primary }}>{user.TotalPlays}</strong>
                            </Typography>
                            <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                              {t('stats.watchTime', 'Watch time')}: <strong style={{ color: theme.palette.text.primary }}>{formatWatchTime(user.TotalWatchTime)}</strong>
                            </Typography>
                          </Box>
                        </CardContent>
                      </CardActionArea>
                    </Card>
                  </Grid>
                )
              })}
        </Grid>

        {/* Bar chart Top 10 */}
        <ChartCard
          title={t('users.leaderboard', 'Top users')}
          loading={leaderboardLoading}
          empty={!leaderboardLoading && leaderboard.length === 0}
          height={300}
          action={<MetricToggle value={metric} onChange={setMetric} />}
        >
          <BarChart
            layout="horizontal"
            series={[
              {
                data: chartValues,
                color: COLORS[0],
                valueFormatter: (v) => metric === 'duration' ? formatWatchTime(v ?? 0) : String(v ?? 0),
              },
            ]}
            yAxis={[{ scaleType: 'band', data: chartNames }]}
            xAxis={[
              {
                label: metric === 'count'
                  ? t('stats.totalPlays', 'Plays')
                  : t('stats.watchTime', 'Watch time'),
              },
            ]}
            height={300}
            margin={{ left: 110, right: 20, top: 10, bottom: 40 }}
            sx={{
              '& .MuiChartsAxis-tickLabel': { fontSize: 12 },
            }}
          />
        </ChartCard>
      </Box>

      <DataTable data={users} columns={columns} loading={loading} searchPlaceholder={t('users.search')} onRefresh={load} />
    </>
  )
}
