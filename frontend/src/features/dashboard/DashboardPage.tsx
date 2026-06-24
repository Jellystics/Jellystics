import { useState, useMemo } from 'react'
import type { ReactNode } from 'react'
import { Grid, Alert, Box } from '@mui/material'
import {
  Play24Regular, Clock24Regular, People24Regular,
  VideoClip24Regular, Library24Regular, Apps24Regular,
} from '@fluentui/react-icons'
import { useTranslation } from 'react-i18next'

import { PieChart } from '@mui/x-charts/PieChart'
import { BarChart } from '@mui/x-charts/BarChart'
import { ScatterChart } from '@mui/x-charts/ScatterChart'
import PageHeader from '@/shared/components/PageHeader/PageHeader'
import StatCard from '@/shared/components/StatCard/StatCard'
import ChartCard from '@/shared/components/ChartCard/ChartCard'
import MetricToggle, { type ActivityMetric } from '@/shared/components/MetricToggle/MetricToggle'
import TimeRangeSelector from '@/shared/components/TimeRangeSelector/TimeRangeSelector'
import LiveSessions from './components/LiveSessions'
import ActivityChart from './components/ActivityChart'
import TopContent from './components/TopContent'
import TopUsers from './components/TopUsers'
import { useDashboard } from './hooks/useDashboard'
import { useFetchWithDays } from './hooks/useFetchWithDays'
import { formatWatchTime } from '@/shared/utils/formatWatchTime'
import api from '@/lib/axios'
import { useChartColors } from '@/lib/chartColors'

type HourStat = { hour: number; plays: number; duration: number }
type DayStat = { day: number; plays: number; duration: number }
type PlaybackMethod = { method: string; count: number; duration: number }
type ClientStat = { client: string; count: number; duration: number }
type TopItem = { Id: string; Name: string; PlayCount: number; Type: string }
type TopUser = { UserId: string; UserName: string; TotalPlays: number; TotalWatchTime: number }
type LibraryViewCount = { Name: string; Count: number }
type LibraryDayPoint = { date: string; libraryId: string; libraryName: string; count: number }

export default function DashboardPage() {
  const CHART_COLORS = useChartColors()
  const CHART_BAR = CHART_COLORS[0]
  const { t } = useTranslation()
  const [hourMetric, setHourMetric] = useState<ActivityMetric>('count')
  const [dayMetric, setDayMetric] = useState<ActivityMetric>('count')

  // ── Plays by hour ─────────────────────────────────────────────────────────
  const { data: hourlyStats, loading: hourLoading, days: hourDays, setDays: setHourDays } =
    useFetchWithDays<HourStat>((d) => api.get(`/stats/getPopularHourOfDay?days=${d}`).then((r) => r.data))

  // ── Plays by day of week ──────────────────────────────────────────────────
  const { data: weeklyStats, loading: dayLoading, days: dayDays, setDays: setDayDays } =
    useFetchWithDays<DayStat>((d) => api.get(`/stats/getPopularDayOfWeek?days=${d}`).then((r) => r.data))

  // ── Playback methods ──────────────────────────────────────────────────────
  const { data: playbackMethods, loading: methodLoading, days: methodDays, setDays: setMethodDays } =
    useFetchWithDays<PlaybackMethod>((d) => api.get(`/stats/getMostUsedPlaybackMethod?days=${d}`).then((r) => r.data))

  // ── Top clients ───────────────────────────────────────────────────────────
  const { data: topClients, loading: clientLoading, days: clientDays, setDays: setClientDays } =
    useFetchWithDays<ClientStat>((d) => api.get(`/stats/getMostUsedClients?days=${d}`).then((r) => r.data))

  // ── Top items ─────────────────────────────────────────────────────────────
  const { data: topItems, loading: topItemsLoading, days: topItemsDays, setDays: setTopItemsDays } =
    useFetchWithDays<TopItem>((d) => api.get(`/stats/getMostPlayedItems?limit=20&days=${d}`).then((r) => r.data))

  // ── Top users ─────────────────────────────────────────────────────────────
  const { data: topUsers, loading: topUsersLoading, days: topUsersDays, setDays: setTopUsersDays } =
    useFetchWithDays<TopUser>((d) => api.get(`/stats/getMostActiveUsers?limit=5&days=${d}`).then((r) => r.data))

  // ── Most viewed libraries ─────────────────────────────────────────────────
  const { data: mostViewedLibraries, loading: libsLoading, days: libsDays, setDays: setLibsDays } =
    useFetchWithDays<LibraryViewCount>((d) => api.post('/stats/getMostViewedLibraries', { days: d }).then((r) => r.data))

  // ── All playbacks scatter (one point per play) ───────────────────────────
  type PlaybackScatterPoint = { ts: string; duration: number; name: string; type: string }
  const { data: scatterData, loading: scatterLoading, days: scatterDays, setDays: setScatterDays } =
    useFetchWithDays<PlaybackScatterPoint>((d) => api.get(`/stats/getPlaybacksScatter?days=${d}`).then((r) => r.data))

  // ── Playbacks over time by library ────────────────────────────────────────
  const { data: libraryOverTime, loading: libraryOverTimeLoading, days: libraryDays, setDays: setLibraryDays } =
    useFetchWithDays<LibraryDayPoint>((d) => api.get(`/stats/getPlaybacksByLibraryOverTime?days=${d}`).then((r) => r.data))

  // ── Stable data (no per-chart time range) ────────────────────────────────
  const { globalStats, sessions, viewsByLibraryType, loading, error, refetch } = useDashboard()

  // ── Derived values ────────────────────────────────────────────────────────
  const mediaTypeLabels: Record<string, string> = {
    Movie: t('stats.mediaType.movie'),
    Series: t('stats.mediaType.series'),
    Audio: t('stats.mediaType.audio'),
    Other: t('stats.mediaType.other'),
  }

  const pieData = viewsByLibraryType
    ? Object.entries(viewsByLibraryType)
        .filter(([, value]) => value > 0)
        .map(([key, value], i) => ({
          id: key,
          value,
          label: mediaTypeLabels[key] ?? key,
          color: CHART_COLORS[i % CHART_COLORS.length],
        }))
    : []

  const libBarLabels = mostViewedLibraries.map((l) => l.Name)
  const libBarValues = mostViewedLibraries.map((l) => l.Count)

  const hourLabels = Array.from({ length: 24 }, (_, i) => `${i}h`)
  const hourValues = Array.from({ length: 24 }, (_, i) => {
    const found = hourlyStats.find((s) => s.hour === i)
    return hourMetric === 'duration' ? (found?.duration ?? 0) : (found?.plays ?? 0)
  })

  const dayShortKeys = ['sun', 'mon', 'tue', 'wed', 'thu', 'fri', 'sat'] as const
  const dayLabels = dayShortKeys.map((k) => t(`days.short.${k}`))
  const dayValues = Array.from({ length: 7 }, (_, i) => {
    const found = weeklyStats.find((s) => s.day === i)
    return dayMetric === 'duration' ? (found?.duration ?? 0) : (found?.plays ?? 0)
  })

  const methodData = playbackMethods.map((m, i) => ({
    id: m.method,
    value: m.count,
    label: m.method,
    color: CHART_COLORS[i % CHART_COLORS.length],
  }))

  const clientNames = topClients.map((c) => c.client)
  const clientValues = topClients.map((c) => c.count)

  const hourFormatter = (v: number | null) =>
    hourMetric === 'duration' ? formatWatchTime(v ?? 0) : String(v ?? 0)
  const dayFormatter = (v: number | null) =>
    dayMetric === 'duration' ? formatWatchTime(v ?? 0) : String(v ?? 0)

  const TYPE_LABELS: Record<string, string> = {
    Movie: t('stats.mediaType.movie'),
    Episode: t('stats.mediaType.series'),
    Series: t('stats.mediaType.series'),
    Audio: t('stats.mediaType.audio'),
  }

  const scatterSeries = useMemo(() => {
    const grouped = new Map<string, PlaybackScatterPoint[]>()
    for (const p of scatterData) {
      const key = TYPE_LABELS[p.type] ? p.type : 'Other'
      if (!grouped.has(key)) grouped.set(key, [])
      grouped.get(key)!.push(p)
    }
    return Array.from(grouped.entries()).map(([type, points], i) => ({
      id: type,
      label: TYPE_LABELS[type] ?? t('stats.mediaType.other'),
      data: points.map((p, j) => ({
        x: new Date(p.ts).getTime(),
        y: p.duration,
        id: `${type}-${j}`,
      })),
      color: CHART_COLORS[i % CHART_COLORS.length],
      valueFormatter: (v: { x: number; y: number }) =>
        `(${new Date(v.x).toLocaleDateString()}, ${v.y})`,
    }))
  }, [scatterData, CHART_COLORS, t])

  const librarySeries = useMemo(() => {
    const map = new Map<string, { name: string; points: { x: number; y: number; id: string }[] }>()
    for (const row of libraryOverTime) {
      if (!map.has(row.libraryId)) map.set(row.libraryId, { name: row.libraryName, points: [] })
      map.get(row.libraryId)!.points.push({
        x: new Date(row.date).getTime(),
        y: row.count,
        id: `${row.libraryId}-${row.date}`,
      })
    }
    return Array.from(map.entries()).map(([id, { name, points }], i) => ({
      id,
      label: name,
      data: points,
      color: CHART_COLORS[i % CHART_COLORS.length],
      valueFormatter: (v: { x: number; y: number }) =>
        `(${new Date(v.x).toLocaleDateString()}, ${v.y})`,
    }))
  }, [libraryOverTime, CHART_COLORS])

  const libMargin = useMemo(() => libBarLabels.length ? Math.max(...libBarLabels.map(l => l.length)) * 7 + 16 : 80, [libBarLabels])
  const clientMargin = useMemo(() => clientNames.length ? Math.max(...clientNames.map(l => l.length)) * 7 + 16 : 80, [clientNames])

  const chartAction = (selector: ReactNode, toggle?: ReactNode) => (
    <Box sx={{ display: 'flex', gap: 1, flexDirection: { xs: 'column', sm: 'row' }, alignItems: { xs: 'flex-end', sm: 'center' } }}>
      {selector}
      {toggle}
    </Box>
  )

  return (
    <>
      <PageHeader title={t('nav.dashboard')} onRefresh={() => refetch()} loading={loading} />
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      {/* Stat cards */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid size={{ xs: 6, sm: 4, lg: 2 }}>
          <StatCard label={t('stats.totalPlays')} value={globalStats?.TotalPlays ?? '—'} icon={<Play24Regular />} loading={loading} />
        </Grid>
        <Grid size={{ xs: 6, sm: 4, lg: 2 }}>
          <StatCard label={t('stats.watchTime')} value={globalStats ? formatWatchTime(globalStats.TotalWatchTime) : '—'} icon={<Clock24Regular />} loading={loading} />
        </Grid>
        <Grid size={{ xs: 6, sm: 4, lg: 2 }}>
          <StatCard label={t('stats.activeUsers')} value={globalStats?.ActiveUsers ?? '—'} icon={<People24Regular />} loading={loading} />
        </Grid>
        <Grid size={{ xs: 6, sm: 4, lg: 2 }}>
          <StatCard label={t('stats.totalUsers')} value={globalStats?.TotalUsers ?? '—'} icon={<VideoClip24Regular />} loading={loading} />
        </Grid>
        <Grid size={{ xs: 6, sm: 4, lg: 2 }}>
          <StatCard label={t('stats.libraries')} value={globalStats?.TotalLibraries ?? '—'} icon={<Library24Regular />} loading={loading} />
        </Grid>
        <Grid size={{ xs: 6, sm: 4, lg: 2 }}>
          <StatCard label={t('stats.totalItems')} value={globalStats?.TotalItems ?? '—'} icon={<Apps24Regular />} loading={loading} />
        </Grid>
      </Grid>

      {/* Activity over time + Live sessions */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid size={{ xs: 12, md: 8 }}>
          <ActivityChart />
        </Grid>
        <Grid size={{ xs: 12, md: 4 }}>
          <LiveSessions initialSessions={sessions} loading={loading} />
        </Grid>
      </Grid>

      {/* Plays by hour + Plays by day of week */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid size={{ xs: 12, md: 6 }}>
          <ChartCard
            title={t('stats.playsByHour')}
            loading={hourLoading}
            empty={!hourLoading && hourlyStats.length === 0}
            height={220}
            action={chartAction(
              <TimeRangeSelector value={hourDays} onChange={setHourDays} />,
              <MetricToggle value={hourMetric} onChange={setHourMetric} />,
            )}
          >
            <BarChart
              xAxis={[{ data: hourLabels, scaleType: 'band' }]}
              series={[{ data: hourValues, color: CHART_BAR, valueFormatter: hourFormatter }]}
              height={220}
              sx={{ width: '100%' }}
              grid={{ horizontal: true }}
              slotProps={{ legend: { hidden: true } as any }}
            />
          </ChartCard>
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <ChartCard
            title={t('stats.playsByDayOfWeek')}
            loading={dayLoading}
            empty={!dayLoading && weeklyStats.length === 0}
            height={220}
            action={chartAction(
              <TimeRangeSelector value={dayDays} onChange={setDayDays} />,
              <MetricToggle value={dayMetric} onChange={setDayMetric} />,
            )}
          >
            <BarChart
              xAxis={[{ data: dayLabels, scaleType: 'band' }]}
              series={[{ data: dayValues, color: CHART_BAR, valueFormatter: dayFormatter }]}
              height={220}
              sx={{ width: '100%' }}
              grid={{ horizontal: true }}
              slotProps={{ legend: { hidden: true } as any }}
            />
          </ChartCard>
        </Grid>
      </Grid>

      {/* Top content + Top users */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid size={{ xs: 12, md: 6 }}>
          <TopContent
            items={topItems}
            loading={topItemsLoading}
            timeRangeSelector={<TimeRangeSelector value={topItemsDays} onChange={setTopItemsDays} />}
          />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <TopUsers
            users={topUsers}
            loading={topUsersLoading}
            action={<TimeRangeSelector value={topUsersDays} onChange={setTopUsersDays} />}
          />
        </Grid>
      </Grid>

      {/* Media type | Most viewed libraries | Playback methods */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid size={{ xs: 12, md: 4 }}>
          <ChartCard
            title={t('stats.viewsByMediaType')}
            loading={loading}
            empty={pieData.length === 0}
            height={260}
          >
            <PieChart
              series={[{ data: pieData, innerRadius: 40, paddingAngle: 2, cornerRadius: 3 }]}
              height={260}
              sx={{ width: '100%' }}
              slotProps={{ legend: { position: { vertical: 'middle', horizontal: 'end' } } }}
            />
          </ChartCard>
        </Grid>
        <Grid size={{ xs: 12, md: 4 }}>
          <ChartCard
            title={t('stats.mostViewedLibraries')}
            loading={libsLoading}
            empty={!libsLoading && mostViewedLibraries.length === 0}
            height={260}
            action={<TimeRangeSelector value={libsDays} onChange={setLibsDays} />}
          >
            <BarChart
              layout="horizontal"
              yAxis={[{ data: libBarLabels, scaleType: 'band', width: libMargin }]}
              series={[{ data: libBarValues, color: CHART_BAR, valueFormatter: (v) => String(v ?? 0) }]}
              height={260}
              sx={{ width: '100%' }}
              grid={{ vertical: true }}
              slotProps={{ legend: { hidden: true } as any }}
              margin={{ left: 4, right: 4, top: 8, bottom: 8 }}
            />
          </ChartCard>
        </Grid>
        <Grid size={{ xs: 12, md: 4 }}>
          <ChartCard
            title={t('stats.playbackMethod')}
            loading={methodLoading}
            empty={!methodLoading && methodData.length === 0}
            height={260}
            action={<TimeRangeSelector value={methodDays} onChange={setMethodDays} />}
          >
            <BarChart
              xAxis={[{ data: playbackMethods.map(m => m.method), scaleType: 'band' }]}
              series={[{ data: playbackMethods.map(m => m.count), color: CHART_BAR, valueFormatter: (v) => `${v ?? 0} ${t('common.plays')}` }]}
              height={260}
              sx={{ width: '100%' }}
              grid={{ horizontal: true }}
              slotProps={{ legend: { hidden: true } as any }}
            />
          </ChartCard>
        </Grid>
      </Grid>

      {/* Top clients */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid size={{ xs: 12 }}>
          <ChartCard
            title={t('stats.topClients')}
            loading={clientLoading}
            empty={!clientLoading && topClients.length === 0}
            height={260}
            action={<TimeRangeSelector value={clientDays} onChange={setClientDays} />}
          >
            <BarChart
              layout="horizontal"
              yAxis={[{ data: clientNames, scaleType: 'band', width: clientMargin }]}
              series={[{ data: clientValues, color: CHART_BAR, valueFormatter: (v) => String(v ?? 0) }]}
              height={260}
              sx={{ width: '100%' }}
              grid={{ vertical: true }}
              slotProps={{ legend: { hidden: true } as any }}
              margin={{ left: 4, right: 4, top: 8, bottom: 8 }}
            />
          </ChartCard>
        </Grid>
      </Grid>

      {/* All playbacks scatter */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid size={{ xs: 12 }}>
          <ChartCard
            title={t('stats.allPlaybacks')}
            loading={scatterLoading}
            empty={!scatterLoading && scatterSeries.length === 0}
            height={300}
            action={<TimeRangeSelector value={scatterDays} onChange={setScatterDays} />}
          >
            <ScatterChart
              series={scatterSeries}
              xAxis={[{ scaleType: 'time', valueFormatter: (v) => new Date(v).toLocaleDateString() }]}
              yAxis={[{ label: 'min' }]}
              height={300}
              sx={{ width: '100%' }}
              slotProps={{ legend: { position: { vertical: 'top', horizontal: 'end' } } }}
              margin={{ left: 50, right: 20, top: 40, bottom: 40 }}
            />
          </ChartCard>
        </Grid>
      </Grid>

      {/* Playbacks over time by library */}
      <Grid container spacing={2}>
        <Grid size={{ xs: 12 }}>
          <ChartCard
            title={t('stats.playbacksByLibrary', 'Playbacks by library')}
            loading={libraryOverTimeLoading}
            empty={!libraryOverTimeLoading && librarySeries.length === 0}
            height={320}
            action={<TimeRangeSelector value={libraryDays} onChange={setLibraryDays} />}
          >
            <ScatterChart
              series={librarySeries}
              xAxis={[{ scaleType: 'time', valueFormatter: (v) => new Date(v).toLocaleDateString() }]}
              yAxis={[{ label: t('common.plays', 'Plays') }]}
              height={320}
              sx={{ width: '100%' }}
              slotProps={{ legend: { position: { vertical: 'top', horizontal: 'end' } } }}
              margin={{ left: 50, right: 20, top: 40, bottom: 40 }}
            />
          </ChartCard>
        </Grid>
      </Grid>
    </>
  )
}
