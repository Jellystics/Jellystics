import { useState } from 'react'
import { Box, Button, ButtonGroup, Popover, Stack, Typography } from '@mui/material'
import { DatePicker } from '@mui/x-date-pickers/DatePicker'
import { differenceInDays, startOfDay } from 'date-fns'
import { useTranslation } from 'react-i18next'
import { useDateRange } from '@/lib/dateRange'

const PRESETS = [7, 14, 30, 90]

export default function TimeRangePicker() {
  const { t } = useTranslation()
  const { from, to, setRange, setPreset } = useDateRange()
  const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null)
  const [localFrom, setLocalFrom] = useState<Date | null>(from)
  const [localTo, setLocalTo] = useState<Date | null>(to)

  // Determine which preset is active (if any)
  const activeDays = (() => {
    const d = differenceInDays(startOfDay(to), startOfDay(from))
    return PRESETS.includes(d) ? d : -1
  })()

  const handleOpenCustom = (e: React.MouseEvent<HTMLElement>) => {
    setLocalFrom(from)
    setLocalTo(to)
    setAnchorEl(e.currentTarget)
  }

  const handleApply = () => {
    if (!localFrom || !localTo) return
    const d = differenceInDays(startOfDay(localTo), startOfDay(localFrom))
    if (d > 0) setRange(localFrom, localTo)
    setAnchorEl(null)
  }

  const isCustomActive = anchorEl !== null || activeDays === -1

  return (
    <Box>
      <ButtonGroup size="small">
        {PRESETS.map((d) => (
          <Button
            key={d}
            variant={activeDays === d && !anchorEl ? 'contained' : 'outlined'}
            onClick={() => {
              setAnchorEl(null)
              setPreset(d)
            }}
            sx={{ minWidth: 40, textTransform: 'none', fontSize: 12, px: 1.25 }}
          >
            {d}d
          </Button>
        ))}
        <Button
          variant={isCustomActive ? 'contained' : 'outlined'}
          onClick={handleOpenCustom}
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
              value={localFrom}
              onChange={setLocalFrom}
              maxDate={localTo ?? new Date()}
              slotProps={{ textField: { size: 'small', fullWidth: true } }}
            />
            <DatePicker
              label={t('timeRange.to')}
              value={localTo}
              onChange={setLocalTo}
              minDate={localFrom ?? undefined}
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
              disabled={!localFrom || !localTo}
            >
              {t('timeRange.apply')}
            </Button>
          </Box>
        </Box>
      </Popover>
    </Box>
  )
}
