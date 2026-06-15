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

const COLORS = ['#a78bfa', '#7c3aed', '#6d28d9', '#5b21b6', '#4c1d95', '#8b5cf6', '#c4b5fd', '#ede9fe']
const DAYS_OF_WEEK = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']

export default function StatisticsPage() {
  const { t } = useTranslation()
  const theme = useTheme()
  const [overTime, setOverTime] = useState<WatchStatOverTime[]>([])
  const [byHour, setByHour] = useState<HourStat[]>([])
  const [byDay, setByDay] = useState<DayStat[]>([])
  const [byMethod, setByMethod] = useState<PlayMethodStat[]>([])
  const [byClient, setByClient] = useState<ClientStat[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
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
  }, [t])

  const totalPlays = overTime.reduce((s, d) => s + d.plays, 0)
  const totalDuration = overTime.reduce((s, d) => s + d.duration, 0)

  const hourData = Array.from({ length: 24 }, (_, h) => ({
    hour: `${String(h).padStart(2, '0')}h`,
    plays: byHour.find((d) => d.hour === h)?.plays ?? 0,
  }))

  const dayData = DAYS_OF_WEEK.map((day, i) => ({
    day,
    plays: byDay.find((d) => d.day === String(i))?.plays ?? 0,
  }))

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
          <StatCard label={t('stats.watchTime30d')} value={`${Math.floor(totalDuration / 60)}h`} icon={<Clock24Regular />} loading={loading} />
        </Grid>
      </Grid>

      <Grid container spacing={2} sx={{ mb: 2 }}>
        <Grid size={{ xs: 12 }}>
          <ChartCard title={t('stats.playsOverTime30d')} loading={loading} height={240}>
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={overTime} margin={{ top: 4, right: 4, bottom: 0, left: -20 }}>
                <CartesianGrid strokeDasharray="3 3" stroke={theme.palette.divider} />
                <XAxis dataKey="date" tick={{ fontSize: 10, fill: theme.palette.text.secondary }} tickLine={false} axisLine={false} />
                <YAxis tick={{ fontSize: 11, fill: theme.palette.text.secondary }} tickLine={false} axisLine={false} />
                <Tooltip contentStyle={tooltipStyle} />
                <Bar dataKey="plays" fill={theme.palette.primary.main} radius={[4, 4, 0, 0]} name={t('common.plays')} />
              </BarChart>
            </ResponsiveContainer>
          </ChartCard>
        </Grid>
      </Grid>

      <Grid container spacing={2} sx={{ mb: 2 }}>
        <Grid size={{ xs: 12, md: 6 }}>
          <ChartCard title={t('stats.playsByHour')} loading={loading} height={220}>
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={hourData} margin={{ top: 4, right: 4, bottom: 0, left: -20 }}>
                <CartesianGrid strokeDasharray="3 3" stroke={theme.palette.divider} />
                <XAxis dataKey="hour" tick={{ fontSize: 9, fill: theme.palette.text.secondary }} tickLine={false} axisLine={false} />
                <YAxis tick={{ fontSize: 11, fill: theme.palette.text.secondary }} tickLine={false} axisLine={false} />
                <Tooltip contentStyle={tooltipStyle} />
                <Bar dataKey="plays" fill={theme.palette.primary.main} radius={[3, 3, 0, 0]} name={t('common.plays')} />
              </BarChart>
            </ResponsiveContainer>
          </ChartCard>
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <ChartCard title={t('stats.playsByDayOfWeek')} loading={loading} height={220}>
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={dayData} margin={{ top: 4, right: 4, bottom: 0, left: -20 }}>
                <CartesianGrid strokeDasharray="3 3" stroke={theme.palette.divider} />
                <XAxis dataKey="day" tick={{ fontSize: 11, fill: theme.palette.text.secondary }} tickLine={false} axisLine={false} />
                <YAxis tick={{ fontSize: 11, fill: theme.palette.text.secondary }} tickLine={false} axisLine={false} />
                <Tooltip contentStyle={tooltipStyle} />
                <Bar dataKey="plays" fill={theme.palette.secondary.main} radius={[3, 3, 0, 0]} name={t('common.plays')} />
              </BarChart>
            </ResponsiveContainer>
          </ChartCard>
        </Grid>
      </Grid>

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, md: 6 }}>
          <ChartCard title={t('stats.playbackMethod')} loading={loading} height={280}>
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie data={byMethod} dataKey="count" nameKey="method" cx="50%" cy="50%" outerRadius={100} label={true}>
                  {byMethod.map((_, i) => <Cell key={i} fill={COLORS[i % COLORS.length]} />)}
                </Pie>
                <Tooltip contentStyle={tooltipStyle} />
                <Legend />
              </PieChart>
            </ResponsiveContainer>
          </ChartCard>
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <ChartCard title={t('stats.topClients')} loading={loading} height={280}>
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie data={byClient} dataKey="count" nameKey="client" cx="50%" cy="50%" outerRadius={100} label={true}>
                  {byClient.map((_, i) => <Cell key={i} fill={COLORS[i % COLORS.length]} />)}
                </Pie>
                <Tooltip contentStyle={tooltipStyle} />
                <Legend />
              </PieChart>
            </ResponsiveContainer>
          </ChartCard>
        </Grid>
      </Grid>
    </>
  )
}
