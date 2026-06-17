import { useState, useEffect } from 'react'
import {
  Box, Card, CardContent, Button, Typography, List, ListItem,
  ListItemText, ListItemSecondaryAction, IconButton, TextField,
  Dialog, DialogTitle, DialogContent, DialogActions, Skeleton, Chip,
} from '@mui/material'
import { Delete24Regular, Add24Regular } from '@fluentui/react-icons'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useTranslation } from 'react-i18next'
import { useSnackbar } from 'notistack'
import ConfirmDialog from '@/shared/components/ConfirmDialog/ConfirmDialog'
import api from '@/lib/axios'

interface Webhook {
  id: number
  name: string
  url: string
  enabled: boolean
}

const schema = z.object({
  name: z.string().min(1),
  url: z.string().url(),
})
type FormData = z.infer<typeof schema>

export default function WebhooksTab() {
  const { t } = useTranslation()
  const { enqueueSnackbar } = useSnackbar()
  const [webhooks, setWebhooks] = useState<Webhook[]>([])
  const [loading, setLoading] = useState(true)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<number | null>(null)

  const { register, handleSubmit, reset, formState: { errors, isSubmitting } } = useForm<FormData>({ resolver: zodResolver(schema) })

  const load = () => {
    api.get('/webhooks/').then((r) => setWebhooks(r.data ?? [])).catch(() => {}).finally(() => setLoading(false))
  }
  useEffect(() => { load() }, [])

  const onAdd = async (data: FormData) => {
    try {
      await api.post('/webhooks/', {
        name: data.name,
        url: data.url,
        enabled: true,
        trigger_type: 'event',
        event_type: 'playback_started',
      })
      enqueueSnackbar(t('settings.webhookAdded'), { variant: 'success' })
      setDialogOpen(false)
      reset()
      load()
    } catch {
      enqueueSnackbar(t('common.error'), { variant: 'error' })
    }
  }

  const onDelete = async (id: number) => {
    try {
      await api.delete(`/webhooks/${id}`)
      enqueueSnackbar(t('settings.webhookDeleted'), { variant: 'success' })
      setWebhooks((prev) => prev.filter((w) => w.id !== id))
    } catch {
      enqueueSnackbar(t('common.error'), { variant: 'error' })
    } finally {
      setDeleteTarget(null)
    }
  }

  return (
    <Box sx={{ maxWidth: 640 }}>
      <Card>
        <CardContent>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
            <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>{t('settings.webhooks')}</Typography>
            <Button variant="contained" size="small" startIcon={<Add24Regular />} onClick={() => setDialogOpen(true)}>
              {t('settings.addWebhook')}
            </Button>
          </Box>

          {loading ? (
            Array.from({ length: 2 }).map((_, i) => <Skeleton key={i} variant="rectangular" height={52} sx={{ mb: 1, borderRadius: 1 }} />)
          ) : webhooks.length === 0 ? (
            <Typography variant="body2" color="text.secondary">{t('settings.noWebhooks')}</Typography>
          ) : (
            <List disablePadding>
              {webhooks.map((w) => (
                <ListItem key={w.id} disablePadding sx={{ py: 1, borderBottom: '1px solid', borderColor: 'divider', '&:last-child': { borderBottom: 0 } }}>
                  <ListItemText
                    primary={<Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>{w.name}<Chip label={w.enabled ? t('common.active') : t('common.disabled')} size="small" color={w.enabled ? 'success' : 'default'} sx={{ height: 18, fontSize: 10 }} /></Box>}
                    secondary={w.url}
                    slotProps={{ primary: { style: { fontSize: 13, fontWeight: 500 } }, secondary: { style: { fontSize: 11 } } }}
                  />
                  <ListItemSecondaryAction>
                    <IconButton size="small" color="error" onClick={() => setDeleteTarget(w.id)}>
                      <Delete24Regular style={{ fontSize: 18 }} />
                    </IconButton>
                  </ListItemSecondaryAction>
                </ListItem>
              ))}
            </List>
          )}
        </CardContent>
      </Card>

      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="xs" fullWidth>
        <DialogTitle>{t('settings.addWebhook')}</DialogTitle>
        <DialogContent>
          <Box component="form" id="webhook-form" onSubmit={handleSubmit(onAdd)} noValidate sx={{ pt: 1 }}>
            <TextField {...register('name')} label={t('settings.webhookName')} fullWidth size="small" error={!!errors.name} helperText={errors.name?.message} sx={{ mb: 2 }} autoFocus />
            <TextField {...register('url')} label={t('common.url')} fullWidth size="small" error={!!errors.url} helperText={errors.url?.message} placeholder="https://..." />
          </Box>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setDialogOpen(false)}>{t('common.cancel')}</Button>
          <Button type="submit" form="webhook-form" variant="contained" disabled={isSubmitting}>{t('common.add')}</Button>
        </DialogActions>
      </Dialog>

      <ConfirmDialog
        open={deleteTarget !== null}
        title={t('settings.deleteWebhook')}
        description={t('settings.deleteWebhookConfirm')}
        onConfirm={() => deleteTarget !== null && onDelete(deleteTarget)}
        onCancel={() => setDeleteTarget(null)}
        dangerous
      />
    </Box>
  )
}
