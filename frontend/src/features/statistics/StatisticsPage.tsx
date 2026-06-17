import { useState, useEffect } from 'react'
import { Grid, Box, Typography, Chip } from '@mui/material'
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
import { useChartColors } from '@/lib/chartColors'
import TimeRangeSelector from '@/shared/components/TimeRangeSelector/TimeRangeSelector'

type ActiveUser = { UserId: string; UserName: string; TotalPlays: number; TotalWatchTime: number }
type PlayedItem = { Id: string; Name: string; PlayCount: number; Type: string }

export default function StatisticsPage() {
  const CHART_COLORS = useChartColors()
  const { t } = useTranslation()

  // ── Plays over time ─────────────────────────────────────────────────────
  const [overTimeDays, setOverTimeDays] = useState(30)
  const [overTime, setOverTime] = useState<WatchStatOverTime[]>([])
  const [overTimeLoading, setOverTimeLoading] = useState(true)
  const [overTimeMetric, setOverTimeMetric] = useState<ActivityMetric>('count')

  useEffect(() => {
    setOverTimeLoading(true)
    api.get(`/stats/getWatchStatisticsOverTime?days=${overTimeDays}`)
      .then(r => setOverTime(r.data ?? []))
      .catch(() => setOverTime([]))
      .finally(() => setOverTimeLoading(false))
  }, [overTimeDays])

  // ── Plays by hour ───────────────────────────────────────────────────────
  const [hourDays, setHourDays] = useState(30)
  const [byHour, setByHour] = useState<HourStat[]>([])
  const [hourLoading, setHourLoading] = useState(true)
  const [hourMetric, setHourMetric] = useState<ActivityMetric>('count')

  useEffect(() => {
    setHourLoading(true)
    api.get(`/stats/getPopularHourOfDay?days=${hourDays}`)
      .then(r => setByHour(r.data ?? []))
      .catch(() => setByHour([]))
      .finally(() => setHourLoading(false))
  }, [hourDays])

  // ── Plays by day of week ────────────────────────────────────────────────
  const [dayDays, setDayDays] = useState(30)
  const [byDay, setByDay] = useState<DayStat[]>([])
  const [dayLoading, setDayLoading] = useState(true)
  const [dayMetric, setDayMetric] = useState<ActivityMetric>('count')

  useEffect(() => {
    setDayLoading(true)
    api.get(`/stats/getPopularDayOfWeek?days=${dayDays}`)
      .then(r => setByDay(r.data ?? []))
      .catch(() => setByDay([]))
      .finally(() => setDayLoading(false))
  }, [dayDays])

  // ── Playback method ─────────────────────────────────────────────────────
  const [methodDays, setMethodDays] = useState(30)
  const [byMethod, setByMethod] = useState<PlayMethodStat[]>([])
  const [methodLoading, setMethodLoading] = useState(true)
  const [methodMetric, setMethodMetric] = useState<ActivityMetric>('count')

  useEffect(() => {
    setMethodLoading(true)
    api.get(`/stats/getMostUsedPlaybackMethod?days=${methodDays}`)
      .then(r => setByMethod(r.data ?? []))
      .catch(() => setByMethod([]))
      .finally(() => setMethodLoading(false))
  }, [methodDays])

  // ── Top clients ─────────────────────────────────────────────────────────
  const [clientDays, setClientDays] = useState(30)
  const [byClient, setByClient] = useState<ClientStat[]>([])
  const [clientLoading, setClientLoading] = useState(true)
  const [clientMetric, setClientMetric] = useState<ActivityMetric>('count')

  useEffect(() => {
    setClientLoading(true)
    api.get(`/stats/getMostUsedClients?days=${clientDays}`)
      .then(r => setByClient(r.data ?? []))
      .catch(() => setByClient([]))
      .finally(() => setClientLoading(false))
  }, [clientDays])

  // ── Most active users ───────────────────────────────────────────────────
  const [usersDays, setUsersDays] = useState(30)
  const [activeUsers, setActiveUsers] = useState<ActiveUser[]>([])
  const [usersLoading, setUsersLoading] = useState(true)

  useEffect(() => {
    setUsersLoading(true)
    api.get(`/stats/getMostActiveUsers?limit=8&days=${usersDays}`)
      .then(r => setActiveUsers((r.data ?? []).slice(0, 8)))
      .catch(() => setActiveUsers([]))
      .finally(() => setUsersLoading(false))
  }, [usersDays])

  // ── Top items ───────────────────────────────────────────────────────────
  const [itemsDays, setItemsDays] = useState(30)
  const [topItems, setTopItems] = useState<PlayedItem[]>([])
  const [itemsLoading, setItemsLoading] = useState(true)

  useEffect(() => {
    setItemsLoading(true)
    api.get(`/stats/getMostPlayedItems?limit=10&days=${itemsDays}`)
      .then(r => setTopItems((r.data ?? []).slice(0, 10)))
      .catch(() => setTopItems([]))
      .finally(() => setItemsLoading(false))
  }, [itemsDays])

  // ── Derived values ──────────────────────────────────────────────────────
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
    plays: byDay.find((d) => d.day === i)?.plays ?? 0,
    duration: byDay.find((d) => d.day === i)?.duration ?? 0,
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

  const chartAction = (selector: React.ReactNode, toggle: React.ReactNode) => (
    <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
      {selector}
      {toggle}
    </Box>
  )

  return (
    <>
      <PageHeader title={t('nav.statistics')} />

      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard label={t('stats.totalPlays')} value={totalPlays} icon={<Play24Regular />} loading={overTimeLoading} />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard label={t('stats.watchTime')} value={formatWatchTime(totalDuration)} icon={<Clock24Regular />} loading={overTimeLoading} />
        </Grid>
      </Grid>

      <Grid container spacing={2} sx={{ mb: 2 }}>
        <Grid size={{ xs: 12 }}>
          <ChartCard
            title={t('stats.playsOverTime')}
            loading={overTimeLoading}
            empty={overTime.length === 0}
            height={240}
            action={chartAction(
              <TimeRangeSelector value={overTimeDays} onChange={setOverTimeDays} />,
              <MetricToggle value={overTimeMetric} onChange={setOverTimeMetric} />,
            )}
          >
            <BarChart
              xAxis={[{ data: overTime.map((d) => d.date), scaleType: 'band' }]}
              series={[{ data: overTime.map((d) => otChart.getValue(d)), label: otChart.label, valueFormatter: otChart.formatter, color: CHART_COLORS[0] }]}
              height={240}
              sx={{ width: '100%' }}
              grid={{ horizontal: true }}
              slotProps={{ legend: { hidden: true } as any }}
            />
          </ChartCard>
        </Grid>
      </Grid>

      <Grid container spacing={2} sx={{ mb: 2 }}>
        <Grid size={{ xs: 12, md: 6 }}>
          <ChartCard
            title={t('stats.playsByHour')}
            loading={hourLoading}
            empty={byHour.length === 0}
            height={220}
            action={chartAction(
              <TimeRangeSelector value={hourDays} onChange={setHourDays} />,
              <MetricToggle value={hourMetric} onChange={setHourMetric} />,
            )}
          >
            <BarChart
              xAxis={[{ data: hourData.map((d) => d.hour), scaleType: 'band' }]}
              series={[{ data: hourData.map((d) => hourChart.getValue(d)), label: hourChart.label, valueFormatter: hourChart.formatter, color: CHART_COLORS[0] }]}
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
            empty={byDay.length === 0}
            height={220}
            action={chartAction(
              <TimeRangeSelector value={dayDays} onChange={setDayDays} />,
              <MetricToggle value={dayMetric} onChange={setDayMetric} />,
            )}
          >
            <BarChart
              xAxis={[{ data: dayData.map((d) => d.day), scaleType: 'band' }]}
              series={[{ data: dayData.map((d) => dayChart.getValue(d)), label: dayChart.label, valueFormatter: dayChart.formatter, color: CHART_COLORS[0] }]}
              height={220}
              sx={{ width: '100%' }}
              grid={{ horizontal: true }}
              slotProps={{ legend: { hidden: true } as any }}
            />
          </ChartCard>
        </Grid>
      </Grid>

      <Grid container spacing={2} sx={{ mb: 2 }}>
        <Grid size={{ xs: 12, md: 6 }}>
          <ChartCard
            title={t('stats.playbackMethod')}
            loading={methodLoading}
            empty={byMethod.length === 0}
            height={280}
            action={chartAction(
              <TimeRangeSelector value={methodDays} onChange={setMethodDays} />,
              <MetricToggle value={methodMetric} onChange={setMethodMetric} />,
            )}
          >
            <PieChart
              series={[{
                data: byMethod.map((item, i) => ({ id: i, value: methodChart.getPieValue(item), label: item.method, color: CHART_COLORS[i % CHART_COLORS.length] })),
                outerRadius: 100, paddingAngle: 2, cornerRadius: 3,
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
            loading={clientLoading}
            empty={byClient.length === 0}
            height={280}
            action={chartAction(
              <TimeRangeSelector value={clientDays} onChange={setClientDays} />,
              <MetricToggle value={clientMetric} onChange={setClientMetric} />,
            )}
          >
            <PieChart
              series={[{
                data: byClient.map((item, i) => ({ id: i, value: clientChart.getPieValue(item), label: item.client, color: CHART_COLORS[i % CHART_COLORS.length] })),
                outerRadius: 100, paddingAngle: 2, cornerRadius: 3,
                valueFormatter: (item) => clientChart.formatter(item.value),
              }]}
              height={280}
              sx={{ width: '100%' }}
            />
          </ChartCard>
        </Grid>
      </Grid>

      <Grid container spacing={2} sx={{ mb: 2 }}>
        <Grid size={{ xs: 12 }}>
          <ChartCard
            title={t('stats.mostActiveUsers')}
            loading={usersLoading}
            empty={activeUsers.length === 0}
            height={220}
            action={<TimeRangeSelector value={usersDays} onChange={setUsersDays} />}
          >
            <BarChart
              xAxis={[{ data: activeUsers.map((d) => d.UserName), scaleType: 'band' }]}
              series={[{ data: activeUsers.map((d) => d.TotalPlays), label: t('common.plays'), valueFormatter: (v) => String(v ?? 0), color: CHART_COLORS[0] }]}
              height={220}
              sx={{ width: '100%' }}
              grid={{ horizontal: true }}
              slotProps={{ legend: { hidden: true } as any }}
            />
          </ChartCard>
        </Grid>
      </Grid>

      <Grid container spacing={2} sx={{ mb: 2 }}>
        <Grid size={{ xs: 12 }}>
          <ChartCard
            title={t('stats.topItems')}
            loading={itemsLoading}
            empty={topItems.length === 0}
            height={480}
            action={<TimeRangeSelector value={itemsDays} onChange={setItemsDays} />}
          >
            <Box sx={{ pt: 0.5 }}>
              {topItems.map((item, i) => (
                <Box
                  key={item.Id}
                  sx={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 1.5,
                    py: 0.75,
                    borderBottom: '1px solid',
                    borderColor: 'divider',
                    '&:last-child': { borderBottom: 0 },
                  }}
                >
                  <Typography
                    variant="caption"
                    color="text.secondary"
                    sx={{ minWidth: 18, textAlign: 'right', fontWeight: 600, fontSize: 11, flexShrink: 0 }}
                  >
                    {i + 1}
                  </Typography>

                  <Box sx={{ flex: 1, minWidth: 0 }}>
                    <Typography variant="body2" sx={{ fontSize: 13, fontWeight: 500 }}>
                      {item.Name}
                    </Typography>
                    <Typography variant="caption" color="text.secondary" sx={{ fontSize: 11 }}>
                      {item.Type}
                    </Typography>
                  </Box>

                  <Box
                    sx={{
                      position: 'relative',
                      width: 36,
                      height: 52,
                      borderRadius: 1,
                      overflow: 'hidden',
                      flexShrink: 0,
                      bgcolor: 'rgba(128,128,128,0.1)',
                    }}
                  >
                    <img
                      src={`/proxy/Items/Images/Primary/?id=${item.Id}&fillWidth=72&quality=85`}
                      alt={item.Name}
                      style={{ position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover' }}
                      onError={(e) => { e.currentTarget.style.display = 'none' }}
                    />
                  </Box>

                  <Chip
                    label={`${item.PlayCount} ${t('common.plays')}`}
                    size="small"
                    sx={{ fontSize: 11, height: 20, flexShrink: 0 }}
                  />
                </Box>
              ))}
            </Box>
          </ChartCard>
        </Grid>
      </Grid>
    </>
  )
}
