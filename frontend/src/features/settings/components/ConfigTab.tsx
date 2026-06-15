import { useState, useEffect } from 'react'
import {
  Box, Button, TextField, Typography, Card, CardContent,
  CircularProgress,
} from '@mui/material'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useTranslation } from 'react-i18next'
import { useSnackbar } from 'notistack'
import api from '@/lib/axios'
import { getAccentColor, setAccentColor } from '@/lib/theme'

const schema = z.object({
  JellyfinUrl: z.string().url(),
  ApiKey: z.string().min(1),
  SyncIntervalMinutes: z.number().min(1),
  KeepLogsForDays: z.number().min(1),
})

type FormData = z.infer<typeof schema>

const ACCENT_COLORS = [
  { label: 'Violet (default)', value: '#a78bfa' },
  { label: 'Blue', value: '#60a5fa' },
  { label: 'Cyan', value: '#22d3ee' },
  { label: 'Green', value: '#4ade80' },
  { label: 'Orange', value: '#fb923c' },
  { label: 'Pink', value: '#f472b6' },
]

export default function ConfigTab() {
  const { t } = useTranslation()
  const { enqueueSnackbar } = useSnackbar()
  const [accent, setAccent] = useState(getAccentColor())

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<FormData>({ resolver: zodResolver(schema) })

  useEffect(() => {
    api.get('/api/getconfig').then((r) => {
      const data = r.data
      reset({
        JellyfinUrl: data.JF_HOST ?? '',
        ApiKey: '',
        SyncIntervalMinutes: data.settings?.Tasks?.JellyfinSync?.Interval ?? 60,
        KeepLogsForDays: data.settings?.KeepLogsForDays ?? 30,
      })
    }).catch(() => {})
  }, [reset])

  const onSubmit = async (data: FormData) => {
    try {
      await api.post('/api/setconfig', { JF_HOST: data.JellyfinUrl, JF_API_KEY: data.ApiKey })
      enqueueSnackbar(t('settings.configSaved'), { variant: 'success' })
    } catch {
      enqueueSnackbar(t('common.saveError'), { variant: 'error' })
    }
  }

  const handleAccentChange = (color: string) => {
    setAccent(color)
    setAccentColor(color)
    window.location.reload()
  }

  return (
    <Box sx={{ maxWidth: 640 }}>
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="subtitle1" sx={{ fontWeight: 600 }} gutterBottom>{t('settings.jellyfinConfig')}</Typography>
          <Box component="form" onSubmit={handleSubmit(onSubmit)} noValidate>
            <TextField
              {...register('JellyfinUrl')}
              label={t('setup.jellyfinUrl')}
              fullWidth size="small"
              error={!!errors.JellyfinUrl}
              helperText={errors.JellyfinUrl?.message}
              sx={{ mb: 2 }}
            />
            <TextField
              {...register('ApiKey')}
              label={t('setup.apiKey')}
              fullWidth size="small"
              error={!!errors.ApiKey}
              helperText={errors.ApiKey?.message}
              sx={{ mb: 2 }}
            />
            <TextField
              {...register('SyncIntervalMinutes', { valueAsNumber: true })}
              label={t('settings.syncInterval')}
              type="number"
              fullWidth size="small"
              error={!!errors.SyncIntervalMinutes}
              helperText={errors.SyncIntervalMinutes?.message}
              sx={{ mb: 2 }}
            />
            <TextField
              {...register('KeepLogsForDays', { valueAsNumber: true })}
              label={t('settings.keepLogsForDays')}
              type="number"
              fullWidth size="small"
              error={!!errors.KeepLogsForDays}
              helperText={errors.KeepLogsForDays?.message}
              sx={{ mb: 3 }}
            />
            <Button type="submit" variant="contained" disabled={isSubmitting}>
              {isSubmitting ? <CircularProgress size={18} /> : t('common.save')}
            </Button>
          </Box>
        </CardContent>
      </Card>

      <Card>
        <CardContent>
          <Typography variant="subtitle1" sx={{ fontWeight: 600 }} gutterBottom>{t('settings.accentColor')}</Typography>
          <Box sx={{ display: 'flex', gap: 1.5, flexWrap: 'wrap', mt: 1 }}>
            {ACCENT_COLORS.map((c) => (
              <Box
                key={c.value}
                onClick={() => handleAccentChange(c.value)}
                sx={{
                  width: 32,
                  height: 32,
                  borderRadius: '50%',
                  bgcolor: c.value,
                  cursor: 'pointer',
                  border: accent === c.value ? '3px solid white' : '3px solid transparent',
                  transition: 'transform 0.1s',
                  '&:hover': { transform: 'scale(1.15)' },
                }}
                title={c.label}
              />
            ))}
          </Box>
        </CardContent>
      </Card>
    </Box>
  )
}
