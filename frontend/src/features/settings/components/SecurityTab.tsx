import {
  Box, Card, CardContent, Button, TextField, Typography,
  CircularProgress,
} from '@mui/material'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useTranslation } from 'react-i18next'
import { useSnackbar } from 'notistack'
import api from '@/lib/axios'

const pwSchema = z.object({
  currentPassword: z.string().min(1),
  newPassword: z.string().min(8, 'Min 8 characters'),
  confirmPassword: z.string(),
}).refine((d) => d.newPassword === d.confirmPassword, {
  message: 'Passwords do not match',
  path: ['confirmPassword'],
})

type PwFormData = z.infer<typeof pwSchema>

export default function SecurityTab() {
  const { t } = useTranslation()
  const { enqueueSnackbar } = useSnackbar()

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<PwFormData>({ resolver: zodResolver(pwSchema) })

  const onSubmit = async (data: PwFormData) => {
    try {
      await api.post('/api/updatePassword', {
        current_password: data.currentPassword,
        new_password: data.newPassword,
      })
      enqueueSnackbar(t('settings.passwordChanged'), { variant: 'success' })
      reset()
    } catch {
      enqueueSnackbar(t('settings.passwordError'), { variant: 'error' })
    }
  }

  return (
    <Box sx={{ maxWidth: 480 }}>
      <Card>
        <CardContent>
          <Typography variant="subtitle1" sx={{ fontWeight: 600 }} gutterBottom>
            {t('settings.changePassword')}
          </Typography>
          <Box component="form" onSubmit={handleSubmit(onSubmit)} noValidate>
            <TextField
              {...register('currentPassword')}
              label={t('settings.currentPassword')}
              type="password" fullWidth size="small"
              error={!!errors.currentPassword}
              helperText={errors.currentPassword?.message}
              sx={{ mb: 2 }}
            />
            <TextField
              {...register('newPassword')}
              label={t('settings.newPassword')}
              type="password" fullWidth size="small"
              error={!!errors.newPassword}
              helperText={errors.newPassword?.message}
              sx={{ mb: 2 }}
            />
            <TextField
              {...register('confirmPassword')}
              label={t('auth.confirmPassword')}
              type="password" fullWidth size="small"
              error={!!errors.confirmPassword}
              helperText={errors.confirmPassword?.message}
              sx={{ mb: 3 }}
            />
            <Button type="submit" variant="contained" disabled={isSubmitting}>
              {isSubmitting ? <CircularProgress size={18} /> : t('settings.updatePassword')}
            </Button>
          </Box>
        </CardContent>
      </Card>
    </Box>
  )
}
