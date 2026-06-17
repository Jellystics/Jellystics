import { ToggleButton, ToggleButtonGroup } from '@mui/material'
import { useTranslation } from 'react-i18next'

export type ActivityMetric = 'count' | 'duration'

interface MetricToggleProps {
  value: ActivityMetric
  onChange: (value: ActivityMetric) => void
}

export default function MetricToggle({ value, onChange }: MetricToggleProps) {
  const { t } = useTranslation()
  const playsLabel = t('common.plays')
  const capitalizedPlays = playsLabel.charAt(0).toUpperCase() + playsLabel.slice(1)

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
      <ToggleButton value="count">{capitalizedPlays}</ToggleButton>
      <ToggleButton value="duration">{t('stats.watchTime')}</ToggleButton>
    </ToggleButtonGroup>
  )
}
