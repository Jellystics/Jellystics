import { useState, useEffect } from 'react'
import {
  Box, Card, CardContent, Button, Typography, List, ListItem,
  ListItemText, ListItemSecondaryAction, IconButton, Skeleton,
} from '@mui/material'
import { Delete24Regular, ArrowDownload24Regular } from '@fluentui/react-icons'
import { useTranslation } from 'react-i18next'
import { useSnackbar } from 'notistack'
import { format, parseISO } from 'date-fns'
import ConfirmDialog from '@/shared/components/ConfirmDialog/ConfirmDialog'
import api from '@/lib/axios'

interface BackupFile {
  name: string
  size: number
  createdAt: string
}

export default function BackupTab() {
  const { t } = useTranslation()
  const { enqueueSnackbar } = useSnackbar()
  const [backups, setBackups] = useState<BackupFile[]>([])
  const [loading, setLoading] = useState(true)
  const [creating, setCreating] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null)

  const loadBackups = () => {
    api.get('/backup/files').then((r) => {
      setBackups((r.data ?? []).map((f: { name: string; size: number; datecreated: string }) => ({
        name: f.name,
        size: f.size,
        createdAt: f.datecreated,
      })))
    }).catch(() => {}).finally(() => setLoading(false))
  }

  useEffect(() => { loadBackups() }, [])

  const createBackup = async () => {
    setCreating(true)
    try {
      await api.get('/backup/beginBackup')
      enqueueSnackbar(t('settings.backupCreated'), { variant: 'success' })
      loadBackups()
    } catch {
      enqueueSnackbar(t('common.error'), { variant: 'error' })
    } finally {
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

  return (
    <Box sx={{ maxWidth: 640 }}>
      <Card>
        <CardContent>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
            <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>{t('settings.backupFiles')}</Typography>
            <Button variant="contained" size="small" onClick={createBackup} disabled={creating}>
              {creating ? t('settings.creating') : t('settings.createBackup')}
            </Button>
          </Box>

          {loading ? (
            Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} variant="rectangular" height={52} sx={{ mb: 1, borderRadius: 1 }} />)
          ) : backups.length === 0 ? (
            <Typography variant="body2" color="text.secondary">{t('settings.noBackups')}</Typography>
          ) : (
            <List disablePadding>
              {backups.map((b) => (
                <ListItem key={b.name} disablePadding sx={{ py: 1, borderBottom: '1px solid', borderColor: 'divider', '&:last-child': { borderBottom: 0 } }}>
                  <ListItemText
                    primary={b.name}
                    secondary={`${format(parseISO(b.createdAt), 'dd/MM/yyyy HH:mm')} · ${(b.size / 1024).toFixed(1)} KB`}
                    slotProps={{ primary: { style: { fontSize: 13 } }, secondary: { style: { fontSize: 11 } } }}
                  />
                  <ListItemSecondaryAction>
                    <IconButton size="small" href={`/backup/files/${encodeURIComponent(b.name)}`} download>
                      <ArrowDownload24Regular style={{ fontSize: 18 }} />
                    </IconButton>
                    <IconButton size="small" color="error" onClick={() => setDeleteTarget(b.name)}>
                      <Delete24Regular style={{ fontSize: 18 }} />
                    </IconButton>
                  </ListItemSecondaryAction>
                </ListItem>
              ))}
            </List>
          )}
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
