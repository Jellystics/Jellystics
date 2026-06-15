import { useState } from 'react'
import {
  Box, Button, TextField, Typography,
  CircularProgress, Alert, Paper, Divider,
} from '@mui/material'
import AuthLogo from './AuthLogo'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useTranslation } from 'react-i18next'
import publicApi from '@/lib/publicApi'

const schema = z.object({
  jellyfinUrl: z.string().min(1),
  jellyfinApiKey: z.string().min(1),
})
type FormData = z.infer<typeof schema>

export default function SetupPage() {
  const { t } = useTranslation()
  const [error, setError] = useState<string | null>(null)

  const { register, handleSubmit, formState: { errors, isSubmitting } } = useForm<FormData>({
    resolver: zodResolver(schema),
  })

  const onSubmit = async (data: FormData) => {
    try {
      setError(null)
      await publicApi.post('/auth/configSetup', {
        JF_HOST: data.jellyfinUrl,
        JF_API_KEY: data.jellyfinApiKey,
      })
      window.location.href = '/login'
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { errorMessage?: string } } })?.response?.data?.errorMessage
      setError(msg ?? t('setup.configError'))
    }
  }

  return (
    <Box
      sx={{
        minHeight: '100vh',
        bgcolor: 'background.default',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        p: 2,
      }}
    >
      <Box sx={{ width: '100%', maxWidth: 480 }}>
        <Paper
          sx={{
            width: '100%',
            pt: 2,
            px: 3,
            pb: 3,
            boxShadow: 'none',
            border: '1px solid',
            borderColor: 'divider',
            borderRadius: 1,
          }}
        >
          <AuthLogo />
          <Divider sx={{ mb: 3 }} />

          <Typography variant="h6" sx={{ fontWeight: 700 }} gutterBottom>
            {t('setup.title')}
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2.5 }}>
            {t('setup.description')}
          </Typography>

          {error && (
            <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
              {error}
            </Alert>
          )}

          <Box
            component="form"
            onSubmit={handleSubmit(onSubmit)}
            noValidate
            sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}
          >
            <TextField
              {...register('jellyfinUrl')}
              label={t('setup.jellyfinUrl')}
              fullWidth
              placeholder={t('placeholder.jellyfinUrl')}
              error={!!errors.jellyfinUrl}
              helperText={errors.jellyfinUrl?.type === 'too_small' ? t('validation.required') : errors.jellyfinUrl?.message ?? t('setup.jellyfinUrlHint')}
              autoFocus
              disabled={isSubmitting}
            />
            <TextField
              {...register('jellyfinApiKey')}
              label={t('setup.apiKey')}
              fullWidth
              error={!!errors.jellyfinApiKey}
              helperText={errors.jellyfinApiKey?.type === 'too_small' ? t('validation.required') : errors.jellyfinApiKey?.message ?? t('setup.apiKeyHint')}
              disabled={isSubmitting}
            />
            <Button
              type="submit"
              variant="contained"
              size="large"
              fullWidth
              disableElevation
              disabled={isSubmitting}
              sx={{ mt: 0.5 }}
            >
              {isSubmitting ? <CircularProgress size={22} color="inherit" /> : t('setup.connect')}
            </Button>
          </Box>
        </Paper>

        <Box sx={{ mt: 2, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 0.75 }}>
          <Typography variant="caption" color="text.disabled" sx={{ fontWeight: 500 }}>
            {t('frame.poweredBy')}
          </Typography>
          <Box
            component="a"
            href="https://github.com/Jellystics/Jellystics"
            target="_blank"
            rel="noopener noreferrer"
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: 0.5,
              opacity: 0.4,
              transition: 'opacity 0.2s',
              textDecoration: 'none',
              '&:hover': { opacity: 1 },
              '& img': { filter: 'grayscale(100%) brightness(2)', transition: 'filter 0.2s' },
              '&:hover img': { filter: 'grayscale(100%) brightness(2)' },
            }}
          >
            <Box component="img" src="/logo.svg" alt="Jellystics" sx={{ width: 14, height: 14 }} />
            <Typography variant="caption" sx={{ fontWeight: 700, color: 'text.primary' }}>
              Jellystics
            </Typography>
          </Box>
        </Box>
      </Box>
    </Box>
  )
}
