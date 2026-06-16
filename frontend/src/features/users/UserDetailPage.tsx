import { useState, useEffect, useCallback, useMemo } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'
import {
  Grid, Alert, Box, Typography, Tabs, Tab, Avatar, Chip,
  Card, CardContent, Skeleton, CardMedia, ToggleButtonGroup, ToggleButton, Fade,
} from '@mui/material'
import { useTranslation } from 'react-i18next'
import { format, parseISO } from 'date-fns'
import { createColumnHelper, type ColumnDef } from '@tanstack/react-table'
import { LineChart } from '@mui/x-charts/LineChart'
import { BarChart } from '@mui/x-charts/BarChart'
import { useTheme } from '@mui/material/styles'
import PageHeader from '@/shared/components/PageHeader/PageHeader'
import StatCard from '@/shared/components/StatCard/StatCard'
import DataTable, { type FilterDef } from '@/shared/components/DataTable/DataTable'
import ChartCard from '@/shared/components/ChartCard/ChartCard'
import ActivityHeatmap from './components/ActivityHeatmap'
import GenreRadarChart from './components/GenreRadarChart'
import MetricToggle, { type ActivityMetric } from '@/shared/components/MetricToggle/MetricToggle'
import api from '@/lib/axios'
import type { Activity } from '@/shared/types/activity'
import type { UserStats, UserActivity } from '@/shared/types/user'
import type { GenreStat } from '@/shared/types/library'
import { Play24Regular, Clock24Regular, Star24Regular, VideoClip24Regular } from '@fluentui/react-icons'
import { formatWatchTime } from '@/shared/utils/formatWatchTime'
import { getDateLocale } from '@/lib/dateLocale'

const CHART_COLORS = ['#a78bfa', '#7c3aed', '#6d28d9', '#5b21b6', '#4c1d95', '#8b5cf6', '#c4b5fd', '#ede9fe', '#a78bfa', '#7c3aed']

const col = createColumnHelper<Activity>()

export default function UserDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { t } = useTranslation()
  const theme = useTheme()
  const [searchParams, setSearchParams] = useSearchParams()

  const USER_VIEWS = ['history', 'genres', 'lastplayed'] as const
  const currentView = searchParams.get('view') ?? 'history'
  const tab = Math.max(0, USER_VIEWS.indexOf(currentView as never))
  const setTab = (idx: number) => setSearchParams({ view: USER_VIEWS[idx] }, { replace: true })
  const [userStats, setUserStats] = useState<UserStats | null>(null)
  const [activity, setActivity] = useState<Activity[]>([])
  const [heatmapData, setHeatmapData] = useState<UserActivity[]>([])
  const [genres, setGenres] = useState<GenreStat[]>([])
  const [watchOverTime, setWatchOverTime] = useState<{ date: string; plays: number; duration: number }[]>([])
  const [genreBarData, setGenreBarData] = useState<{ genre: string; plays: number }[]>([])
  const [lastPlayed, setLastPlayed] = useState<Activity[]>([])
  const [globalStats, setGlobalStats] = useState<{ Plays?: number; total_playback_duration?: number } | null>(null)
  const [heatmapMetric, setHeatmapMetric] = useState<ActivityMetric>('count')

  const [watchMetric, setWatchMetric] = useState<ActivityMetric>('duration')
  const [watchDays, setWatchDays] = useState<number>(30)
  const [chartVisible, setChartVisible] = useState(true)

  const handleWatchDaysChange = (_: React.MouseEvent<HTMLElement>, v: number | null) => {
    if (v === null) return
    setChartVisible(false)
    setTimeout(() => {
      setWatchDays(v)
      setChartVisible(true)
    }, 180)
  }
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(() => {
    if (!id) return
    setLoading(true)
    setError(null)
    Promise.all([
      api.get(`/stats/getUserStats?userId=${id}`),
      api.get(`/stats/getUserActivity?userId=${id}`),
      api.get(`/stats/getUserActivityByDate?userId=${id}`),
      api.get(`/stats/getUserGenreStats?userId=${id}`),
      api.get(`/stats/getWatchStatisticsOverTime?userId=${id}&days=0`),
      api.get(`/stats/getGenreUserStats?userid=${id}&size=10&page=1`),
      api.post(`/stats/getUserLastPlayed`, { userid: id }),
      api.post(`/stats/getGlobalUserStats`, { userid: id }),
    ])
      .then(([statsRes, activityRes, heatmapRes, genreRes, overTimeRes, genreBarRes, lastPlayedRes, globalStatsRes]) => {
        setUserStats(statsRes.data)
        setActivity(activityRes.data ?? [])
        setHeatmapData(heatmapRes.data ?? [])
        setGenres(genreRes.data ?? [])
        setWatchOverTime(overTimeRes.data ?? [])
        const barResults: { genre: string; plays: number }[] = (genreBarRes.data?.results ?? [])
          .map((r: { genre: string; plays: string | number }) => ({ genre: r.genre, plays: Number(r.plays) }))
        setGenreBarData(barResults)
        setLastPlayed((lastPlayedRes.data ?? []).slice(0, 12))
        setGlobalStats(globalStatsRes.data ?? null)
      })
      .catch(() => setError(t('common.loadError')))
      .finally(() => setLoading(false))
  }, [id, t])

  useEffect(() => { load() }, [load])

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const columns: ColumnDef<Activity, any>[] = [
    col.accessor('NowPlayingItemName', {
      header: t('activity.item'),
      cell: (i) => {
        const row = i.row.original
        const label = row.SeriesName ? `${row.SeriesName} — ${i.getValue() as string}` : i.getValue() as string
        return (
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Box sx={{
              width: 45, height: 30, borderRadius: 0.75, overflow: 'hidden', flexShrink: 0,
              bgcolor: 'rgba(255,255,255,0.06)', position: 'relative',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
            }}>
              <VideoClip24Regular style={{ opacity: 0.4, fontSize: 16 }} />
              <Box
                component="img"
                src={`/proxy/Items/Images/Primary/?id=${encodeURIComponent(row.ParentId ?? row.ItemId)}&fillWidth=90&quality=80`}
                onError={(e: React.SyntheticEvent<HTMLImageElement>) => {
                  if (row.ParentId && e.currentTarget.src.includes(encodeURIComponent(row.ParentId))) {
                    e.currentTarget.src = `/proxy/Items/Images/Primary/?id=${encodeURIComponent(row.ItemId)}&fillWidth=90&quality=80`
                  } else {
                    e.currentTarget.style.display = 'none'
                  }
                }}
                sx={{ position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover' }}
              />
            </Box>
            <Typography variant="body2" noWrap title={label}>{label}</Typography>
          </Box>
        )
      },
    }),
    col.accessor('Client', { header: t('activity.client') }),
    col.accessor('PlayMethod', { header: t('activity.method'), cell: (i) => (i.getValue() as string | undefined) ?? '—' }),
    col.accessor('ActivityDateInserted', {
      header: t('activity.date'),
      cell: (i) => { try { return format(parseISO(i.getValue() as string), 'dd/MM/yyyy HH:mm', { locale: getDateLocale() }) } catch { return i.getValue() as string } },
    }),
    col.accessor('PlayDuration', {
      header: t('activity.duration'),
      cell: (i) => { const s = Math.floor(((i.getValue() as number) ?? 0) / 10_000_000); const h = Math.floor(s / 3600); const m = Math.floor((s % 3600) / 60); return h > 0 ? `${h}${t('time.hourShort')} ${m}${t('time.minuteShort')}` : `${m}${t('time.minuteShort')}` },
    }),
  ]

  const filteredWatchOverTime = useMemo(() => {
    if (watchDays === 0) return watchOverTime
    const cutoff = new Date()
    cutoff.setDate(cutoff.getDate() - watchDays)
    const cutoffStr = cutoff.toISOString().slice(0, 10)
    return watchOverTime.filter((d) => d.date >= cutoffStr)
  }, [watchOverTime, watchDays])

  const activityFilterDefs = useMemo<FilterDef[]>(() => [
    { id: 'Client', label: t('activity.client'), type: 'select' },
    { id: 'PlayMethod', label: t('activity.method'), type: 'select' },
    { id: 'NowPlayingItemType', label: t('activity.mediaType', 'Type'), type: 'select' },
    { id: 'DeviceName', label: t('activity.device'), type: 'select' },
    {
      id: 'PlayDuration',
      label: t('activity.duration'),
      type: 'range',
      unit: 'min',
      transform: (ticks: number) => Math.floor(ticks / 10_000_000 / 60),
    },
  ], [t])

  const username = userStats?.UserName ?? id ?? '...'

  return (
    <>
      <PageHeader title={username} onRefresh={load} loading={loading} />
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
                {t('users.lastSeen')}: {userStats?.LastSeen ? format(parseISO(userStats.LastSeen), 'dd/MM/yyyy HH:mm', { locale: getDateLocale() }) : '—'}
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
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard
            label={t('stats.recentPlays', 'Plays (24h)')}
            value={globalStats?.Plays ?? '—'}
            icon={<Star24Regular />}
            loading={loading}
          />
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
        empty={filteredWatchOverTime.length === 0}
        height={200}
        action={
          <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
            <ToggleButtonGroup
              value={watchDays}
              exclusive
              size="small"
              onChange={handleWatchDaysChange}
              sx={{ '& .MuiToggleButton-root': { px: 1.5, py: 0.25, fontSize: 12, textTransform: 'none' } }}
            >
              <ToggleButton value={7}>7d</ToggleButton>
              <ToggleButton value={30}>30d</ToggleButton>
              <ToggleButton value={90}>90d</ToggleButton>
              <ToggleButton value={0}>{t('common.all', 'All')}</ToggleButton>
            </ToggleButtonGroup>
            <MetricToggle value={watchMetric} onChange={setWatchMetric} />
          </Box>
        }
      >
        <Fade in={chartVisible} timeout={180}>
          <Box>
            <LineChart
              xAxis={[{ data: filteredWatchOverTime.map((d) => d.date), scaleType: 'point' }]}
              series={[{
                data: filteredWatchOverTime.map((d) => (watchMetric === 'duration' ? d.duration : d.plays)),
                area: true,
                label: watchMetric === 'duration' ? t('stats.watchTime') : t('common.plays'),
                valueFormatter: (v) => watchMetric === 'duration' ? formatWatchTime(v ?? 0) : String(v ?? 0),
                color: theme.palette.primary.main,
                showMark: false,
              }]}
              height={200}
              sx={{ width: '100%' }}
              grid={{ horizontal: true }}
              slotProps={{ legend: { hidden: true } }}
              skipAnimation
            />
          </Box>
        </Fade>
      </ChartCard>

      <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 3, mt: 3 }}>
        <Tabs value={tab} onChange={(_, v) => setTab(v as number)}>
          <Tab label={t('users.watchHistory')} />
          <Tab label={t('library.genres')} />
          <Tab label={t('users.lastPlayed', 'Derniers regardés')} />
        </Tabs>
      </Box>

      {tab === 0 && (
        <DataTable
          data={activity}
          columns={columns}
          loading={loading}
          filterDefs={activityFilterDefs}
          onRefresh={load}
        />
      )}

      {tab === 1 && (
        <>
          <Card sx={{ mb: 2 }}>
            <CardContent>
              {loading ? (
                <Skeleton variant="rectangular" width="100%" height={320} sx={{ borderRadius: 2 }} />
              ) : genres.length === 0 ? (
                <Typography variant="body2" color="text.secondary" sx={{ py: 4, textAlign: 'center' }}>
                  {t('common.noData')}
                </Typography>
              ) : (
                <GenreRadarChart genres={genres} />
              )}
            </CardContent>
          </Card>

          <ChartCard
            title={t('stats.genreDetail', 'Détail par genre')}
            loading={loading}
            empty={genreBarData.length === 0}
            height={240}
          >
            <BarChart
              layout="horizontal"
              yAxis={[{ data: genreBarData.map((d) => d.genre), scaleType: 'band' }]}
              xAxis={[{ label: t('common.plays') }]}
              series={[{
                data: genreBarData.map((d) => d.plays),
                label: t('common.plays'),
                valueFormatter: (v) => String(v ?? 0),
                color: CHART_COLORS[0],
              }]}
              height={240}
              margin={{ left: 120 }}
              sx={{ width: '100%' }}
              grid={{ vertical: true }}
              slotProps={{ legend: { hidden: true } }}
            />
          </ChartCard>
        </>
      )}

      {tab === 2 && (
        loading ? (
          <Grid container spacing={2}>
            {Array.from({ length: 8 }).map((_, i) => (
              <Grid key={i} size={{ xs: 6, sm: 4, md: 3 }}>
                <Skeleton variant="rectangular" sx={{ borderRadius: 2, aspectRatio: '2/3' }} />
              </Grid>
            ))}
          </Grid>
        ) : lastPlayed.length === 0 ? (
          <Typography variant="body2" color="text.secondary" sx={{ py: 4, textAlign: 'center' }}>
            {t('common.noData')}
          </Typography>
        ) : (
          <Grid container spacing={2}>
            {lastPlayed.map((item) => {
              const secs = Math.floor(((item.PlayDuration ?? 0)) / 10_000_000)
              const h = Math.floor(secs / 3600)
              const m = Math.floor((secs % 3600) / 60)
              const durationLabel = h > 0
                ? `${h}${t('time.hourShort')} ${m}${t('time.minuteShort')}`
                : `${m}${t('time.minuteShort')}`
              const itemId = item.NowPlayingItemType === 'Episode' ? item.EpisodeId ?? item.Id : item.Id
              return (
                <Grid key={item.Id} size={{ xs: 6, sm: 4, md: 3 }}>
                  <Card sx={{ height: '100%' }}>
                    <CardMedia
                      component="img"
                      src={`/proxy/Items/Images/Primary/?id=${itemId}&fillWidth=300&quality=80`}
                      alt={item.NowPlayingItemName}
                      sx={{ aspectRatio: '2/3', objectFit: 'cover' }}
                      onError={(e) => { (e.currentTarget as HTMLImageElement).style.display = 'none' }}
                    />
                    <CardContent sx={{ p: 1, '&:last-child': { pb: 1 } }}>
                      <Typography variant="caption" fontWeight={700} noWrap display="block" title={item.NowPlayingItemName}>
                        {item.SeriesName ? `${item.SeriesName}` : item.NowPlayingItemName}
                      </Typography>
                      {item.SeriesName && (
                        <Typography variant="caption" color="text.secondary" noWrap display="block">
                          {item.NowPlayingItemName}
                        </Typography>
                      )}
                      <Typography variant="caption" color="text.secondary" display="block">
                        {(() => { try { return format(parseISO(item.ActivityDateInserted), 'dd/MM/yy', { locale: getDateLocale() }) } catch { return '—' } })()}
                        {' · '}{durationLabel}
                      </Typography>
                    </CardContent>
                  </Card>
                </Grid>
              )
            })}
          </Grid>
        )
      )}
    </>
  )
}
