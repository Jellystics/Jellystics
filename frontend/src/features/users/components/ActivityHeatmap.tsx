import { Box, Tooltip, Typography } from '@mui/material'
import { useTheme } from '@mui/material/styles'
import { useTranslation } from 'react-i18next'
import type { UserActivity } from '@/shared/types/user'
import { formatWatchTime } from '@/shared/utils/formatWatchTime'
import MetricToggle, { type ActivityMetric } from '@/shared/components/MetricToggle/MetricToggle'

interface ActivityHeatmapProps {
  data: UserActivity[]
  metric: ActivityMetric
  onMetricChange?: (metric: ActivityMetric) => void
}

const WEEKS = 52
const DAYS = 7

function toLocalDateKey(date: Date) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function hexToRgb(hex: string) {
  const r = parseInt(hex.slice(1, 3), 16)
  const g = parseInt(hex.slice(3, 5), 16)
  const b = parseInt(hex.slice(5, 7), 16)
  return { r, g, b }
}

export default function ActivityHeatmap({ data, metric, onMetricChange }: ActivityHeatmapProps) {
  const theme = useTheme()
  const { t } = useTranslation()

  const dataMap = new Map(data.map((d) => [d.date, metric === 'duration' ? d.duration ?? 0 : d.count]))
  const maxCount = Math.max(...data.map((d) => metric === 'duration' ? d.duration ?? 0 : d.count), 1)

  const today = new Date()
  const startDate = new Date(today)
  startDate.setDate(startDate.getDate() - (WEEKS * DAYS - 1))

  const days: { date: string; count: number }[] = []
  for (let i = 0; i < WEEKS * DAYS; i++) {
    const d = new Date(startDate)
    d.setDate(startDate.getDate() + i)
    const key = toLocalDateKey(d)
    days.push({ date: key, count: dataMap.get(key) ?? 0 })
  }

  const { r, g, b } = hexToRgb(theme.palette.primary.main)

  const CELL = 12
  const GAP = 2

  return (
    <Box>
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 2, mb: 1 }}>
        <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
          {t('users.activityHeatmap')}
        </Typography>
        {onMetricChange && <MetricToggle value={metric} onChange={onMetricChange} />}
      </Box>
      <Box sx={{ overflowX: 'auto', pb: 1 }}>
        <Box
          sx={{
            display: 'grid',
            gridTemplateColumns: `repeat(${WEEKS}, ${CELL}px)`,
            gridTemplateRows: `repeat(${DAYS}, ${CELL}px)`,
            gridAutoFlow: 'column',
            gap: `${GAP}px`,
            width: 'fit-content',
          }}
        >
          {days.map(({ date, count }) => {
            const intensity = count > 0 ? Math.min(count / maxCount, 1) : 0
            const opacity = count > 0 ? 0.15 + intensity * 0.85 : 0.06
            const label = metric === 'duration' ? formatWatchTime(count) : `${count} ${t('common.plays')}`
            return (
              <Tooltip key={date} title={`${date}: ${label}`} arrow>
                <Box
                  sx={{
                    width: CELL,
                    height: CELL,
                    borderRadius: 0.5,
                    bgcolor: `rgba(${r},${g},${b},${opacity})`,
                    cursor: 'pointer',
                    transition: 'transform 0.1s',
                    '&:hover': { transform: 'scale(1.3)' },
                  }}
                />
              </Tooltip>
            )
          })}
        </Box>
      </Box>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mt: 0.5 }}>
        <Typography variant="caption" color="text.secondary">{t('common.less')}</Typography>
        {[0.06, 0.3, 0.55, 0.8, 1].map((op) => (
          <Box key={op} sx={{ width: CELL, height: CELL, borderRadius: 0.5, bgcolor: `rgba(${r},${g},${b},${op})` }} />
        ))}
        <Typography variant="caption" color="text.secondary">{t('common.more')}</Typography>
      </Box>
    </Box>
  )
}
