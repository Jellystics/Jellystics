import { ToggleButtonGroup, ToggleButton } from '@mui/material'
import { useTranslation } from 'react-i18next'
import { differenceInDays, startOfDay } from 'date-fns'
import { useDateRange } from '@/lib/dateRange'

export default function TimeRangeSelector() {
  const { t } = useTranslation()
  const { from, to, setPreset } = useDateRange()

  // Derive which toggle is active: compute full-day diff to match preset values
  const diffDays = differenceInDays(startOfDay(to), startOfDay(from))
  // 0 = "All time" preset (from = 2000-01-01), otherwise match 7/30/90
  const isAllTime = from.getFullYear() <= 2000
  const value = isAllTime ? 0 : [7, 30, 90].includes(diffDays) ? diffDays : null

  return (
    <ToggleButtonGroup
      value={value}
      exclusive
      onChange={(_, v) => { if (v !== null) setPreset(v as number) }}
      size="small"
      sx={{ height: 32 }}
    >
      <ToggleButton value={7} sx={{ px: 1.5, textTransform: 'none', fontSize: 13 }}>7d</ToggleButton>
      <ToggleButton value={30} sx={{ px: 1.5, textTransform: 'none', fontSize: 13 }}>30d</ToggleButton>
      <ToggleButton value={90} sx={{ px: 1.5, textTransform: 'none', fontSize: 13 }}>90d</ToggleButton>
      <ToggleButton value={0} sx={{ px: 1.5, textTransform: 'none', fontSize: 13 }}>{t('common.all')}</ToggleButton>
    </ToggleButtonGroup>
  )
}
