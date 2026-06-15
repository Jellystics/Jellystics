import { useState } from 'react'
import { Box, Button, ButtonGroup, Popover, Stack, Typography } from '@mui/material'
import { DatePicker } from '@mui/x-date-pickers/DatePicker'
import { differenceInDays, startOfDay, subDays } from 'date-fns'
import { useTranslation } from 'react-i18next'

const PRESETS = [7, 14, 30, 90]

interface TimeRangePickerProps {
  value: number
  onChange: (days: number) => void
}

export default function TimeRangePicker({ value, onChange }: TimeRangePickerProps) {
  const { t } = useTranslation()
  const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null)
  const [from, setFrom] = useState<Date | null>(subDays(new Date(), 29))
  const [to, setTo] = useState<Date | null>(new Date())

  const isPreset = PRESETS.includes(value)

  const handleApply = () => {
    if (!from || !to) return
    const d = differenceInDays(startOfDay(to), startOfDay(from)) + 1
    if (d > 0) onChange(d)
    setAnchorEl(null)
  }

  return (
    <Box>
      <ButtonGroup size="small">
        {PRESETS.map((d) => (
          <Button
            key={d}
            variant={value === d && !anchorEl ? 'contained' : 'outlined'}
            onClick={() => {
              setAnchorEl(null)
              onChange(d)
            }}
            sx={{ minWidth: 40, textTransform: 'none', fontSize: 12, px: 1.25 }}
          >
            {d}d
          </Button>
        ))}
        <Button
          variant={!isPreset || !!anchorEl ? 'contained' : 'outlined'}
          onClick={(e) => setAnchorEl(e.currentTarget)}
          sx={{ textTransform: 'none', fontSize: 12, px: 1.25 }}
        >
          {t('timeRange.custom')}
        </Button>
      </ButtonGroup>

      <Popover
        open={!!anchorEl}
        anchorEl={anchorEl}
        onClose={() => setAnchorEl(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'right' }}
        sx={{ mt: 0.5 }}
      >
        <Box sx={{ p: 2.5, minWidth: 300 }}>
          <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 2 }}>
            {t('timeRange.customRange')}
          </Typography>
          <Stack direction="row" spacing={1.5} sx={{ mb: 2 }}>
            <DatePicker
              label={t('timeRange.from')}
              value={from}
              onChange={setFrom}
              maxDate={to ?? new Date()}
              slotProps={{ textField: { size: 'small', fullWidth: true } }}
            />
            <DatePicker
              label={t('timeRange.to')}
              value={to}
              onChange={setTo}
              minDate={from ?? undefined}
              disableFuture
              slotProps={{ textField: { size: 'small', fullWidth: true } }}
            />
          </Stack>
          <Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: 1 }}>
            <Button size="small" onClick={() => setAnchorEl(null)}>
              {t('common.cancel')}
            </Button>
            <Button
              size="small"
              variant="contained"
              onClick={handleApply}
              disabled={!from || !to}
            >
              {t('timeRange.apply')}
            </Button>
          </Box>
        </Box>
      </Popover>
    </Box>
  )
}
