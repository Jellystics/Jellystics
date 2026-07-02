import { useState, useEffect, useMemo, useRef } from 'react'
import {
  Box, Card, CardContent, Button, Typography, TextField, IconButton, Menu, MenuItem, ListItemIcon, ListItemText,
} from '@mui/material'
import { Delete24Regular, ArrowDownload24Regular, MoreVertical24Regular } from '@fluentui/react-icons'
import { createColumnHelper } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'
import { useSnackbar } from 'notistack'
import ConfirmDialog from '@/shared/components/ConfirmDialog/ConfirmDialog'
import DataTable from '@/shared/components/DataTable/DataTable'
import api from '@/lib/axios'
import { formatDateTime } from '@/shared/utils/formatDate'

interface BackupFile {
  name: string
  size: number
  createdAt: string
}

const col = createColumnHelper<BackupFile>()

export default function BackupTab() {
  const { t } = useTranslation()
  const { enqueueSnackbar } = useSnackbar()
  const [backups, setBackups] = useState<BackupFile[]>([])
  const [loading, setLoading] = useState(true)
  const [creating, setCreating] = useState(false)
  const creatingRef = useRef(false)
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null)
  const [keepDays, setKeepDays] = useState<number>(30)
  const [keepDaysSaving, setKeepDaysSaving] = useState(false)

  const loadBackups = () => {
    api.get('/backup/files').then((r) => {
      setBackups((r.data ?? []).map((f: { name: string; size: number; datecreated: string }) => ({
        name: f.name,
        size: f.size,
        createdAt: f.datecreated,
      })))
    }).catch(() => {}).finally(() => setLoading(false))
  }

  useEffect(() => {
    loadBackups()
    api.get('/api/getconfig').then((r) => {
      setKeepDays(r.data?.settings?.KeepLogsForDays ?? 30)
    }).catch(() => {})
  }, [])

  const createBackup = async () => {
    if (creatingRef.current) return
    creatingRef.current = true
    setCreating(true)
    try {
      await api.get('/backup/beginBackup')
      enqueueSnackbar(t('settings.backupCreated'), { variant: 'success' })
      loadBackups()
    } catch {
      enqueueSnackbar(t('common.error'), { variant: 'error' })
    } finally {
      creatingRef.current = false
      setCreating(false)
    }
  }

  const deleteBackup = async (name: string) => {
    try {
      await api.delete(`/backup/files/${encodeURIComponent(name)}`)
      enqueueSnackbar(t('settings.backupDeleted'), { variant: 'success' })
      setBackups((prev) => prev.filter((b) => b.name !== name))
    } catch {
      enqueueSnackbar(t('common.error'), { variant: 'error' })
    } finally {
      setDeleteTarget(null)
    }
  }

  const saveKeepDays = async () => {
    setKeepDaysSaving(true)
    try {
      await api.post('/api/setconfig', { KeepLogsForDays: keepDays })
      enqueueSnackbar(t('common.saved'), { variant: 'success' })
    } catch {
      enqueueSnackbar(t('common.error'), { variant: 'error' })
    } finally {
      setKeepDaysSaving(false)
    }
  }

  const [menuAnchor, setMenuAnchor] = useState<{ el: HTMLElement; name: string } | null>(null)

  const columns = useMemo(() => [
    col.accessor('name', {
      header: t('settings.backupName'),
      enableGlobalFilter: false,
      cell: (i) => (
        <Typography variant="body2" sx={{ fontFamily: 'monospace', fontSize: 12 }}>{i.getValue()}</Typography>
      ),
    }),
    col.accessor('createdAt', {
      header: t('activity.date'),
      cell: (i) => {
        return formatDateTime(i.getValue())
      },
    }),
    col.accessor('size', {
      header: t('settings.backupSize'),
      enableGlobalFilter: false,
      cell: (i) => `${(i.getValue() / 1024).toFixed(1)} ${t('units.kilobytes')}`,
    }),
    col.display({
      id: 'actions',
      header: () => <Box sx={{ textAlign: 'center' }}>{t('common.actions')}</Box>,
      cell: (i) => (
        <Box sx={{ display: 'flex', justifyContent: 'center' }}>
          <IconButton size="small" onClick={(e) => setMenuAnchor({ el: e.currentTarget, name: i.row.original.name })}>
            <MoreVertical24Regular style={{ fontSize: 18 }} />
          </IconButton>
        </Box>
      ),
    }),
  ], [t])

  return (
    <Box>
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
            <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>{t('settings.backupFiles')}</Typography>
            <Button variant="contained" size="small" onClick={createBackup} disabled={creating}>
              {creating ? t('settings.creating') : t('settings.createBackup')}
            </Button>
          </Box>
          <DataTable
            data={backups}
            columns={columns}
            loading={loading}
            searchPlaceholder={t('activity.date')}
            onRefresh={loadBackups}
          />
          <Menu
            anchorEl={menuAnchor?.el}
            open={Boolean(menuAnchor)}
            onClose={() => setMenuAnchor(null)}
            anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
            transformOrigin={{ vertical: 'top', horizontal: 'right' }}
          >
            <MenuItem onClick={() => {
              const a = document.createElement('a')
              a.href = `/backup/files/${encodeURIComponent(menuAnchor?.name ?? '')}`
              a.download = menuAnchor?.name ?? ''
              document.body.appendChild(a)
              a.click()
              document.body.removeChild(a)
              setMenuAnchor(null)
            }}>
              <ListItemIcon><ArrowDownload24Regular style={{ fontSize: 18 }} /></ListItemIcon>
              <ListItemText>{t('common.download')}</ListItemText>
            </MenuItem>
            <MenuItem onClick={() => { setDeleteTarget(menuAnchor?.name ?? null); setMenuAnchor(null) }} sx={{ color: 'error.main', '& .MuiListItemIcon-root': { color: 'error.main' } }}>
              <ListItemIcon><Delete24Regular style={{ fontSize: 18 }} /></ListItemIcon>
              <ListItemText>{t('common.delete')}</ListItemText>
            </MenuItem>
          </Menu>
        </CardContent>
      </Card>

      <Card>
        <CardContent>
          <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 0.5 }}>{t('settings.keepLogsForDays')}</Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>{t('settings.keepLogsForDaysDesc')}</Typography>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
            <TextField
              type="number"
              size="small"
              value={keepDays}
              onChange={(e) => setKeepDays(Math.max(1, Number(e.target.value)))}
              sx={{ width: 120 }}
              slotProps={{ htmlInput: { min: 1 } }}
            />
            <Button variant="contained" size="small" onClick={saveKeepDays} disabled={keepDaysSaving}>
              {t('common.save')}
            </Button>
          </Box>
        </CardContent>
      </Card>

      <ConfirmDialog
        open={deleteTarget !== null}
        title={t('settings.deleteBackup')}
        description={t('settings.deleteBackupConfirm', { name: deleteTarget ?? '' })}
        onConfirm={() => deleteTarget && deleteBackup(deleteTarget)}
        onCancel={() => setDeleteTarget(null)}
        dangerous
      />
    </Box>
  )
}
