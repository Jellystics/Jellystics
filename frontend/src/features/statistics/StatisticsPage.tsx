import { useState, useEffect, useMemo } from 'react'
import { Grid, Box, Typography, Chip, Pagination } from '@mui/material'
import { useNavigate } from 'react-router-dom'
import { BarChart } from '@mui/x-charts/BarChart'
import { useTranslation } from 'react-i18next'
import PageHeader from '@/shared/components/PageHeader/PageHeader'
import ChartCard from '@/shared/components/ChartCard/ChartCard'
import StatCard from '@/shared/components/StatCard/StatCard'
import api from '@/lib/axios'
import type { WatchStatOverTime, HourStat, DayStat, PlayMethodStat, ClientStat } from '@/shared/types/stats'
import { Play24Regular, Clock24Regular, CheckmarkCircle24Regular, DismissCircle24Regular, DataPie24Regular } from '@fluentui/react-icons'
import { formatWatchTime } from '@/shared/utils/formatWatchTime'
import { getItemImageUrl } from '@/shared/utils/imageUrl'
import MetricToggle, { type ActivityMetric } from '@/shared/components/MetricToggle/MetricToggle'
import { useChartColors } from '@/lib/chartColors'
import TimeRangeSelector from '@/shared/components/TimeRangeSelector/TimeRangeSelector'

type ActiveUser = { UserId: string; UserName: string; TotalPlays: number; TotalWatchTime: number }
type PlayedItem = { Id: string; Name: string; PlayCount: number; Type: string }

export default function StatisticsPage() {
  const CHART_COLORS = useChartColors()
  const { t } = useTranslation()
  const navigate = useNavigate()

  const [refreshKey, setRefreshKey] = useState(0)
  const refresh = () => setRefreshKey(k => k + 1)

  // ── Global stats (StatCards) ────────────────────────────────────────────
  const [globalStats, setGlobalStats] = useState<{ TotalPlays: number; TotalWatchTime: number } | null>(null)
  const [globalLoading, setGlobalLoading] = useState(true)

  useEffect(() => {
    setGlobalLoading(true)
    api.get('/stats/getGlobalStats')
      .then(r => setGlobalStats(r.data))
      .catch(() => {})
      .finally(() => setGlobalLoading(false))
  }, [refreshKey])

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
  }, [overTimeDays, refreshKey])

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
  }, [hourDays, refreshKey])

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
  }, [dayDays, refreshKey])

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
  }, [methodDays, refreshKey])

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
  }, [clientDays, refreshKey])

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
  }, [usersDays, refreshKey])

  // ── Top items ───────────────────────────────────────────────────────────
  const [itemsDays, setItemsDays] = useState(30)
  const [topItems, setTopItems] = useState<PlayedItem[]>([])
  const [topItemsPage, setTopItemsPage] = useState(0)
  const TOP_ITEMS_PAGE_SIZE = 5
  const [itemsLoading, setItemsLoading] = useState(true)

  useEffect(() => {
    setItemsLoading(true)
    api.get(`/stats/getMostPlayedItems?limit=10&days=${itemsDays}`)
      .then(r => { setTopItems((r.data ?? []).slice(0, 10)); setTopItemsPage(0) })
      .catch(() => setTopItems([]))
      .finally(() => setItemsLoading(false))
  }, [itemsDays, refreshKey])

  // ── Completion rate ──────────────────────────────────────────────────
  const [completionDays, setCompletionDays] = useState(30)
  const [completion, setCompletion] = useState<any>(null)
  const [completionLoading, setCompletionLoading] = useState(true)

  useEffect(() => {
    setCompletionLoading(true)
    api.get(`/stats/getCompletionRate?days=${completionDays}`)
      .then(r => setCompletion(r.data))
      .catch(() => setCompletion(null))
      .finally(() => setCompletionLoading(false))
  }, [completionDays, refreshKey])

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

  const clientMargin = useMemo(() => byClient.length ? Math.max(...byClient.map(d => d.client.length)) * 7 + 16 : 80, [byClient])
  const completionTypeMargin = useMemo(() => completion?.byType?.length ? Math.max(...completion.byType.map((d: any) => d.type.length)) * 7 + 16 : 80, [completion])

  const chartAction = (selector: React.ReactNode, toggle: React.ReactNode) => (
    <Box sx={{ display: 'flex', gap: 1, flexDirection: { xs: 'column', sm: 'row' }, alignItems: { xs: 'flex-end', sm: 'center' } }}>
      {selector}
      {toggle}
    </Box>
  )

  return (
    <>
      <PageHeader title={t('nav.statistics')} onRefresh={refresh} loading={globalLoading} />

      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard label={t('stats.totalPlays')} value={globalStats?.TotalPlays ?? 0} icon={<Play24Regular />} loading={globalLoading} />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard label={t('stats.watchTime')} value={formatWatchTime(globalStats?.TotalWatchTime ?? 0)} icon={<Clock24Regular />} loading={globalLoading} />
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
            <BarChart
              xAxis={[{ data: byMethod.map(d => d.method), scaleType: 'band' }]}
              series={[{ data: byMethod.map(d => methodChart.getPieValue(d)), valueFormatter: methodChart.formatter, color: CHART_COLORS[0] }]}
              height={280}
              sx={{ width: '100%' }}
              grid={{ horizontal: true }}
              slotProps={{ legend: { hidden: true } as any }}
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
            <BarChart
              layout="horizontal"
              yAxis={[{ data: byClient.map(d => d.client), scaleType: 'band', width: clientMargin }]}
              series={[{ data: byClient.map(d => clientChart.getPieValue(d)), valueFormatter: clientChart.formatter, color: CHART_COLORS[0] }]}
              height={280}
              margin={{ left: 4, right: 4, top: 8, bottom: 8 }}
              sx={{ width: '100%' }}
              grid={{ vertical: true }}
              slotProps={{ legend: { hidden: true } as any }}
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
            height="auto"
            action={<TimeRangeSelector value={itemsDays} onChange={setItemsDays} />}
          >
            <Box sx={{ pt: 0.5 }}>
              {topItems
                .slice(topItemsPage * TOP_ITEMS_PAGE_SIZE, (topItemsPage + 1) * TOP_ITEMS_PAGE_SIZE)
                .map((item, i) => (
                <Box
                  key={item.Id}
                  onClick={() => navigate(`/items/${item.Id}`)}
                  sx={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 1.5,
                    py: 0.75,
                    borderBottom: '1px solid',
                    borderColor: 'divider',
                    '&:last-child': { borderBottom: 0 },
                    cursor: 'pointer',
                    borderRadius: 1,
                    mx: -0.5,
                    px: 0.5,
                    transition: 'background 150ms',
                    '&:hover': { bgcolor: 'action.hover' },
                  }}
                >
                  <Typography
                    variant="caption"
                    color="text.secondary"
                    sx={{ minWidth: 18, textAlign: 'right', fontWeight: 600, fontSize: 11, flexShrink: 0 }}
                  >
                    {topItemsPage * TOP_ITEMS_PAGE_SIZE + i + 1}
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
                      src={getItemImageUrl(item.Id, 72, 85)}
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
              {topItems.length > TOP_ITEMS_PAGE_SIZE && (
                <Box sx={{ display: 'flex', justifyContent: 'center', mt: 1 }}>
                  <Pagination
                    count={Math.ceil(topItems.length / TOP_ITEMS_PAGE_SIZE)}
                    page={topItemsPage + 1}
                    onChange={(_, p) => setTopItemsPage(p - 1)}
                    size="small"
                  />
                </Box>
              )}
            </Box>
          </ChartCard>
        </Grid>
      </Grid>

      {/* ── Completion Rate ──────────────────────────────────────────────── */}
      <Typography variant="h6" sx={{ fontWeight: 700, mt: 3, mb: 1 }}>
        {t('insights.completionRate', 'Completion Rate')}
      </Typography>

      <Grid container spacing={2} sx={{ mb: 2 }}>
        <Grid size={{ xs: 6, md: 4 }}>
          <StatCard
            label={t('insights.avgCompletion', 'Avg Completion')}
            value={completion ? `${Math.round(completion.overall.avgCompletionRate * 100)}%` : '—'}
            icon={<DataPie24Regular />}
            loading={completionLoading}
          />
        </Grid>
        <Grid size={{ xs: 6, md: 4 }}>
          <StatCard
            label={t('insights.completedPlays', 'Completed Plays')}
            value={completion?.overall?.completedPlays ?? 0}
            icon={<CheckmarkCircle24Regular />}
            loading={completionLoading}
          />
        </Grid>
        <Grid size={{ xs: 6, md: 4 }}>
          <StatCard
            label={t('insights.abandonedPlays', 'Abandoned Plays')}
            value={completion?.overall?.abandonedPlays ?? 0}
            icon={<DismissCircle24Regular />}
            loading={completionLoading}
          />
        </Grid>
      </Grid>

      <Grid container spacing={2} sx={{ mb: 2 }}>
        <Grid size={{ xs: 12, md: 6 }}>
          <ChartCard
            title={t('insights.completionDistribution', 'Completion Distribution')}
            loading={completionLoading}
            empty={!completion?.distribution?.length}
            height={280}
            action={<TimeRangeSelector value={completionDays} onChange={setCompletionDays} />}
          >
            <BarChart
              xAxis={[{ data: (completion?.distribution ?? []).map((d: any) => d.bucket), scaleType: 'band' }]}
              series={[{ data: (completion?.distribution ?? []).map((d: any) => d.count), label: t('common.plays', 'Plays'), valueFormatter: (v) => String(v ?? 0), color: CHART_COLORS[0] }]}
              height={280}
              sx={{ width: '100%' }}
              grid={{ horizontal: true }}
              slotProps={{ legend: { hidden: true } as any }}
            />
          </ChartCard>
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <ChartCard
            title={t('insights.completionByType', 'Completion by Type')}
            loading={completionLoading}
            empty={!completion?.byType?.length}
            height={280}
            action={<TimeRangeSelector value={completionDays} onChange={setCompletionDays} />}
          >
            <BarChart
              layout="horizontal"
              yAxis={[{ data: (completion?.byType ?? []).map((d: any) => d.type), scaleType: 'band', width: completionTypeMargin }]}
              series={[{ data: (completion?.byType ?? []).map((d: any) => Math.round(d.avgCompletionRate * 100)), label: t('insights.completionPercent', 'Completion %'), valueFormatter: (v) => `${v ?? 0}%`, color: CHART_COLORS[1] }]}
              height={280}
              margin={{ left: 4, right: 4, top: 8, bottom: 8 }}
              sx={{ width: '100%' }}
              grid={{ vertical: true }}
              slotProps={{ legend: { hidden: true } as any }}
            />
          </ChartCard>
        </Grid>
      </Grid>
    </>
  )
}
