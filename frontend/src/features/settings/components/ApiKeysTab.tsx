import { useState, useEffect, useMemo } from 'react'
import {
  Box, Card, CardContent, Button, Typography, IconButton,
  Menu, MenuItem, ListItemIcon, ListItemText,
  TextField, Dialog, DialogTitle, DialogContent, DialogActions,
} from '@mui/material'
import { Delete24Regular, Add24Regular, Copy24Regular, MoreVertical24Regular } from '@fluentui/react-icons'
import { createColumnHelper } from '@tanstack/react-table'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useTranslation } from 'react-i18next'
import { useSnackbar } from 'notistack'
import ConfirmDialog from '@/shared/components/ConfirmDialog/ConfirmDialog'
import DataTable, { type FilterDef } from '@/shared/components/DataTable/DataTable'
import api from '@/lib/axios'
import { formatDateTime } from '@/shared/utils/formatDate'

interface ApiKey {
  name: string
  key: string
  createdAt?: string
}

const schema = z.object({ name: z.string().min(1) })
type FormData = z.infer<typeof schema>

const col = createColumnHelper<ApiKey>()

function normalizeApiKeys(value: unknown): ApiKey[] {
  if (Array.isArray(value)) {
    return value.filter((item): item is ApiKey =>
      typeof item === 'object' && item !== null &&
      typeof (item as ApiKey).name === 'string' &&
      typeof (item as ApiKey).key === 'string'
    )
  }
  if (typeof value === 'object' && value !== null && Array.isArray((value as { keys?: unknown }).keys)) {
    return normalizeApiKeys((value as { keys: unknown }).keys)
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
  const [menuAnchor, setMenuAnchor] = useState<{ el: HTMLElement; key: ApiKey } | null>(null)

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

  const columns = useMemo(() => [
    col.accessor('name', {
      header: t('settings.keyName'),
      cell: (i) => <Typography variant="body2" sx={{ fontWeight: 500, fontSize: 13 }}>{i.getValue()}</Typography>,
    }),
    col.accessor('createdAt', {
      header: t('activity.date'),
      cell: (i) => {
        return formatDateTime(i.getValue())
      },
    }),
    col.accessor('key', {
      header: t('setup.apiKey'),
      enableGlobalFilter: false,
      cell: (i) => (
        <Typography variant="caption" sx={{ fontFamily: 'monospace', color: 'text.secondary' }}>
          {i.getValue().slice(0, 12)}••••••••
        </Typography>
      ),
    }),
    col.display({
      id: 'actions',
      header: () => <Box sx={{ textAlign: 'center' }}>{t('common.actions')}</Box>,
      cell: (i) => (
        <Box sx={{ display: 'flex', justifyContent: 'center' }}>
          <IconButton size="small" onClick={(e) => setMenuAnchor({ el: e.currentTarget, key: i.row.original })}>
            <MoreVertical24Regular style={{ fontSize: 18 }} />
          </IconButton>
        </Box>
      ),
    }),
  ], [t])

  const filterDefs = useMemo<FilterDef[]>(() => [
    { id: 'name', label: t('settings.keyName'), type: 'select' },
  ], [t])

  return (
    <Box>
      <Card>
        <CardContent>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
            <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>{t('settings.apiKeys')}</Typography>
            <Button variant="contained" size="small" startIcon={<Add24Regular />} onClick={() => setDialogOpen(true)}>
              {t('settings.createApiKey')}
            </Button>
          </Box>

          <DataTable
            data={keys}
            columns={columns}
            loading={loading}
            searchPlaceholder={`${t('settings.keyName')}, ${t('activity.date')}`}
            onRefresh={load}
            filterDefs={filterDefs}
          />

          <Menu
            anchorEl={menuAnchor?.el}
            open={Boolean(menuAnchor)}
            onClose={() => setMenuAnchor(null)}
            anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
            transformOrigin={{ vertical: 'top', horizontal: 'right' }}
          >
            <MenuItem onClick={() => {
              if (menuAnchor) navigator.clipboard.writeText(menuAnchor.key.key)
              enqueueSnackbar(t('common.copied'), { variant: 'success' })
              setMenuAnchor(null)
            }}>
              <ListItemIcon><Copy24Regular style={{ fontSize: 18 }} /></ListItemIcon>
              <ListItemText>{t('settings.copyKey')}</ListItemText>
            </MenuItem>
            <MenuItem
              onClick={() => { setDeleteTarget(menuAnchor?.key.key ?? null); setMenuAnchor(null) }}
              sx={{ color: 'error.main', '& .MuiListItemIcon-root': { color: 'error.main' } }}
            >
              <ListItemIcon><Delete24Regular style={{ fontSize: 18 }} /></ListItemIcon>
              <ListItemText>{t('common.delete')}</ListItemText>
            </MenuItem>
          </Menu>
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
