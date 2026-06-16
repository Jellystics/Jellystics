import { useState, useEffect, useMemo } from 'react'
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
  const [hourDays, setHourDays] = useState(30)
  const [hourlyStats, setHourlyStats] = useState<HourStat[]>([])
  const [hourLoading, setHourLoading] = useState(true)
  useEffect(() => {
    setHourLoading(true)
    api.get(`/stats/getPopularHourOfDay?days=${hourDays}`)
      .then(r => setHourlyStats(r.data ?? []))
      .catch(() => setHourlyStats([]))
      .finally(() => setHourLoading(false))
  }, [hourDays])

  // ── Plays by day of week ──────────────────────────────────────────────────
  const [dayDays, setDayDays] = useState(30)
  const [weeklyStats, setWeeklyStats] = useState<DayStat[]>([])
  const [dayLoading, setDayLoading] = useState(true)
  useEffect(() => {
    setDayLoading(true)
    api.get(`/stats/getPopularDayOfWeek?days=${dayDays}`)
      .then(r => setWeeklyStats(r.data ?? []))
      .catch(() => setWeeklyStats([]))
      .finally(() => setDayLoading(false))
  }, [dayDays])

  // ── Playback methods ──────────────────────────────────────────────────────
  const [methodDays, setMethodDays] = useState(30)
  const [playbackMethods, setPlaybackMethods] = useState<PlaybackMethod[]>([])
  const [methodLoading, setMethodLoading] = useState(true)
  useEffect(() => {
    setMethodLoading(true)
    api.get(`/stats/getMostUsedPlaybackMethod?days=${methodDays}`)
      .then(r => setPlaybackMethods(r.data ?? []))
      .catch(() => setPlaybackMethods([]))
      .finally(() => setMethodLoading(false))
  }, [methodDays])

  // ── Top clients ───────────────────────────────────────────────────────────
  const [clientDays, setClientDays] = useState(30)
  const [topClients, setTopClients] = useState<ClientStat[]>([])
  const [clientLoading, setClientLoading] = useState(true)
  useEffect(() => {
    setClientLoading(true)
    api.get(`/stats/getMostUsedClients?days=${clientDays}`)
      .then(r => setTopClients(r.data ?? []))
      .catch(() => setTopClients([]))
      .finally(() => setClientLoading(false))
  }, [clientDays])

  // ── Top items ─────────────────────────────────────────────────────────────
  const [topItemsDays, setTopItemsDays] = useState(30)
  const [topItems, setTopItems] = useState<TopItem[]>([])
  const [topItemsLoading, setTopItemsLoading] = useState(true)
  useEffect(() => {
    setTopItemsLoading(true)
    api.get(`/stats/getMostPlayedItems?limit=20&days=${topItemsDays}`)
      .then(r => setTopItems(r.data ?? []))
      .catch(() => setTopItems([]))
      .finally(() => setTopItemsLoading(false))
  }, [topItemsDays])

  // ── Top users ─────────────────────────────────────────────────────────────
  const [topUsersDays, setTopUsersDays] = useState(30)
  const [topUsers, setTopUsers] = useState<TopUser[]>([])
  const [topUsersLoading, setTopUsersLoading] = useState(true)
  useEffect(() => {
    setTopUsersLoading(true)
    api.get(`/stats/getMostActiveUsers?limit=5&days=${topUsersDays}`)
      .then(r => setTopUsers(r.data ?? []))
      .catch(() => setTopUsers([]))
      .finally(() => setTopUsersLoading(false))
  }, [topUsersDays])

  // ── Most viewed libraries ─────────────────────────────────────────────────
  const [libsDays, setLibsDays] = useState(30)
  const [mostViewedLibraries, setMostViewedLibraries] = useState<LibraryViewCount[]>([])
  const [libsLoading, setLibsLoading] = useState(true)
  useEffect(() => {
    setLibsLoading(true)
    api.post('/stats/getMostViewedLibraries', { days: libsDays })
      .then(r => setMostViewedLibraries(r.data ?? []))
      .catch(() => setMostViewedLibraries([]))
      .finally(() => setLibsLoading(false))
  }, [libsDays])

  // ── Playbacks over time by library ────────────────────────────────────────
  const [libraryDays, setLibraryDays] = useState(30)
  const [libraryOverTime, setLibraryOverTime] = useState<LibraryDayPoint[]>([])
  const [libraryOverTimeLoading, setLibraryOverTimeLoading] = useState(true)
  useEffect(() => {
    setLibraryOverTimeLoading(true)
    api.get(`/stats/getPlaybacksByLibraryOverTime?days=${libraryDays}`)
      .then(r => setLibraryOverTime(r.data ?? []))
      .catch(() => setLibraryOverTime([]))
      .finally(() => setLibraryOverTimeLoading(false))
  }, [libraryDays])

  // ── Stable data (no per-chart time range) ────────────────────────────────
  const { globalStats, sessions, viewsByLibraryType, loading, error } = useDashboard()

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
    }))
  }, [libraryOverTime, CHART_COLORS])

  const chartAction = (selector: ReactNode, toggle?: ReactNode) => (
    <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
      {selector}
      {toggle}
    </Box>
  )

  return (
    <>
      <PageHeader title={t('nav.dashboard')} />
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
              slotProps={{ legend: { hidden: true } }}
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
              slotProps={{ legend: { hidden: true } }}
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
              slotProps={{ legend: { position: { vertical: 'middle', horizontal: 'right' } } }}
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
              yAxis={[{ data: libBarLabels, scaleType: 'band' }]}
              xAxis={[{ label: t('common.plays') }]}
              series={[{ data: libBarValues, color: CHART_BAR, valueFormatter: (v) => String(v ?? 0) }]}
              height={260}
              sx={{ width: '100%' }}
              grid={{ vertical: true }}
              slotProps={{ legend: { hidden: true } }}
              margin={{ left: 110, right: 16, top: 8, bottom: 36 }}
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
            <PieChart
              series={[{
                data: methodData,
                paddingAngle: 2,
                cornerRadius: 3,
                valueFormatter: (item) => `${item.value} ${t('common.plays')}`,
              }]}
              height={260}
              sx={{ width: '100%' }}
              slotProps={{ legend: { position: { vertical: 'middle', horizontal: 'right' } } }}
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
              yAxis={[{ data: clientNames, scaleType: 'band' }]}
              xAxis={[{ label: t('common.plays') }]}
              series={[{ data: clientValues, color: CHART_BAR, valueFormatter: (v) => String(v ?? 0) }]}
              height={260}
              sx={{ width: '100%' }}
              grid={{ vertical: true }}
              slotProps={{ legend: { hidden: true } }}
              margin={{ left: 160, right: 16, top: 8, bottom: 36 }}
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
              slotProps={{ legend: { position: { vertical: 'top', horizontal: 'right' } } }}
              margin={{ left: 50, right: 20, top: 40, bottom: 40 }}
            />
          </ChartCard>
        </Grid>
      </Grid>
    </>
  )
}
