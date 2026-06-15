import { useState, useEffect, useRef } from 'react'
import {
  Box, Card, CardContent, Button, Typography, Chip,
  List, ListItem, ListItemText, ListItemSecondaryAction, Skeleton,
  Alert, LinearProgress,
} from '@mui/material'
import { ArrowUpload24Regular, Document24Regular } from '@fluentui/react-icons'
import { useTranslation } from 'react-i18next'
import { useSnackbar } from 'notistack'
import { useSocket } from '@/shared/hooks/useSocket'
import api from '@/lib/axios'

interface Task {
  name: string
  displayName: string
  running: boolean
  lastRun?: string
}

export default function TasksTab() {
  const { t } = useTranslation()
  const { enqueueSnackbar } = useSnackbar()
  const [tasks, setTasks] = useState<Task[]>([])
  const [logs, setLogs] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const logsRef = useRef<HTMLDivElement>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [importFile, setImportFile] = useState<File | null>(null)
  const [importing, setImporting] = useState(false)
  const [importResult, setImportResult] = useState<{ count?: number; error?: string } | null>(null)

  useEffect(() => {
    api
      .get('/api/getTasks')
      .then((r) => setTasks(r.data ?? []))
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  useSocket('TaskLog', (msg) => {
    setLogs((prev) => [...prev.slice(-500), String(msg)])
    setTimeout(() => {
      if (logsRef.current) logsRef.current.scrollTop = logsRef.current.scrollHeight
    }, 50)
  })

  const handleImport = async () => {
    if (!importFile) return
    setImporting(true)
    setImportResult(null)
    try {
      const form = new FormData()
      form.append('file', importFile)
      const res = await api.post('/sync/importPlaybackBackup', form, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })
      setImportResult({ count: res.data.imported })
      enqueueSnackbar(t('settings.importBackupSuccess', { count: res.data.imported }), { variant: 'success' })
      setImportFile(null)
    } catch (e: unknown) {
      const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error ?? t('settings.importBackupError')
      setImportResult({ error: msg })
      enqueueSnackbar(t('settings.importBackupError'), { variant: 'error' })
    } finally {
      setImporting(false)
    }
  }

  const runTask = async (name: string) => {
    try {
      await api.post(`/api/runTask/${name}`)
      enqueueSnackbar(t('settings.taskStarted'), { variant: 'info' })
    } catch {
      enqueueSnackbar(t('common.error'), { variant: 'error' })
    }
  }

  return (
    <Box>
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="subtitle1" sx={{ fontWeight: 600 }} gutterBottom>{t('settings.scheduledTasks')}</Typography>
          {loading ? (
            Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} variant="rectangular" height={52} sx={{ mb: 1, borderRadius: 1 }} />)
          ) : (
            <List disablePadding>
              {tasks.map((task) => (
                <ListItem key={task.name} disablePadding sx={{ py: 1, borderBottom: '1px solid', borderColor: 'divider', '&:last-child': { borderBottom: 0 } }}>
                  <ListItemText
                    primary={task.displayName}
                    secondary={task.lastRun ? `${t('settings.lastRun')}: ${task.lastRun}` : t('settings.neverRun')}
                    slotProps={{ primary: { style: { fontSize: 14, fontWeight: 500 } }, secondary: { style: { fontSize: 12 } } }}
                  />
                  <ListItemSecondaryAction>
                    {task.running ? (
                      <Chip label={t('settings.running')} size="small" color="primary" />
                    ) : (
                      <Button size="small" variant="outlined" onClick={() => runTask(task.name)}>
                        {t('settings.run')}
                      </Button>
                    )}
                  </ListItemSecondaryAction>
                </ListItem>
              ))}
            </List>
          )}
        </CardContent>
      </Card>

      {/* Import Playback Backup */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 0.5 }}>{t('settings.importBackup')}</Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>{t('settings.importBackupDesc')}</Typography>

          <input
            ref={fileInputRef}
            type="file"
            accept=".tsv,.txt"
            style={{ display: 'none' }}
            onChange={(e) => {
              setImportFile(e.target.files?.[0] ?? null)
              setImportResult(null)
              e.target.value = ''
            }}
          />

          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, flexWrap: 'wrap' }}>
            <Button
              variant="outlined"
              size="small"
              startIcon={<Document24Regular style={{ fontSize: 16 }} />}
              onClick={() => fileInputRef.current?.click()}
              disabled={importing}
            >
              {t('settings.importBackupSelect')}
            </Button>

            {importFile && (
              <Typography variant="body2" color="text.secondary" sx={{ fontSize: 12 }} noWrap>
                {importFile.name} ({(importFile.size / 1024).toFixed(1)} KB)
              </Typography>
            )}

            <Button
              variant="contained"
              size="small"
              startIcon={<ArrowUpload24Regular style={{ fontSize: 16 }} />}
              onClick={handleImport}
              disabled={!importFile || importing}
              sx={{ ml: 'auto' }}
            >
              {t('settings.importBackupRun')}
            </Button>
          </Box>

          {importing && <LinearProgress sx={{ mt: 2, borderRadius: 1 }} />}

          {importResult && !importing && (
            <Alert
              severity={importResult.error ? 'error' : 'success'}
              sx={{ mt: 2, borderRadius: 2 }}
            >
              {importResult.error
                ? importResult.error
                : t('settings.importBackupSuccess', { count: importResult.count })}
            </Alert>
          )}
        </CardContent>
      </Card>

      {/* Terminal output */}
      <Card>
        <CardContent>
          <Typography variant="subtitle1" sx={{ fontWeight: 600 }} gutterBottom>{t('settings.taskOutput')}</Typography>
          <Box sx={{ borderRadius: 1, overflow: 'hidden', border: '1px solid', borderColor: 'divider' }}>
            <Box
              ref={logsRef}
              sx={{
                bgcolor: '#0a0a0f',
                p: 1.5,
                height: 300,
                overflowY: 'auto',
                scrollbarGutter: 'stable',
                fontFamily: 'monospace',
                fontSize: 12,
                color: '#d4d4d8',
              }}
            >
            {logs.length === 0 ? (
              <Typography variant="caption" color="text.secondary">{t('settings.noLogs')}</Typography>
            ) : (
              logs.map((log, i) => (
                <Box key={i} sx={{ mb: 0.25 }}>{log}</Box>
              ))
            )}
          </Box>
          </Box>
        </CardContent>
      </Card>
    </Box>
  )
}
