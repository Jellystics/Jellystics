import { useState, useEffect } from 'react'
import { LineChart, lineClasses } from '@mui/x-charts/LineChart'
import { labelMarkClasses } from '@mui/x-charts/ChartsLabel'
import { Box, ButtonGroup, Button, Tooltip, IconButton } from '@mui/material'
import { format, parseISO } from 'date-fns'
import ChartCard from '@/shared/components/ChartCard/ChartCard'
import { useTranslation } from 'react-i18next'
import MetricToggle, { type ActivityMetric } from '@/shared/components/MetricToggle/MetricToggle'
import { formatWatchTime } from '@/shared/utils/formatWatchTime'
import { getDateLocale } from '@/lib/dateLocale'
import api from '@/lib/axios'
import { ChartMultiple24Regular } from '@fluentui/react-icons'
import { useChartColors } from '@/lib/chartColors'

interface ActivityPoint {
  date: string
  plays: number
  duration: number
}

const PRESETS = [
  { days: 7, label: '7d' },
  { days: 30, label: '1m' },
  { days: 90, label: '3m' },
]

export default function ActivityChart() {
  const chartColors = useChartColors()
  const COLOR_PLAYS = chartColors[0]
  const COLOR_DURATION = chartColors[1]
  const { t } = useTranslation()
  const [days, setDays] = useState(30)
  const [metric, setMetric] = useState<ActivityMetric>('count')
  const [combined, setCombined] = useState(false)
  const [data, setData] = useState<ActivityPoint[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setLoading(true)
    api
      .get(`/stats/getWatchStatisticsOverTime?days=${days}`)
      .then((r) => setData(Array.isArray(r.data) ? r.data : []))
      .catch(() => setData([]))
      .finally(() => setLoading(false))
  }, [days])

  const labels = data.map((d) => format(parseISO(d.date), 'MMM d', { locale: getDateLocale() }))
  const playsValues = data.map((d) => d.plays)
  const durationValues = data.map((d) => d.duration)
  const singleValues = data.map((d) => (metric === 'duration' ? d.duration : d.plays))
  const singleLabel = metric === 'duration' ? t('stats.watchTime') : t('common.plays')

  const actions = (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
      <ButtonGroup size="small" variant="outlined" disableElevation>
        {PRESETS.map((p) => (
          <Button
            key={p.days}
            onClick={() => setDays(p.days)}
            variant={days === p.days ? 'contained' : 'outlined'}
            sx={{ minWidth: 36, px: 1, fontSize: 11, fontWeight: 600, py: 0.25 }}
          >
            {p.label}
          </Button>
        ))}
      </ButtonGroup>
      {!combined && <MetricToggle value={metric} onChange={setMetric} />}
      <Tooltip title={combined ? t('dashboard.singleView', 'Vue simple') : t('dashboard.combinedView', 'Vue combinée')}>
        <IconButton
          size="small"
          onClick={() => setCombined((v) => !v)}
          color={combined ? 'primary' : 'default'}
          sx={{ ml: 0.5 }}
        >
          <ChartMultiple24Regular style={{ fontSize: 18 }} />
        </IconButton>
      </Tooltip>
    </Box>
  )

  return (
    <ChartCard
      title={t('dashboard.activity', 'Activity')}
      loading={loading}
      empty={!loading && data.length === 0}
      height={combined ? 260 : 220}
      action={actions}
    >
      {combined ? (
        <LineChart
          xAxis={[{ id: 'x', scaleType: 'point', data: labels, height: 28 }]}
          yAxis={[
            { id: 'plays', width: 40 },
            { id: 'duration', width: 40 },
          ]}
          series={[
            {
              id: 'plays',
              yAxisId: 'plays',
              data: playsValues,
              label: t('common.plays'),
              color: COLOR_PLAYS,
              showMark: false,
              labelMarkType: 'line',
              valueFormatter: (v) => String(v ?? 0),
            },
            {
              id: 'duration',
              yAxisId: 'duration',
              data: durationValues,
              label: t('stats.watchTime'),
              color: COLOR_DURATION,
              showMark: false,
              labelMarkType: 'line',
              valueFormatter: (v) => formatWatchTime(v ?? 0),
            },
          ]}
          rightAxis="duration"
          height={260}
          sx={{
            width: '100%',
            [`& .${lineClasses.line}[data-series="duration"], [data-series="duration"] .${labelMarkClasses.fill}`]: {
              strokeDasharray: '5 4',
              strokeWidth: 1.5,
            },
            [`& .${lineClasses.line}[data-series="plays"], [data-series="plays"] .${labelMarkClasses.fill}`]: {
              strokeWidth: 1.5,
            },
          }}
          grid={{ horizontal: true }}
          slotProps={{ legend: { position: { vertical: 'top', horizontal: 'right' }, padding: { top: -4 } } }}
          margin={{ right: 48, left: 40, top: 28, bottom: 28 }}
        />
      ) : (
        <LineChart
          xAxis={[{ data: labels, scaleType: 'point' }]}
          series={[{
            data: singleValues,
            area: true,
            label: singleLabel,
            valueFormatter: (v) => metric === 'duration' ? formatWatchTime(v ?? 0) : String(v ?? 0),
            color: metric === 'duration' ? COLOR_DURATION : COLOR_PLAYS,
            showMark: false,
          }]}
          height={220}
          sx={{ width: '100%' }}
          grid={{ horizontal: true }}
          slotProps={{ legend: { hidden: true } }}
        />
      )}
    </ChartCard>
  )
}
