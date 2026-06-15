import { ResponsiveContainer, AreaChart, Area, XAxis, YAxis, Tooltip, CartesianGrid } from 'recharts'
import { useTheme } from '@mui/material'
import { format, parseISO } from 'date-fns'
import ChartCard from '@/shared/components/ChartCard/ChartCard'
import { useTranslation } from 'react-i18next'
import MetricToggle, { type ActivityMetric } from '@/shared/components/MetricToggle/MetricToggle'
import { formatWatchTime } from '@/shared/utils/formatWatchTime'
import { getDateLocale } from '@/lib/dateLocale'

interface ActivityPoint {
  date: string
  plays: number
  duration: number
}

interface ActivityChartProps {
  data: ActivityPoint[]
  loading: boolean
  metric: ActivityMetric
  onMetricChange: (metric: ActivityMetric) => void
}

export default function ActivityChart({ data, loading, metric, onMetricChange }: ActivityChartProps) {
  const theme = useTheme()
  const { t } = useTranslation()

  const formatted = data.map((d) => ({
    ...d,
    label: format(parseISO(d.date), 'MMM d', { locale: getDateLocale() }),
    count: d.plays,
  }))
  const dataKey = metric === 'duration' ? 'duration' : 'count'
  const label = metric === 'duration' ? t('stats.watchTime') : t('common.plays')

  const isEmpty = data.length === 0

  return (
    <ChartCard
      title={t('dashboard.activityLast7Days')}
      loading={loading}
      empty={isEmpty}
      height={220}
      action={<MetricToggle value={metric} onChange={onMetricChange} />}
    >
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={formatted} margin={{ top: 4, right: 4, bottom: 0, left: -20 }}>
          <defs>
            <linearGradient id="playGrad" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={theme.palette.primary.main} stopOpacity={0.3} />
              <stop offset="95%" stopColor={theme.palette.primary.main} stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" stroke={theme.palette.divider} />
          <XAxis dataKey="label" tick={{ fontSize: 11, fill: theme.palette.text.secondary }} tickLine={false} axisLine={false} />
          <YAxis tick={{ fontSize: 11, fill: theme.palette.text.secondary }} tickLine={false} axisLine={false} />
          <Tooltip
            formatter={(value) => metric === 'duration' ? formatWatchTime(Number(value)) : value}
            contentStyle={{
              backgroundColor: theme.palette.background.paper,
              border: `1px solid ${theme.palette.divider}`,
              borderRadius: 8,
              fontSize: 12,
            }}
            itemStyle={{ color: theme.palette.text.primary }}
            labelStyle={{ color: theme.palette.text.secondary, marginBottom: 4 }}
          />
          <Area
            type="monotone"
            dataKey={dataKey}
            stroke={theme.palette.primary.main}
            strokeWidth={2}
            fill="url(#playGrad)"
            name={label}
          />
        </AreaChart>
      </ResponsiveContainer>
    </ChartCard>
  )
}
