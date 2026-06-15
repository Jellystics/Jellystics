import { ToggleButton, ToggleButtonGroup } from '@mui/material'
import { useTranslation } from 'react-i18next'

export type ActivityMetric = 'count' | 'duration'

interface MetricToggleProps {
  value: ActivityMetric
  onChange: (value: ActivityMetric) => void
}

export default function MetricToggle({ value, onChange }: MetricToggleProps) {
  const { t } = useTranslation()
  return (
    <ToggleButtonGroup
      exclusive
      size="small"
      value={value}
      onChange={(_, next) => {
        if (next) onChange(next)
      }}
      sx={{
        '& .MuiToggleButton-root': {
          px: 1.25,
          py: 0.35,
          fontSize: 11,
          textTransform: 'none',
        },
      }}
    >
      <ToggleButton value="count">{t('common.plays')}</ToggleButton>
      <ToggleButton value="duration">{t('stats.watchTime')}</ToggleButton>
    </ToggleButtonGroup>
  )
}
