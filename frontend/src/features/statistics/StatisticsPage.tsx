import { useState, useEffect } from 'react'
import { Grid, Alert } from '@mui/material'
import {
  BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid,
  PieChart, Pie, Cell, Legend,
} from 'recharts'
import { useTheme } from '@mui/material/styles'
import { useTranslation } from 'react-i18next'
import PageHeader from '@/shared/components/PageHeader/PageHeader'
import ChartCard from '@/shared/components/ChartCard/ChartCard'
import StatCard from '@/shared/components/StatCard/StatCard'
import api from '@/lib/axios'
import type { WatchStatOverTime, HourStat, DayStat, PlayMethodStat, ClientStat } from '@/shared/types/stats'
import { Play24Regular, Clock24Regular } from '@fluentui/react-icons'
import { formatWatchTime } from '@/shared/utils/formatWatchTime'
import MetricToggle, { type ActivityMetric } from '@/shared/components/MetricToggle/MetricToggle'

const COLORS = ['#a78bfa', '#7c3aed', '#6d28d9', '#5b21b6', '#4c1d95', '#8b5cf6', '#c4b5fd', '#ede9fe']

export default function StatisticsPage() {
  const { t } = useTranslation()
  const theme = useTheme()
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
      ])
        .then(([otRes, hourRes, dayRes, methodRes, clientRes]) => {
          setOverTime(otRes.data ?? [])
          setByHour(hourRes.data ?? [])
          setByDay(dayRes.data ?? [])
          setByMethod(methodRes.data ?? [])
          setByClient(clientRes.data ?? [])
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
    barKey: metric === 'duration' ? 'duration' : 'plays',
    pieKey: metric === 'duration' ? 'duration' : 'count',
    label: metric === 'duration' ? t('stats.watchTime') : t('common.plays'),
    formatter: (value: unknown) => metric === 'duration' ? formatWatchTime(Number(value ?? 0)) : String(value ?? 0),
  })

  const overTimeChart = chartMetric(overTimeMetric)
  const hourChart = chartMetric(hourMetric)
  const dayChart = chartMetric(dayMetric)
  const methodChart = chartMetric(methodMetric)
  const clientChart = chartMetric(clientMetric)

  const tooltipStyle = {
    backgroundColor: theme.palette.background.paper,
    border: `1px solid ${theme.palette.divider}`,
    borderRadius: 8,
    fontSize: 12,
  }

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
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={overTime} margin={{ top: 4, right: 4, bottom: 0, left: -20 }}>
                <CartesianGrid strokeDasharray="3 3" stroke={theme.palette.divider} />
                <XAxis dataKey="date" tick={{ fontSize: 10, fill: theme.palette.text.secondary }} tickLine={false} axisLine={false} />
                <YAxis tick={{ fontSize: 11, fill: theme.palette.text.secondary }} tickLine={false} axisLine={false} />
                <Tooltip contentStyle={tooltipStyle} formatter={overTimeChart.formatter} />
                <Bar dataKey={overTimeChart.barKey} fill={theme.palette.primary.main} radius={[4, 4, 0, 0]} name={overTimeChart.label} />
              </BarChart>
            </ResponsiveContainer>
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
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={hourData} margin={{ top: 4, right: 4, bottom: 0, left: -20 }}>
                <CartesianGrid strokeDasharray="3 3" stroke={theme.palette.divider} />
                <XAxis dataKey="hour" tick={{ fontSize: 9, fill: theme.palette.text.secondary }} tickLine={false} axisLine={false} />
                <YAxis tick={{ fontSize: 11, fill: theme.palette.text.secondary }} tickLine={false} axisLine={false} />
                <Tooltip contentStyle={tooltipStyle} formatter={hourChart.formatter} />
                <Bar dataKey={hourChart.barKey} fill={theme.palette.primary.main} radius={[3, 3, 0, 0]} name={hourChart.label} />
              </BarChart>
            </ResponsiveContainer>
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
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={dayData} margin={{ top: 4, right: 4, bottom: 0, left: -20 }}>
                <CartesianGrid strokeDasharray="3 3" stroke={theme.palette.divider} />
                <XAxis dataKey="day" tick={{ fontSize: 11, fill: theme.palette.text.secondary }} tickLine={false} axisLine={false} />
                <YAxis tick={{ fontSize: 11, fill: theme.palette.text.secondary }} tickLine={false} axisLine={false} />
                <Tooltip contentStyle={tooltipStyle} formatter={dayChart.formatter} />
                <Bar dataKey={dayChart.barKey} fill={theme.palette.secondary.main} radius={[3, 3, 0, 0]} name={dayChart.label} />
              </BarChart>
            </ResponsiveContainer>
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
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie data={byMethod} dataKey={methodChart.pieKey} nameKey="method" cx="50%" cy="50%" outerRadius={100} label={true}>
                  {byMethod.map((_, i) => <Cell key={i} fill={COLORS[i % COLORS.length]} />)}
                </Pie>
                <Tooltip contentStyle={tooltipStyle} formatter={methodChart.formatter} />
                <Legend />
              </PieChart>
            </ResponsiveContainer>
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
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie data={byClient} dataKey={clientChart.pieKey} nameKey="client" cx="50%" cy="50%" outerRadius={100} label={true}>
                  {byClient.map((_, i) => <Cell key={i} fill={COLORS[i % COLORS.length]} />)}
                </Pie>
                <Tooltip contentStyle={tooltipStyle} formatter={clientChart.formatter} />
                <Legend />
              </PieChart>
            </ResponsiveContainer>
          </ChartCard>
        </Grid>
      </Grid>
    </>
  )
}
