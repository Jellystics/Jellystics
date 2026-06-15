import { Grid, Alert } from '@mui/material'
import {
  Play24Regular, Clock24Regular, People24Regular,
  VideoClip24Regular, Library24Regular, Apps24Regular,
} from '@fluentui/react-icons'
import { useTranslation } from 'react-i18next'
import { useState } from 'react'

import { PieChart } from '@mui/x-charts/PieChart'
import { BarChart } from '@mui/x-charts/BarChart'
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

// Chart colors — independent of the UI primary color
const CHART_COLORS = ['#60a5fa', '#34d399', '#fb923c', '#f472b6', '#a78bfa', '#facc15', '#38bdf8', '#4ade80']
const CHART_BAR = '#60a5fa'

export default function DashboardPage() {
  const { t } = useTranslation()
  const [hourMetric, setHourMetric] = useState<ActivityMetric>('count')
  const [dayMetric, setDayMetric] = useState<ActivityMetric>('count')

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
      <Grid container spacing={2}>
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
    </>
  )
}
