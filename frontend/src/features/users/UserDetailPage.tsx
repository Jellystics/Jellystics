import { useState, useEffect, useCallback, useMemo } from 'react'
import { useParams, useSearchParams, useNavigate } from 'react-router-dom'
import {
  Grid, Alert, Box, Typography, Tabs, Tab, Avatar, Chip, Tooltip, Pagination,
  Card, CardContent, Skeleton, CardMedia, ToggleButtonGroup, ToggleButton, Fade,
} from '@mui/material'
import { useTranslation } from 'react-i18next'
import { format, parseISO } from 'date-fns'
import { createColumnHelper, type ColumnDef } from '@tanstack/react-table'
import { LineChart } from '@mui/x-charts/LineChart'
import { BarChart } from '@mui/x-charts/BarChart'
import { useTheme, alpha } from '@mui/material/styles'
import PageHeader from '@/shared/components/PageHeader/PageHeader'
import StatCard from '@/shared/components/StatCard/StatCard'
import DataTable, { type FilterDef } from '@/shared/components/DataTable/DataTable'
import ChartCard from '@/shared/components/ChartCard/ChartCard'
import ActivityHeatmap from './components/ActivityHeatmap'
import MetricToggle, { type ActivityMetric } from '@/shared/components/MetricToggle/MetricToggle'
import api from '@/lib/axios'
import type { Activity } from '@/shared/types/activity'
import type { UserStats, UserActivity } from '@/shared/types/user'

type GenreRow = { genre: string; plays: number; duration: number }
import { Play24Regular, Clock24Regular, Star24Regular, VideoClip24Regular, ArrowLeft24Regular } from '@fluentui/react-icons'
import { formatWatchTime } from '@/shared/utils/formatWatchTime'
import { ticksToMinutes } from '@/shared/utils/formatTicks'
import { getDateLocale } from '@/lib/dateLocale'
import { useChartColors } from '@/lib/chartColors'
import { getActivityImageUrl } from '@/shared/utils/activityImage'

interface BingedSeries {
  seriesId: string
  seriesName: string
  bingeCount: number
  totalEpisodesWatched: number
  avgEpisodesPerBinge: number
}

interface BingeStats {
  totalBingeSessions: number
  topBingedSeries: BingedSeries[]
  topBingeUsers: { userId: string; userName: string; bingeCount: number; totalEpisodesWatched: number }[]
}

interface HeatmapCell {
  day: number
  hour: number
  plays: number
  duration: number
}

interface WatchHeatmapData {
  cells: HeatmapCell[]
  maxPlays: number
  maxDuration: number
}

const col = createColumnHelper<Activity>()

export default function UserDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { t } = useTranslation()
  const theme = useTheme()
  const CHART_COLORS = useChartColors()
  const [searchParams, setSearchParams] = useSearchParams()
  const navigate = useNavigate()

  const USER_VIEWS = ['history', 'genres', 'lastplayed'] as const
  const currentView = searchParams.get('view') ?? 'history'
  const tab = Math.max(0, USER_VIEWS.indexOf(currentView as never))
  const setTab = (idx: number) => setSearchParams({ view: USER_VIEWS[idx] }, { replace: true })
  const [userStats, setUserStats] = useState<UserStats | null>(null)
  const [activity, setActivity] = useState<Activity[]>([])
  const [heatmapData, setHeatmapData] = useState<UserActivity[]>([])
  const [watchOverTime, setWatchOverTime] = useState<{ date: string; plays: number; duration: number }[]>([])
  const [lastPlayed, setLastPlayed] = useState<Activity[]>([])
  const [genreMovie, setGenreMovie] = useState<GenreRow[]>([])
  const [genreEpisode, setGenreEpisode] = useState<GenreRow[]>([])
  const [genreAudio, setGenreAudio] = useState<GenreRow[]>([])
  const [genreLoading, setGenreLoading] = useState(false)
  const [genreMetric, setGenreMetric] = useState<ActivityMetric>('count')
  const [globalStats, setGlobalStats] = useState<{ Plays?: number; total_playback_duration?: number } | null>(null)
  const [heatmapMetric, setHeatmapMetric] = useState<ActivityMetric>('count')

  const [byHour, setByHour] = useState<{ hour: number; plays: number; duration: number }[]>([])
  const [byDay, setByDay] = useState<{ day: number; plays: number; duration: number }[]>([])
  const [hourMetric, setHourMetric] = useState<ActivityMetric>('count')
  const [dayMetric, setDayMetric] = useState<ActivityMetric>('count')

  const [watchMetric, setWatchMetric] = useState<ActivityMetric>('duration')
  const [watchDays, setWatchDays] = useState<number>(30)
  const [chartVisible, setChartVisible] = useState(true)

  const [bingeStats, setBingeStats] = useState<BingeStats | null>(null)
  const [bingeDays, setBingeDays] = useState<number>(30)
  const [bingePage, setBingePage] = useState(0)
  const BINGE_PAGE_SIZE = 5
  const [watchHeatmap, setWatchHeatmap] = useState<WatchHeatmapData | null>(null)
  const [diversity, setDiversity] = useState<any>(null)

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
      api.get(`/stats/getWatchStatisticsOverTime?userId=${id}&days=0`),
      api.post(`/stats/getUserLastPlayed`, { userid: id }),
      api.post(`/stats/getGlobalUserStats`, { userid: id }),
      api.get(`/stats/getPopularHourOfDay?days=0&userId=${id}`),
      api.get(`/stats/getPopularDayOfWeek?days=0&userId=${id}`),
    ])
      .then(([statsRes, activityRes, heatmapRes, overTimeRes, lastPlayedRes, globalStatsRes, hourRes, dayRes]) => {
        setUserStats(statsRes.data)
        setActivity(activityRes.data ?? [])
        setHeatmapData(heatmapRes.data ?? [])
        setWatchOverTime(overTimeRes.data ?? [])
        setLastPlayed((lastPlayedRes.data ?? []).slice(0, 12))
        setGlobalStats(globalStatsRes.data ?? null)
        setByHour(hourRes.data ?? [])
        setByDay(dayRes.data ?? [])
      })
      .catch(() => setError(t('common.loadError')))
      .finally(() => setLoading(false))
  }, [id, t])

  useEffect(() => { load() }, [load])

  useEffect(() => {
    if (!id) return
    api.get(`/stats/getBingeStats?days=${bingeDays}&userId=${id}`)
      .then((res) => { setBingeStats(res.data); setBingePage(0) })
      .catch(() => setBingeStats(null))
  }, [id, bingeDays])

  useEffect(() => {
    if (!id) return
    api.get(`/stats/getWatchHeatmap?days=0&userId=${id}`)
      .then((res) => setWatchHeatmap(res.data))
      .catch(() => setWatchHeatmap(null))
  }, [id])

  useEffect(() => {
    if (!id) return
    api.get(`/stats/getViewingDiversity?days=0&userId=${id}`)
      .then((res) => setDiversity(res.data))
      .catch(() => setDiversity(null))
  }, [id])

  // Lazy-load genre breakdown by media type when the genres tab is visited
  useEffect(() => {
    if (tab !== 1 || !id) return
    setGenreLoading(true)
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const map = (data: any) => (data?.results ?? []).map((r: any) => ({
      genre: r.genre as string,
      plays: Number(r.plays),
      duration: Number(r.duration),
    }))
    Promise.all([
      api.get(`/stats/getGenreUserStats?userid=${id}&type=Movie&size=8&page=1`),
      api.get(`/stats/getGenreUserStats?userid=${id}&type=Episode&size=8&page=1`),
      api.get(`/stats/getGenreUserStats?userid=${id}&type=Audio&size=8&page=1`),
    ]).then(([movieRes, episodeRes, audioRes]) => {
      setGenreMovie(map(movieRes.data))
      setGenreEpisode(map(episodeRes.data))
      setGenreAudio(map(audioRes.data))
    }).catch(() => {}).finally(() => setGenreLoading(false))
  }, [tab, id])

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const columns: ColumnDef<Activity, any>[] = [
    col.accessor('NowPlayingItemName', {
      header: t('activity.item'),
      cell: (i) => {
        const row = i.row.original
        const label = row.SeriesName ? `${row.SeriesName} — ${i.getValue() as string}` : i.getValue() as string
        return (
          <Box
            onClick={(e) => { e.stopPropagation(); navigate(`/items/${row.ItemId}`) }}
            sx={{ display: 'flex', alignItems: 'center', gap: 1, cursor: 'pointer', '&:hover .itemname': { textDecoration: 'underline' } }}
          >
            <Box sx={{
              width: 45, height: 30, borderRadius: 0.75, overflow: 'hidden', flexShrink: 0,
              bgcolor: 'rgba(255,255,255,0.06)', position: 'relative',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
            }}>
              <VideoClip24Regular style={{ opacity: 0.4, fontSize: 16 }} />
              {getActivityImageUrl(row, 90) && (
                <Box
                  component="img"
                  src={getActivityImageUrl(row, 90)!}
                  onError={(e: React.SyntheticEvent<HTMLImageElement>) => { e.currentTarget.style.display = 'none' }}
                  sx={{ position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover' }}
                />
              )}
            </Box>
            <Typography className="itemname" variant="body2" noWrap title={label}>{label}</Typography>
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
      cell: (i) => formatWatchTime(ticksToMinutes(i.getValue() as number)),
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
      transform: (ticks: number) => ticksToMinutes(ticks),
    },
  ], [t])

  const username = userStats?.UserName ?? id ?? '...'

  return (
    <>
      <Box
        component="button"
        onClick={() => navigate(-1 as any)}
        style={{ all: 'unset', cursor: 'pointer' }}
      >
        <Typography variant="body2" color="primary.main" sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mb: 2 }}>
          <ArrowLeft24Regular style={{ fontSize: 18 }} />
          {t('common.back', 'Back')}
        </Typography>
      </Box>
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

      {/* Watch Patterns Heatmap */}
      <Card sx={{ mb: 3, p: 2 }}>
        <Typography variant="h6" sx={{ fontWeight: 700, mb: 2 }}>
          {t('insights.watchHeatmap', 'Watch Patterns')}
        </Typography>
        {!watchHeatmap ? (
          <Skeleton variant="rectangular" height={220} sx={{ borderRadius: 1 }} />
        ) : (
          <Box sx={{ overflowX: 'auto' }}>
            <Box sx={{ display: 'grid', gridTemplateColumns: 'auto repeat(24, 1fr)', gridTemplateRows: 'auto repeat(7, 28px)', gap: '2px', minWidth: 600 }}>
              {/* Hour labels row */}
              <Box />
              {Array.from({ length: 24 }, (_, h) => (
                <Box key={`h-${h}`} sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                  <Typography variant="caption" sx={{ fontSize: 10, color: 'text.secondary' }}>{h}</Typography>
                </Box>
              ))}
              {/* Day rows */}
              {[
                t('days.short.sun', 'Sun'), t('days.short.mon', 'Mon'), t('days.short.tue', 'Tue'),
                t('days.short.wed', 'Wed'), t('days.short.thu', 'Thu'), t('days.short.fri', 'Fri'),
                t('days.short.sat', 'Sat'),
              ].map((dayLabel, dayIdx) => (
                <Box key={`day-${dayIdx}`} sx={{ display: 'contents' }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', pr: 1 }}>
                    <Typography variant="caption" sx={{ fontSize: 11, color: 'text.secondary', fontWeight: 700 }}>{dayLabel}</Typography>
                  </Box>
                  {Array.from({ length: 24 }, (_, h) => {
                    const cell = watchHeatmap.cells.find((c) => c.day === dayIdx && c.hour === h)
                    const plays = cell?.plays ?? 0
                    const duration = cell?.duration ?? 0
                    const intensity = watchHeatmap.maxPlays > 0 ? plays / watchHeatmap.maxPlays : 0
                    return (
                      <Tooltip
                        key={`${dayIdx}-${h}`}
                        title={`${dayLabel} ${h}:00 — ${plays} plays, ${formatWatchTime(duration)}`}
                        arrow
                        placement="top"
                      >
                        <Box
                          sx={{
                            borderRadius: 0.5,
                            bgcolor: plays > 0
                              ? alpha(CHART_COLORS[0], Math.max(0.1, intensity))
                              : alpha(theme.palette.text.primary, 0.04),
                            cursor: 'default',
                            '&:hover': { outline: '2px solid', outlineColor: 'text.disabled', outlineOffset: -2, zIndex: 1 },
                          }}
                        />
                      </Tooltip>
                    )
                  })}
                </Box>
              ))}
            </Box>
          </Box>
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
              slotProps={{ legend: { hidden: true } as any }}
              skipAnimation
            />
          </Box>
        </Fade>
      </ChartCard>

      <Grid container spacing={2} sx={{ mt: 3 }}>
        <Grid size={{ xs: 12, md: 6 }}>
          <ChartCard
            title={t('stats.playsByHour')}
            loading={loading}
            empty={!loading && byHour.length === 0}
            height={200}
            action={<MetricToggle value={hourMetric} onChange={setHourMetric} />}
          >
            <BarChart
              xAxis={[{ data: Array.from({ length: 24 }, (_, h) => `${h}h`), scaleType: 'band' }]}
              series={[{
                data: Array.from({ length: 24 }, (_, h) => {
                  const found = byHour.find((d) => d.hour === h)
                  return hourMetric === 'duration' ? (found?.duration ?? 0) : (found?.plays ?? 0)
                }),
                color: CHART_COLORS[0],
                valueFormatter: (v) => hourMetric === 'duration' ? `${v ?? 0} min` : String(v ?? 0),
              }]}
              height={200}
              sx={{ width: '100%' }}
              grid={{ horizontal: true }}
              slotProps={{ legend: { hidden: true } as any }}
            />
          </ChartCard>
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <ChartCard
            title={t('stats.playsByDayOfWeek')}
            loading={loading}
            empty={!loading && byDay.length === 0}
            height={200}
            action={<MetricToggle value={dayMetric} onChange={setDayMetric} />}
          >
            <BarChart
              xAxis={[{
                data: [
                  t('days.short.sun'), t('days.short.mon'), t('days.short.tue'), t('days.short.wed'),
                  t('days.short.thu'), t('days.short.fri'), t('days.short.sat'),
                ],
                scaleType: 'band',
              }]}
              series={[{
                data: Array.from({ length: 7 }, (_, d) => {
                  const found = byDay.find((r) => r.day === d)
                  return dayMetric === 'duration' ? (found?.duration ?? 0) : (found?.plays ?? 0)
                }),
                color: CHART_COLORS[0],
                valueFormatter: (v) => dayMetric === 'duration' ? `${v ?? 0} min` : String(v ?? 0),
              }]}
              height={200}
              sx={{ width: '100%' }}
              grid={{ horizontal: true }}
              slotProps={{ legend: { hidden: true } as any }}
            />
          </ChartCard>
        </Grid>
      </Grid>

      {/* Binge Stats */}
      <Card sx={{ mb: 3, mt: 3, p: 2 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
          <Typography variant="h6" sx={{ fontWeight: 700 }}>
            {t('insights.bingeStats', 'Binge-watching')}
          </Typography>
          <ToggleButtonGroup
            value={bingeDays}
            exclusive
            size="small"
            onChange={(_, v) => { if (v !== null) setBingeDays(v) }}
            sx={{ '& .MuiToggleButton-root': { px: 1.5, py: 0.25, fontSize: 12, textTransform: 'none' } }}
          >
            <ToggleButton value={7}>7d</ToggleButton>
            <ToggleButton value={30}>30d</ToggleButton>
            <ToggleButton value={90}>90d</ToggleButton>
            <ToggleButton value={0}>{t('common.all', 'All')}</ToggleButton>
          </ToggleButtonGroup>
        </Box>
        <Grid container spacing={2}>
          <Grid size={{ xs: 12, sm: 4 }}>
            <StatCard
              label={t('insights.bingeSessions', 'Binge sessions')}
              value={bingeStats?.totalBingeSessions ?? '—'}
              icon={<Play24Regular />}
              loading={!bingeStats && loading}
            />
          </Grid>
          <Grid size={{ xs: 12, sm: 8 }}>
            {bingeStats && bingeStats.topBingedSeries.length > 0 ? (
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 700 }}>
                  {t('insights.topBingedSeries', 'Top binged series')}
                </Typography>
                {bingeStats.topBingedSeries
                  .slice(bingePage * BINGE_PAGE_SIZE, (bingePage + 1) * BINGE_PAGE_SIZE)
                  .map((s) => (
                  <Box
                    key={s.seriesId}
                    sx={{
                      display: 'flex', alignItems: 'center', gap: 1.5,
                      p: 1, borderRadius: 1,
                      bgcolor: alpha(theme.palette.primary.main, 0.06),
                      cursor: 'pointer',
                    }}
                    onClick={() => navigate(`/items/${s.seriesId}`)}
                  >
                    <Box
                      component="img"
                      src={`/proxy/Items/Images/Primary/?id=${encodeURIComponent(s.seriesId)}&fillWidth=60&quality=80`}
                      sx={{ width: 40, height: 56, borderRadius: 0.5, objectFit: 'cover', flexShrink: 0, bgcolor: 'action.hover' }}
                      onError={(e: React.SyntheticEvent<HTMLImageElement>) => { e.currentTarget.style.display = 'none' }}
                    />
                    <Typography variant="body2" sx={{ fontWeight: 700, flex: 1, minWidth: 0 }} noWrap>
                      {s.seriesName}
                    </Typography>
                    <Chip label={`${s.bingeCount} binges`} size="small" color="primary" variant="outlined" />
                    <Typography variant="caption" color="text.secondary" noWrap>
                      ~{s.avgEpisodesPerBinge.toFixed(1)} ep/binge
                    </Typography>
                  </Box>
                ))}
                {bingeStats.topBingedSeries.length > BINGE_PAGE_SIZE && (
                  <Box sx={{ display: 'flex', justifyContent: 'center', mt: 1 }}>
                    <Pagination
                      count={Math.ceil(bingeStats.topBingedSeries.length / BINGE_PAGE_SIZE)}
                      page={bingePage + 1}
                      onChange={(_, p) => setBingePage(p - 1)}
                      size="small"
                    />
                  </Box>
                )}
              </Box>
            ) : (
              <Typography variant="body2" color="text.secondary" sx={{ py: 2, textAlign: 'center' }}>
                {t('common.noData')}
              </Typography>
            )}
          </Grid>
        </Grid>
      </Card>

      {/* Viewing Diversity */}
      {diversity && (diversity.uniqueGenres > 0 || diversity.uniqueLibraries > 0) && (
        <Card sx={{ mb: 3, mt: 3, p: 2 }}>
          <Typography variant="h6" sx={{ fontWeight: 700, mb: 2 }}>
            {t('insights.viewingDiversity', 'Viewing Diversity')}
          </Typography>
          <Grid container spacing={2} sx={{ mb: 2 }}>
            <Grid size={{ xs: 6, sm: 3 }}>
              <StatCard
                label={t('insights.score', 'Score')}
                value={`${Math.round((diversity.diversityScore ?? 0) * 100)}%`}
                icon={<Star24Regular />}
              />
            </Grid>
            <Grid size={{ xs: 6, sm: 3 }}>
              <StatCard
                label={t('insights.genres', 'Genres')}
                value={diversity.uniqueGenres ?? 0}
                icon={<VideoClip24Regular />}
              />
            </Grid>
            <Grid size={{ xs: 6, sm: 3 }}>
              <StatCard
                label={t('insights.libraries', 'Libraries')}
                value={diversity.uniqueLibraries ?? 0}
                icon={<Play24Regular />}
              />
            </Grid>
            <Grid size={{ xs: 6, sm: 3 }}>
              <StatCard
                label={t('insights.uniqueItems', 'Unique items')}
                value={diversity.uniqueItems ?? 0}
                icon={<Clock24Regular />}
              />
            </Grid>
          </Grid>
          {diversity.genreBreakdown?.length > 0 && (
            <Box>
              <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 700, mb: 1 }}>
                {t('insights.genreBreakdown', 'Genre breakdown')}
              </Typography>
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                {diversity.genreBreakdown.map((g: any) => (
                  <Chip
                    key={g.genre}
                    label={`${g.genre} (${g.percent}%)`}
                    size="small"
                    variant="outlined"
                  />
                ))}
              </Box>
            </Box>
          )}
        </Card>
      )}

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

      {tab === 1 && (() => {
        const sections: { label: string; data: GenreRow[]; color: string }[] = [
          { label: t('stats.mediaType.movie', 'Films'), data: genreMovie, color: CHART_COLORS[0] },
          { label: t('stats.mediaType.series', 'Séries'), data: genreEpisode, color: CHART_COLORS[1] },
          { label: t('stats.mediaType.audio', 'Musique'), data: genreAudio, color: CHART_COLORS[2] },
        ].filter((s) => genreLoading || s.data.length > 0)

        const fmtDuration = (secs: number) => {
          const h = Math.floor(secs / 3600)
          const m = Math.floor((secs % 3600) / 60)
          return h > 0 ? `${h}${t('time.hourShort')} ${m}${t('time.minuteShort')}` : `${m}${t('time.minuteShort')}`
        }

        return (
          <>
            <Box sx={{ display: 'flex', justifyContent: 'flex-end', mb: 2 }}>
              <MetricToggle value={genreMetric} onChange={setGenreMetric} />
            </Box>
            {sections.length === 0 && !genreLoading ? (
              <Typography variant="body2" color="text.secondary" sx={{ py: 4, textAlign: 'center' }}>
                {t('common.noData')}
              </Typography>
            ) : (
              <Grid container spacing={2}>
                {(genreLoading ? [
                  { label: t('stats.mediaType.movie', 'Films'), data: [], color: CHART_COLORS[0] },
                  { label: t('stats.mediaType.series', 'Séries'), data: [], color: CHART_COLORS[1] },
                  { label: t('stats.mediaType.audio', 'Musique'), data: [], color: CHART_COLORS[2] },
                ] : sections).map(({ label, data, color }) => {
                  const chartHeight = Math.max(160, data.length * 36 + 40)
                  return (
                    <Grid key={label} size={{ xs: 12, md: 4 }}>
                      <ChartCard title={label} loading={genreLoading} empty={!genreLoading && data.length === 0} height={chartHeight}>
                        <BarChart
                          layout="horizontal"
                          yAxis={[{ data: data.map((d) => d.genre), scaleType: 'band' }]}
                          xAxis={[{ tickMinStep: 1 }]}
                          series={[{
                            data: data.map((d) => genreMetric === 'duration' ? d.duration : d.plays),
                            color,
                            valueFormatter: (v) => genreMetric === 'duration' ? fmtDuration(v ?? 0) : String(v ?? 0),
                          }]}
                          height={chartHeight}
                          margin={{ left: 110, right: 16, top: 8, bottom: 24 }}
                          sx={{ width: '100%' }}
                          grid={{ vertical: true }}
                          slotProps={{ legend: { hidden: true } as any }}
                        />
                      </ChartCard>
                    </Grid>
                  )
                })}
              </Grid>
            )}
          </>
        )
      })()}

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
              const durationLabel = formatWatchTime(ticksToMinutes(item.PlayDuration ?? 0))
              const itemId = item.NowPlayingItemType === 'Episode' ? item.EpisodeId ?? item.Id : item.Id
              return (
                <Grid key={item.Id} size={{ xs: 6, sm: 4, md: 3 }}>
                  <Card
                    onClick={() => navigate(`/items/${item.ItemId}`)}
                    sx={{ height: '100%', cursor: 'pointer', transition: 'opacity 150ms', '&:hover': { opacity: 0.8 } }}
                  >
                    <CardMedia
                      component="img"
                      src={`/proxy/Items/Images/Primary/?id=${itemId}&fillWidth=300&quality=80`}
                      alt={item.NowPlayingItemName}
                      sx={{ aspectRatio: '2/3', objectFit: 'cover' }}
                      onError={(e) => { (e.currentTarget as HTMLImageElement).style.display = 'none' }}
                    />
                    <CardContent sx={{ p: 1, '&:last-child': { pb: 1 } }}>
                      <Typography variant="caption" sx={{ fontWeight: 700, display: 'block' }} noWrap title={item.NowPlayingItemName}>
                        {item.SeriesName ? `${item.SeriesName}` : item.NowPlayingItemName}
                      </Typography>
                      {item.SeriesName && (
                        <Typography variant="caption" color="text.secondary" noWrap sx={{ display: 'block' }}>
                          {item.NowPlayingItemName}
                        </Typography>
                      )}
                      <Typography variant="caption" color="text.secondary" sx={{ display: 'block' }}>
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
