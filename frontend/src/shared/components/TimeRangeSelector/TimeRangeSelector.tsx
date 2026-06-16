import { ToggleButtonGroup, ToggleButton } from '@mui/material'
import { useTranslation } from 'react-i18next'

interface Props {
  value: number
  onChange: (days: number) => void
}

export default function TimeRangeSelector({ value, onChange }: Props) {
  const { t } = useTranslation()
  return (
    <ToggleButtonGroup
      value={value}
      exclusive
      onChange={(_, v) => { if (v !== null) onChange(v) }}
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
