import { useState, useEffect } from 'react'
import { Grid, Alert } from '@mui/material'

import { BarChart } from '@mui/x-charts/BarChart'
import { PieChart } from '@mui/x-charts/PieChart'
import { useTranslation } from 'react-i18next'
import PageHeader from '@/shared/components/PageHeader/PageHeader'
import ChartCard from '@/shared/components/ChartCard/ChartCard'
import StatCard from '@/shared/components/StatCard/StatCard'
import api from '@/lib/axios'
import type { WatchStatOverTime, HourStat, DayStat, PlayMethodStat, ClientStat } from '@/shared/types/stats'
import { Play24Regular, Clock24Regular } from '@fluentui/react-icons'
import { formatWatchTime } from '@/shared/utils/formatWatchTime'
import MetricToggle, { type ActivityMetric } from '@/shared/components/MetricToggle/MetricToggle'

type ActiveUser = { UserId: string; UserName: string; TotalPlays: number; TotalWatchTime: number }
type PlayedItem = { Id: string; Name: string; PlayCount: number; Type: string }

// Chart colors — independent of the UI primary color
const CHART_COLORS = ['#60a5fa', '#34d399', '#fb923c', '#f472b6', '#a78bfa', '#facc15', '#38bdf8', '#4ade80']

export default function StatisticsPage() {
  const { t } = useTranslation()
  const [overTime, setOverTime] = useState<WatchStatOverTime[]>([])
  const [byHour, setByHour] = useState<HourStat[]>([])
  const [byDay, setByDay] = useState<DayStat[]>([])
  const [byMethod, setByMethod] = useState<PlayMethodStat[]>([])
  const [byClient, setByClient] = useState<ClientStat[]>([])
  const [overTimeMetric, setOverTimeMetric] = useState<ActivityMetric>('count')
  const [hourMetric, setHourMetric] = useState<ActivityMetric>('count')
  const [dayMetric, setDayMetric] = useState<ActivityMetric>('count')
  const [methodMetric, setMethodMetric] = useState<ActivityMetric>('count')
  const [clientMetric, setClientMetric] = useState<ActivityMetric>('count')
  const [activeUsers, setActiveUsers] = useState<ActiveUser[]>([])
  const [topItems, setTopItems] = useState<PlayedItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const load = (showLoading = true) => {
      if (showLoading) setLoading(true)
      Promise.all([
        api.get('/stats/getWatchStatisticsOverTime?days=30'),
        api.get('/stats/getPopularHourOfDay'),
        api.get('/stats/getPopularDayOfWeek'),
        api.get('/stats/getMostUsedPlaybackMethod'),
        api.get('/stats/getMostUsedClients'),
        api.get('/stats/getMostActiveUsers?limit=8'),
        api.get('/stats/getMostPlayedItems?limit=10'),
      ])
        .then(([otRes, hourRes, dayRes, methodRes, clientRes, usersRes, itemsRes]) => {
          setOverTime(otRes.data ?? [])
          setByHour(hourRes.data ?? [])
          setByDay(dayRes.data ?? [])
          setByMethod(methodRes.data ?? [])
          setByClient(clientRes.data ?? [])
          setActiveUsers((usersRes.data ?? []).slice(0, 8))
          setTopItems((itemsRes.data ?? []).slice(0, 10))
        })
        .catch(() => setError(t('common.loadError')))
        .finally(() => setLoading(false))
    }

    load()
    const interval = window.setInterval(() => load(false), 15000)
    return () => window.clearInterval(interval)
  }, [t])

  const totalPlays = overTime.reduce((s, d) => s + d.plays, 0)
  const totalDuration = overTime.reduce((s, d) => s + d.duration, 0)

  const hourData = Array.from({ length: 24 }, (_, h) => ({
    hour: `${String(h).padStart(2, '0')}${t('time.hourShort')}`,
    plays: byHour.find((d) => d.hour === h)?.plays ?? 0,
    duration: byHour.find((d) => d.hour === h)?.duration ?? 0,
  }))

  const dayData = [
    t('days.short.sun'), t('days.short.mon'), t('days.short.tue'), t('days.short.wed'),
    t('days.short.thu'), t('days.short.fri'), t('days.short.sat'),
  ].map((day, i) => ({
    day,
    plays: byDay.find((d) => d.day === String(i))?.plays ?? 0,
    duration: byDay.find((d) => d.day === String(i))?.duration ?? 0,
  }))

  const chartMetric = (metric: ActivityMetric) => ({
    label: metric === 'duration' ? t('stats.watchTime') : t('common.plays'),
    formatter: (value: number | null) => metric === 'duration' ? formatWatchTime(value ?? 0) : String(value ?? 0),
    getValue: (d: { plays: number; duration: number }) => metric === 'duration' ? d.duration : d.plays,
    getPieValue: (d: { count: number; duration: number }) => metric === 'duration' ? d.duration : d.count,
  })

  const otChart = chartMetric(overTimeMetric)
  const hourChart = chartMetric(hourMetric)
  const dayChart = chartMetric(dayMetric)
  const methodChart = chartMetric(methodMetric)
  const clientChart = chartMetric(clientMetric)

  return (
    <>
      <PageHeader title={t('nav.statistics')} />
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard label={t('stats.totalPlays30d')} value={totalPlays} icon={<Play24Regular />} loading={loading} />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard label={t('stats.watchTime30d')} value={formatWatchTime(totalDuration)} icon={<Clock24Regular />} loading={loading} />
        </Grid>
      </Grid>

      <Grid container spacing={2} sx={{ mb: 2 }}>
        <Grid size={{ xs: 12 }}>
          <ChartCard
            title={t('stats.playsOverTime30d')}
            loading={loading}
            empty={overTime.length === 0}
            height={240}
            action={<MetricToggle value={overTimeMetric} onChange={setOverTimeMetric} />}
          >
            <BarChart
              xAxis={[{ data: overTime.map((d) => d.date), scaleType: 'band' }]}
              series={[{ data: overTime.map((d) => otChart.getValue(d)), label: otChart.label, valueFormatter: otChart.formatter, color: CHART_COLORS[0] }]}
              height={240}
              sx={{ width: '100%' }}
              grid={{ horizontal: true }}
              slotProps={{ legend: { hidden: true } }}
            />
          </ChartCard>
        </Grid>
      </Grid>

      <Grid container spacing={2} sx={{ mb: 2 }}>
        <Grid size={{ xs: 12, md: 6 }}>
          <ChartCard
            title={t('stats.playsByHour')}
            loading={loading}
            empty={byHour.length === 0}
            height={220}
            action={<MetricToggle value={hourMetric} onChange={setHourMetric} />}
          >
            <BarChart
              xAxis={[{ data: hourData.map((d) => d.hour), scaleType: 'band' }]}
              series={[{ data: hourData.map((d) => hourChart.getValue(d)), label: hourChart.label, valueFormatter: hourChart.formatter, color: CHART_COLORS[0] }]}
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
            empty={byDay.length === 0}
            height={220}
            action={<MetricToggle value={dayMetric} onChange={setDayMetric} />}
          >
            <BarChart
              xAxis={[{ data: dayData.map((d) => d.day), scaleType: 'band' }]}
              series={[{ data: dayData.map((d) => dayChart.getValue(d)), label: dayChart.label, valueFormatter: dayChart.formatter, color: CHART_COLORS[1] }]}
              height={220}
              sx={{ width: '100%' }}
              grid={{ horizontal: true }}
              slotProps={{ legend: { hidden: true } }}
            />
          </ChartCard>
        </Grid>
      </Grid>

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, md: 6 }}>
          <ChartCard
            title={t('stats.playbackMethod')}
            loading={loading}
            empty={byMethod.length === 0}
            height={280}
            action={<MetricToggle value={methodMetric} onChange={setMethodMetric} />}
          >
            <PieChart
              series={[{
                data: byMethod.map((item, i) => ({ id: i, value: methodChart.getPieValue(item), label: item.method, color: CHART_COLORS[i % CHART_COLORS.length] })),
                outerRadius: 100,
                paddingAngle: 2,
                cornerRadius: 3,
                valueFormatter: (item) => methodChart.formatter(item.value),
              }]}
              height={280}
              sx={{ width: '100%' }}
            />
          </ChartCard>
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <ChartCard
            title={t('stats.topClients')}
            loading={loading}
            empty={byClient.length === 0}
            height={280}
            action={<MetricToggle value={clientMetric} onChange={setClientMetric} />}
          >
            <PieChart
              series={[{
                data: byClient.map((item, i) => ({ id: i, value: clientChart.getPieValue(item), label: item.client, color: CHART_COLORS[i % CHART_COLORS.length] })),
                outerRadius: 100,
                paddingAngle: 2,
                cornerRadius: 3,
                valueFormatter: (item) => clientChart.formatter(item.value),
              }]}
              height={280}
              sx={{ width: '100%' }}
            />
          </ChartCard>
        </Grid>
      </Grid>

      <Grid container spacing={2} sx={{ mt: 2 }}>
        <Grid size={{ xs: 12 }}>
          <ChartCard
            title={t('stats.mostActiveUsers', 'Utilisateurs les plus actifs')}
            loading={loading}
            empty={activeUsers.length === 0}
            height={220}
          >
            <BarChart
              xAxis={[{ data: activeUsers.map((d) => d.UserName), scaleType: 'band' }]}
              series={[{
                data: activeUsers.map((d) => d.TotalPlays),
                label: t('common.plays'),
                valueFormatter: (v) => String(v ?? 0),
                color: CHART_COLORS[0],
              }]}
              height={220}
              sx={{ width: '100%' }}
              grid={{ horizontal: true }}
              slotProps={{ legend: { hidden: true } }}
            />
          </ChartCard>
        </Grid>
      </Grid>

      <Grid container spacing={2} sx={{ mt: 2, mb: 2 }}>
        <Grid size={{ xs: 12 }}>
          <ChartCard
            title={t('stats.topItems', 'Contenus les plus vus')}
            loading={loading}
            empty={topItems.length === 0}
            height={280}
          >
            <BarChart
              layout="horizontal"
              yAxis={[{ data: topItems.map((d) => d.Name), scaleType: 'band' }]}
              xAxis={[{ label: t('common.plays') }]}
              series={[{
                data: topItems.map((d) => d.PlayCount),
                label: t('common.plays'),
                valueFormatter: (v) => String(v ?? 0),
                color: CHART_COLORS[2],
              }]}
              height={280}
              sx={{ width: '100%' }}
              grid={{ vertical: true }}
              slotProps={{ legend: { hidden: true } }}
            />
          </ChartCard>
        </Grid>
      </Grid>
    </>
  )
}
