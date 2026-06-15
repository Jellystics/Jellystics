import { ResponsiveContainer, AreaChart, Area, XAxis, YAxis, Tooltip, CartesianGrid } from 'recharts'
import { useTheme } from '@mui/material'
import { format, parseISO } from 'date-fns'
import ChartCard from '@/shared/components/ChartCard/ChartCard'
import { useTranslation } from 'react-i18next'

interface ActivityPoint {
  date: string
  plays: number
  duration: number
}

interface ActivityChartProps {
  data: ActivityPoint[]
  loading: boolean
}

export default function ActivityChart({ data, loading }: ActivityChartProps) {
  const theme = useTheme()
  const { t } = useTranslation()

  const formatted = data.map((d) => ({
    ...d,
    label: format(parseISO(d.date), 'MMM d'),
  }))

  return (
    <ChartCard title={t('dashboard.activityLast7Days')} loading={loading} height={220}>
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
            dataKey="plays"
            stroke={theme.palette.primary.main}
            strokeWidth={2}
            fill="url(#playGrad)"
            name={t('common.plays')}
          />
        </AreaChart>
      </ResponsiveContainer>
    </ChartCard>
  )
}
