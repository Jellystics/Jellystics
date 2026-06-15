import { ToggleButton, ToggleButtonGroup } from '@mui/material'

export type ActivityMetric = 'count' | 'duration'

interface MetricToggleProps {
  value: ActivityMetric
  onChange: (value: ActivityMetric) => void
}

export default function MetricToggle({ value, onChange }: MetricToggleProps) {
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
      <ToggleButton value="count">Lectures</ToggleButton>
      <ToggleButton value="duration">Durée</ToggleButton>
    </ToggleButtonGroup>
  )
}
