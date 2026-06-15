import { useState, useEffect } from 'react'
import {
  Box, Button, TextField, Typography, Card, CardContent,
  CircularProgress, FormControl, InputLabel, Select, MenuItem,
} from '@mui/material'
import { useForm, Controller } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useTranslation } from 'react-i18next'
import i18next from 'i18next'
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
  { labelKey: 'colors.violet', value: '#a78bfa' },
  { labelKey: 'colors.blue', value: '#60a5fa' },
  { labelKey: 'colors.cyan', value: '#22d3ee' },
  { labelKey: 'colors.green', value: '#4ade80' },
  { labelKey: 'colors.orange', value: '#fb923c' },
  { labelKey: 'colors.pink', value: '#f472b6' },
]

const languages = [
  { code: 'en', displayName: 'English' },
  { code: 'fr', displayName: 'Français' },
  { code: 'es', displayName: 'Español' },
]

export default function ConfigTab() {
  const { t } = useTranslation()
  const { enqueueSnackbar } = useSnackbar()
  const [accent, setAccent] = useState(getAccentColor())
  const [lang, setLang] = useState(i18next.language)

  const {
    control,
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

  const handleLanguageChange = (code: string) => {
    setLang(code)
    i18next.changeLanguage(code)
  }

  return (
    <Box sx={{ maxWidth: 640 }}>
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="subtitle1" sx={{ fontWeight: 600 }} gutterBottom>{t('settings.jellyfinConfig')}</Typography>
          <Box component="form" onSubmit={handleSubmit(onSubmit)} noValidate>
            <Controller
              name="JellyfinUrl"
              control={control}
              render={({ field }) => (
                <TextField
                  {...field}
                  label={t('setup.jellyfinUrl')}
                  fullWidth size="small"
                  error={!!errors.JellyfinUrl}
                  helperText={errors.JellyfinUrl?.message}
                  sx={{ mb: 2 }}
                />
              )}
            />
            <Controller
              name="ApiKey"
              control={control}
              render={({ field }) => (
                <TextField
                  {...field}
                  label={t('setup.apiKey')}
                  fullWidth size="small"
                  error={!!errors.ApiKey}
                  helperText={errors.ApiKey?.message}
                  sx={{ mb: 2 }}
                />
              )}
            />
            <Controller
              name="SyncIntervalMinutes"
              control={control}
              render={({ field }) => (
                <TextField
                  {...field}
                  label={t('settings.syncInterval')}
                  type="number"
                  fullWidth size="small"
                  error={!!errors.SyncIntervalMinutes}
                  helperText={errors.SyncIntervalMinutes?.message}
                  sx={{ mb: 2 }}
                  onChange={(e) => field.onChange(e.target.value === '' ? '' : Number(e.target.value))}
                />
              )}
            />
            <Controller
              name="KeepLogsForDays"
              control={control}
              render={({ field }) => (
                <TextField
                  {...field}
                  label={t('settings.keepLogsForDays')}
                  type="number"
                  fullWidth size="small"
                  error={!!errors.KeepLogsForDays}
                  helperText={errors.KeepLogsForDays?.message}
                  sx={{ mb: 3 }}
                  onChange={(e) => field.onChange(e.target.value === '' ? '' : Number(e.target.value))}
                />
              )}
            />
            <Button type="submit" variant="contained" disabled={isSubmitting}>
              {isSubmitting ? <CircularProgress size={18} /> : t('common.save')}
            </Button>
          </Box>
        </CardContent>
      </Card>

      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="subtitle1" sx={{ fontWeight: 600 }} gutterBottom>{t('frame.language')}</Typography>
          <FormControl size="small" sx={{ minWidth: 200 }}>
            <InputLabel>{t('frame.language')}</InputLabel>
            <Select value={lang} label={t('frame.language')} onChange={(e) => handleLanguageChange(e.target.value)}>
              {languages.map((l) => (
                <MenuItem key={l.code} value={l.code}>{l.displayName}</MenuItem>
              ))}
            </Select>
          </FormControl>
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
                title={t(c.labelKey)}
              />
            ))}
          </Box>
        </CardContent>
      </Card>
    </Box>
  )
}
