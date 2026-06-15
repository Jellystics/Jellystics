import { useState, useEffect } from 'react'
import {
  Box, Card, CardContent, Button, Typography, List, ListItem,
  ListItemText, ListItemSecondaryAction, IconButton, TextField,
  Dialog, DialogTitle, DialogContent, DialogActions, Skeleton,
} from '@mui/material'
import { Delete24Regular, Add24Regular, Copy24Regular } from '@fluentui/react-icons'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useTranslation } from 'react-i18next'
import { useSnackbar } from 'notistack'
import ConfirmDialog from '@/shared/components/ConfirmDialog/ConfirmDialog'
import api from '@/lib/axios'

interface ApiKey {
  name: string
  key: string
}

const schema = z.object({ name: z.string().min(1) })
type FormData = z.infer<typeof schema>

function normalizeApiKeys(value: unknown): ApiKey[] {
  if (Array.isArray(value)) {
    return value.filter((item): item is ApiKey =>
      typeof item === 'object' &&
      item !== null &&
      typeof (item as ApiKey).name === 'string' &&
      typeof (item as ApiKey).key === 'string'
    )
  }

  if (typeof value === 'object' && value !== null && Array.isArray((value as { keys?: unknown }).keys)) {
    return normalizeApiKeys((value as { keys: unknown }).keys)
  }

  if (typeof value === 'object' && value !== null && Array.isArray((value as { api_keys?: unknown }).api_keys)) {
    return normalizeApiKeys((value as { api_keys: unknown }).api_keys)
  }

  return []
}

export default function ApiKeysTab() {
  const { t } = useTranslation()
  const { enqueueSnackbar } = useSnackbar()
  const [keys, setKeys] = useState<ApiKey[]>([])
  const [loading, setLoading] = useState(true)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null)

  const { register, handleSubmit, reset, formState: { errors, isSubmitting } } = useForm<FormData>({ resolver: zodResolver(schema) })

  const load = () => {
    api.get('/api/keys').then((r) => setKeys(normalizeApiKeys(r.data))).catch(() => setKeys([])).finally(() => setLoading(false))
  }
  useEffect(() => { load() }, [])

  const onAdd = async (data: FormData) => {
    try {
      await api.post('/api/keys', data)
      enqueueSnackbar(t('settings.apiKeyCreated'), { variant: 'success' })
      setDialogOpen(false)
      reset()
      load()
    } catch {
      enqueueSnackbar(t('common.error'), { variant: 'error' })
    }
  }

  const onDelete = async (key: string) => {
    try {
      await api.delete('/api/keys', { data: { key } })
      enqueueSnackbar(t('settings.apiKeyDeleted'), { variant: 'success' })
      setKeys((prev) => prev.filter((k) => k.key !== key))
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
            <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>{t('settings.apiKeys')}</Typography>
            <Button variant="contained" size="small" startIcon={<Add24Regular />} onClick={() => setDialogOpen(true)}>
              {t('settings.createApiKey')}
            </Button>
          </Box>

          {loading ? (
            Array.from({ length: 2 }).map((_, i) => <Skeleton key={i} variant="rectangular" height={52} sx={{ mb: 1, borderRadius: 1 }} />)
          ) : keys.length === 0 ? (
            <Typography variant="body2" color="text.secondary">{t('settings.noApiKeys')}</Typography>
          ) : (
            <List disablePadding>
              {keys.map((k) => (
                <ListItem key={k.key} disablePadding sx={{ py: 1, borderBottom: '1px solid', borderColor: 'divider', '&:last-child': { borderBottom: 0 } }}>
                  <ListItemText
                    primary={k.name}
                    secondary={
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                        <Typography variant="caption" sx={{ fontFamily: 'monospace', color: 'text.secondary' }}>
                          {k.key.slice(0, 12)}••••••••
                        </Typography>
                        <IconButton size="small" onClick={() => { navigator.clipboard.writeText(k.key); enqueueSnackbar(t('common.copied'), { variant: 'success' }) }}>
                          <Copy24Regular style={{ fontSize: 14 }} />
                        </IconButton>
                      </Box>
                    }
                    slotProps={{ primary: { style: { fontSize: 13, fontWeight: 500 } } }}
                  />
                  <ListItemSecondaryAction>
                    <IconButton size="small" color="error" onClick={() => setDeleteTarget(k.key)}>
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
        <DialogTitle>{t('settings.createApiKey')}</DialogTitle>
        <DialogContent>
          <Box component="form" id="apikey-form" onSubmit={handleSubmit(onAdd)} noValidate sx={{ pt: 1 }}>
            <TextField {...register('name')} label={t('settings.keyName')} fullWidth size="small" error={!!errors.name} helperText={errors.name?.message} autoFocus />
          </Box>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setDialogOpen(false)}>{t('common.cancel')}</Button>
          <Button type="submit" form="apikey-form" variant="contained" disabled={isSubmitting}>{t('common.create')}</Button>
        </DialogActions>
      </Dialog>

      <ConfirmDialog
        open={deleteTarget !== null}
        title={t('settings.deleteApiKey')}
        description={t('settings.deleteApiKeyConfirm')}
        onConfirm={() => deleteTarget !== null && onDelete(deleteTarget)}
        onCancel={() => setDeleteTarget(null)}
        dangerous
      />
    </Box>
  )
}
