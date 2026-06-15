import { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'
import {
  Grid, Alert, Box, Typography, Tabs, Tab, Avatar, Chip,
  Card, CardContent, Skeleton,
} from '@mui/material'
import { useTranslation } from 'react-i18next'
import { format, parseISO } from 'date-fns'
import { createColumnHelper, type ColumnDef } from '@tanstack/react-table'
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid } from 'recharts'
import { useTheme } from '@mui/material/styles'
import PageHeader from '@/shared/components/PageHeader/PageHeader'
import StatCard from '@/shared/components/StatCard/StatCard'
import DataTable from '@/shared/components/DataTable/DataTable'
import ChartCard from '@/shared/components/ChartCard/ChartCard'
import ActivityHeatmap from './components/ActivityHeatmap'
import MetricToggle, { type ActivityMetric } from '@/shared/components/MetricToggle/MetricToggle'
import api from '@/lib/axios'
import type { Activity } from '@/shared/types/activity'
import type { UserStats, UserActivity } from '@/shared/types/user'
import type { GenreStat } from '@/shared/types/library'
import { Play24Regular, Clock24Regular, Star24Regular } from '@fluentui/react-icons'
import { formatWatchTime } from '@/shared/utils/formatWatchTime'

const col = createColumnHelper<Activity>()

export default function UserDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { t } = useTranslation()
  const theme = useTheme()
  const [tab, setTab] = useState(0)
  const [userStats, setUserStats] = useState<UserStats | null>(null)
  const [activity, setActivity] = useState<Activity[]>([])
  const [heatmapData, setHeatmapData] = useState<UserActivity[]>([])
  const [genres, setGenres] = useState<GenreStat[]>([])
  const [watchOverTime, setWatchOverTime] = useState<{ date: string; plays: number; duration: number }[]>([])
  const [heatmapMetric, setHeatmapMetric] = useState<ActivityMetric>('count')
  const [watchMetric, setWatchMetric] = useState<ActivityMetric>('duration')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!id) return
    setLoading(true)
    Promise.all([
      api.get(`/stats/getUserStats?userId=${id}`),
      api.get(`/stats/getUserActivity?userId=${id}`),
      api.get(`/stats/getUserActivityByDate?userId=${id}`),
      api.get(`/stats/getUserGenreStats?userId=${id}`),
      api.get(`/stats/getWatchStatisticsOverTime?userId=${id}&days=30`),
    ])
      .then(([statsRes, activityRes, heatmapRes, genreRes, overTimeRes]) => {
        setUserStats(statsRes.data)
        setActivity(activityRes.data ?? [])
        setHeatmapData(heatmapRes.data ?? [])
        setGenres(genreRes.data ?? [])
        setWatchOverTime(overTimeRes.data ?? [])
      })
      .catch(() => setError(t('common.loadError')))
      .finally(() => setLoading(false))
  }, [id, t])

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const columns: ColumnDef<Activity, any>[] = [
    col.accessor('NowPlayingItemName', {
      header: t('activity.item'),
      cell: (i) => { const row = i.row.original; return row.SeriesName ? `${row.SeriesName} — ${i.getValue() as string}` : i.getValue() as string },
    }),
    col.accessor('Client', { header: t('activity.client') }),
    col.accessor('PlayMethod', { header: t('activity.method'), cell: (i) => (i.getValue() as string | undefined) ?? '—' }),
    col.accessor('ActivityDateInserted', {
      header: t('activity.date'),
      cell: (i) => { try { return format(parseISO(i.getValue() as string), 'dd/MM/yyyy HH:mm') } catch { return i.getValue() as string } },
    }),
    col.accessor('PlayDuration', {
      header: t('activity.duration'),
      cell: (i) => { const s = Math.floor(((i.getValue() as number) ?? 0) / 10_000_000); const h = Math.floor(s / 3600); const m = Math.floor((s % 3600) / 60); return h > 0 ? `${h}h ${m}m` : `${m}m` },
    }),
  ]

  const username = userStats?.UserName ?? id ?? '...'

  return (
    <>
      <PageHeader title={username} />
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      <Card sx={{ mb: 3 }}>
        <CardContent sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          {loading ? <Skeleton variant="circular" width={56} height={56} /> : (
            <Avatar
              src={`/proxy/Users/Images/Primary/?id=${id}&fillWidth=112&quality=90`}
              sx={{ width: 56, height: 56, bgcolor: 'primary.main', fontSize: 22, fontWeight: 700 }}
            >
              {username.charAt(0).toUpperCase()}
            </Avatar>
          )}
          <Box>
            {loading ? <Skeleton width={120} height={28} /> : <Typography variant="h6" sx={{ fontWeight: 700 }}>{username}</Typography>}
            {loading ? <Skeleton width={180} height={20} /> : (
              <Typography variant="body2" color="text.secondary">
                {t('users.lastSeen')}: {userStats?.LastSeen ? format(parseISO(userStats.LastSeen), 'dd/MM/yyyy HH:mm') : '—'}
              </Typography>
            )}
          </Box>
          {userStats?.FavoriteGenre && <Chip label={userStats.FavoriteGenre} size="small" sx={{ ml: 'auto' }} />}
        </CardContent>
      </Card>

      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard label={t('stats.totalPlays')} value={userStats?.TotalPlays ?? '—'} icon={<Play24Regular />} loading={loading} />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard label={t('stats.watchTime')} value={userStats ? formatWatchTime(userStats.TotalWatchTime) : '—'} icon={<Clock24Regular />} loading={loading} />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard label={t('users.favoriteGenre')} value={userStats?.FavoriteGenre ?? '—'} icon={<Star24Regular />} loading={loading} />
        </Grid>
      </Grid>

      <Card sx={{ mb: 3, p: 2 }}>
        {loading ? (
          <Skeleton variant="rectangular" height={100} sx={{ borderRadius: 1 }} />
        ) : (
          <ActivityHeatmap data={heatmapData} metric={heatmapMetric} onMetricChange={setHeatmapMetric} />
        )}
      </Card>

      <ChartCard
        title={t('users.watchOverTime')}
        loading={loading}
        height={200}
        action={<MetricToggle value={watchMetric} onChange={setWatchMetric} />}
      >
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={watchOverTime} margin={{ top: 4, right: 4, bottom: 0, left: -20 }}>
            <defs>
              <linearGradient id="userGrad" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor={theme.palette.primary.main} stopOpacity={0.3} />
                <stop offset="95%" stopColor={theme.palette.primary.main} stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.05)" />
            <XAxis dataKey="date" tick={{ fontSize: 11, fill: theme.palette.text.secondary }} tickLine={false} axisLine={false} />
            <YAxis tick={{ fontSize: 11, fill: theme.palette.text.secondary }} tickLine={false} axisLine={false} />
            <Tooltip
              contentStyle={{ backgroundColor: theme.palette.background.paper, border: `1px solid ${theme.palette.divider}`, borderRadius: 8, fontSize: 12 }}
              formatter={(value) => watchMetric === 'duration' ? formatWatchTime(Number(value)) : value}
            />
            <Area
              type="monotone"
              dataKey={watchMetric === 'duration' ? 'duration' : 'plays'}
              stroke={theme.palette.primary.main}
              strokeWidth={2}
              fill="url(#userGrad)"
              name={watchMetric === 'duration' ? 'Durée' : t('common.plays')}
            />
          </AreaChart>
        </ResponsiveContainer>
      </ChartCard>

      <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 3, mt: 3 }}>
        <Tabs value={tab} onChange={(_, v) => setTab(v as number)}>
          <Tab label={t('users.watchHistory')} />
          <Tab label={t('library.genres')} />
        </Tabs>
      </Box>

      {tab === 0 && <DataTable data={activity} columns={columns} loading={loading} />}

      {tab === 1 && (
        <Card>
          <CardContent>
            {loading ? (
              Array.from({ length: 8 }).map((_, i) => <Skeleton key={i} variant="text" sx={{ mb: 0.5 }} />)
            ) : genres.length === 0 ? (
              <Typography variant="body2" color="text.secondary">{t('common.noData')}</Typography>
            ) : (
              genres.map((g) => (
                <Box key={g.Genre} sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', py: 0.75, borderBottom: '1px solid', borderColor: 'divider', '&:last-child': { borderBottom: 0 } }}>
                  <Typography variant="body2">{g.Genre}</Typography>
                  <Box sx={{ display: 'flex', gap: 1 }}>
                    <Chip label={`${g.Count} ${t('common.items')}`} size="small" sx={{ fontSize: 11, height: 20 }} />
                    <Chip label={`${g.PlayCount} ${t('common.plays')}`} size="small" color="primary" sx={{ fontSize: 11, height: 20 }} />
                  </Box>
                </Box>
              ))
            )}
          </CardContent>
        </Card>
      )}
    </>
  )
}
