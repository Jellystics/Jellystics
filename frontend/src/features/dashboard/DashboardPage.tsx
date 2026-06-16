import { Grid, Alert, ToggleButtonGroup, ToggleButton } from '@mui/material'
import {
  Play24Regular, Clock24Regular, People24Regular,
  VideoClip24Regular, Library24Regular, Apps24Regular,
} from '@fluentui/react-icons'
import { useTranslation } from 'react-i18next'
import { useState, useEffect, useMemo } from 'react'

import { PieChart } from '@mui/x-charts/PieChart'
import { BarChart } from '@mui/x-charts/BarChart'
import { ScatterChart } from '@mui/x-charts/ScatterChart'
import PageHeader from '@/shared/components/PageHeader/PageHeader'
import StatCard from '@/shared/components/StatCard/StatCard'
import ChartCard from '@/shared/components/ChartCard/ChartCard'
import MetricToggle, { type ActivityMetric } from '@/shared/components/MetricToggle/MetricToggle'
import LiveSessions from './components/LiveSessions'
import ActivityChart from './components/ActivityChart'
import TopContent from './components/TopContent'
import TopUsers from './components/TopUsers'
import { useDashboard } from './hooks/useDashboard'
import { formatWatchTime } from '@/shared/utils/formatWatchTime'
import api from '@/lib/axios'
import { useChartColors } from '@/lib/chartColors'

export default function DashboardPage() {
  const CHART_COLORS = useChartColors()
  const CHART_BAR = CHART_COLORS[0]
  const { t } = useTranslation()
  const [hourMetric, setHourMetric] = useState<ActivityMetric>('count')
  const [dayMetric, setDayMetric] = useState<ActivityMetric>('count')

  interface LibraryDayPoint { date: string; libraryId: string; libraryName: string; count: number }
  const [libraryOverTime, setLibraryOverTime] = useState<LibraryDayPoint[]>([])
  const [libraryOverTimeLoading, setLibraryOverTimeLoading] = useState(true)
  const [libraryDays, setLibraryDays] = useState<7 | 30 | 0>(0)

  useEffect(() => {
    setLibraryOverTimeLoading(true)
    api.get('/stats/getPlaybacksByLibraryOverTime', { params: { days: 0 } })
      .then((r) => setLibraryOverTime(r.data ?? []))
      .catch(() => setLibraryOverTime([]))
      .finally(() => setLibraryOverTimeLoading(false))
  }, [])

  const filteredLibraryOverTime = useMemo(() => {
    if (libraryDays === 0) return libraryOverTime
    const cutoff = new Date()
    cutoff.setDate(cutoff.getDate() - libraryDays)
    const cutoffStr = cutoff.toISOString().slice(0, 10)
    return libraryOverTime.filter((r) => r.date >= cutoffStr)
  }, [libraryOverTime, libraryDays])

  const {
    globalStats, sessions, topItems, topUsers,
    viewsByLibraryType, mostViewedLibraries, hourlyStats, weeklyStats,
    playbackMethods, topClients, loading, error,
  } = useDashboard()

  const mediaTypeLabels: Record<string, string> = {
    Movie: t('stats.mediaType.movie'),
    Series: t('stats.mediaType.series'),
    Audio: t('stats.mediaType.audio'),
    Other: t('stats.mediaType.other'),
  }

  // Media type pie
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

  // Most viewed libraries bar
  const libBarLabels = mostViewedLibraries.map((l) => l.Name)
  const libBarValues = mostViewedLibraries.map((l) => l.Count)

  // Plays by hour (0-23)
  const hourLabels = Array.from({ length: 24 }, (_, i) => `${i}h`)
  const hourValues = Array.from({ length: 24 }, (_, i) => {
    const found = hourlyStats.find((s) => s.hour === i)
    return hourMetric === 'duration' ? (found?.duration ?? 0) : (found?.plays ?? 0)
  })

  // Plays by day of week (0=Sun … 6=Sat)
  const dayShortKeys = ['sun', 'mon', 'tue', 'wed', 'thu', 'fri', 'sat'] as const
  const dayLabels = dayShortKeys.map((k) => t(`days.short.${k}`))
  const dayValues = Array.from({ length: 7 }, (_, i) => {
    const found = weeklyStats.find((s) => s.day === i)
    return dayMetric === 'duration' ? (found?.duration ?? 0) : (found?.plays ?? 0)
  })

  // Playback methods
  const methodData = playbackMethods.map((m, i) => ({
    id: m.method,
    value: m.count,
    label: m.method,
    color: CHART_COLORS[i % CHART_COLORS.length],
  }))

  // Top clients horizontal bar
  const clientNames = topClients.map((c) => c.client)
  const clientValues = topClients.map((c) => c.count)

  const hourFormatter = (v: number | null) =>
    hourMetric === 'duration' ? formatWatchTime(v ?? 0) : String(v ?? 0)

  const dayFormatter = (v: number | null) =>
    dayMetric === 'duration' ? formatWatchTime(v ?? 0) : String(v ?? 0)

  // Scatter: group by library, x = date timestamp, y = count
  const librarySeries = useMemo(() => {
    const map = new Map<string, { name: string; points: { x: number; y: number; id: string }[] }>()
    for (const row of filteredLibraryOverTime) {
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
  }, [filteredLibraryOverTime])

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

      {/* Activity over 30d + Live sessions */}
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
            loading={loading}
            empty={!loading && hourlyStats.length === 0}
            height={220}
            action={<MetricToggle value={hourMetric} onChange={setHourMetric} />}
          >
            <BarChart
              xAxis={[{ data: hourLabels, scaleType: 'band' }]}
              series={[{
                data: hourValues,
                color: CHART_BAR,
                valueFormatter: hourFormatter,
              }]}
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
            loading={loading}
            empty={!loading && weeklyStats.length === 0}
            height={220}
            action={<MetricToggle value={dayMetric} onChange={setDayMetric} />}
          >
            <BarChart
              xAxis={[{ data: dayLabels, scaleType: 'band' }]}
              series={[{
                data: dayValues,
                color: CHART_BAR,
                valueFormatter: dayFormatter,
              }]}
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
          <TopContent items={topItems} loading={loading} />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <TopUsers users={topUsers} loading={loading} />
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
              series={[{
                data: pieData,
                innerRadius: 40,
                paddingAngle: 2,
                cornerRadius: 3,
              }]}
              height={260}
              sx={{ width: '100%' }}
              slotProps={{ legend: { position: { vertical: 'middle', horizontal: 'right' } } }}
            />
          </ChartCard>
        </Grid>
        <Grid size={{ xs: 12, md: 4 }}>
          <ChartCard
            title={t('stats.mostViewedLibraries')}
            loading={loading}
            empty={mostViewedLibraries.length === 0}
            height={260}
          >
            <BarChart
              layout="horizontal"
              yAxis={[{ data: libBarLabels, scaleType: 'band' }]}
              xAxis={[{ label: t('common.plays') }]}
              series={[{
                data: libBarValues,
                color: CHART_BAR,
                valueFormatter: (v) => String(v ?? 0),
              }]}
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
            loading={loading}
            empty={methodData.length === 0}
            height={260}
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
            loading={loading}
            empty={topClients.length === 0}
            height={260}
          >
            <BarChart
              layout="horizontal"
              yAxis={[{ data: clientNames, scaleType: 'band' }]}
              xAxis={[{ label: t('common.plays') }]}
              series={[{
                data: clientValues,
                color: CHART_BAR,
                valueFormatter: (v) => String(v ?? 0),
              }]}
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
            action={
              <ToggleButtonGroup
                value={libraryDays}
                exclusive
                onChange={(_, v) => { if (v !== null) setLibraryDays(v) }}
                size="small"
                sx={{ '& .MuiToggleButton-root': { px: 1.5, py: 0.25, fontSize: 12, textTransform: 'none', borderRadius: '90px !important', border: 'none', '&.Mui-selected': { fontWeight: 600 } } }}
              >
                <ToggleButton value={7}>7d</ToggleButton>
                <ToggleButton value={30}>30d</ToggleButton>
                <ToggleButton value={0}>{t('common.all', 'All')}</ToggleButton>
              </ToggleButtonGroup>
            }
          >
            <ScatterChart
              series={librarySeries}
              xAxis={[{
                scaleType: 'time',
                valueFormatter: (v) => new Date(v).toLocaleDateString(),
              }]}
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
