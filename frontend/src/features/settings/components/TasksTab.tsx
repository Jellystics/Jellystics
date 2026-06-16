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

interface ImportResult {
  count?: number
  error?: string
}

export default function TasksTab() {
  const { t } = useTranslation()
  const { enqueueSnackbar } = useSnackbar()
  const [tasks, setTasks] = useState<Task[]>([])
  const [logs, setLogs] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const logsRef = useRef<HTMLDivElement>(null)
  const playbackFileInputRef = useRef<HTMLInputElement>(null)
  const jellystatFileInputRef = useRef<HTMLInputElement>(null)
  const [playbackFile, setPlaybackFile] = useState<File | null>(null)
  const [jellystatFile, setJellystatFile] = useState<File | null>(null)
  const [playbackImporting, setPlaybackImporting] = useState(false)
  const [jellystatImporting, setJellystatImporting] = useState(false)
  const [playbackImportResult, setPlaybackImportResult] = useState<ImportResult | null>(null)
  const [jellystatImportResult, setJellystatImportResult] = useState<ImportResult | null>(null)

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

  const importPlaybackReportingBackup = async () => {
    if (!playbackFile) return
    setPlaybackImporting(true)
    setPlaybackImportResult(null)
    try {
      const form = new FormData()
      form.append('file', playbackFile)
      const res = await api.post('/sync/importPlaybackBackup', form, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })
      const count = res.data.imported
      setPlaybackImportResult({ count })
      enqueueSnackbar(t('settings.importBackupSuccess', { count }), { variant: 'success' })
      setPlaybackFile(null)
    } catch (e: unknown) {
      const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error ?? t('settings.importBackupError')
      setPlaybackImportResult({ error: msg })
      enqueueSnackbar(t('settings.importBackupError'), { variant: 'error' })
    } finally {
      setPlaybackImporting(false)
    }
  }

  const importJellystatBackup = async () => {
    if (!jellystatFile) return
    setJellystatImporting(true)
    setJellystatImportResult(null)
    try {
      const form = new FormData()
      form.append('file', jellystatFile)
      const uploaded = await api.post('/backup/upload', form, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })
      const fileName = uploaded.data.fileName as string
      const restoredResponse = await api.get(`/backup/restore/${encodeURIComponent(fileName)}`)
      const restored = (restoredResponse.data.restored ?? {}) as Record<string, number>
      const count = Object.values(restored).reduce((sum, value) => sum + Number(value || 0), 0)

      setJellystatImportResult({ count })
      enqueueSnackbar(t('settings.jellystatImportSuccess', { count }), { variant: 'success' })
      setJellystatFile(null)
    } catch (e: unknown) {
      const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error ?? t('settings.jellystatImportError')
      setJellystatImportResult({ error: msg })
      enqueueSnackbar(t('settings.jellystatImportError'), { variant: 'error' })
    } finally {
      setJellystatImporting(false)
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

      {/* Jellyfin Playback Reporting Plugin import */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 0.5 }}>{t('settings.importPlaybackReportingBackup')}</Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>{t('settings.importPlaybackReportingBackupDesc')}</Typography>

          <input
            ref={playbackFileInputRef}
            type="file"
            accept=".tsv,.txt,text/tab-separated-values,text/plain"
            style={{ display: 'none' }}
            onChange={(e) => {
              setPlaybackFile(e.target.files?.[0] ?? null)
              setPlaybackImportResult(null)
              e.target.value = ''
            }}
          />

          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, flexWrap: 'wrap' }}>
            <Button
              variant="outlined"
              size="small"
              startIcon={<Document24Regular style={{ fontSize: 16 }} />}
              onClick={() => playbackFileInputRef.current?.click()}
              disabled={playbackImporting}
            >
              {t('settings.importBackupSelect')}
            </Button>

            {playbackFile && (
              <Typography variant="body2" color="text.secondary" sx={{ fontSize: 12 }} noWrap>
                {playbackFile.name} ({(playbackFile.size / 1024).toFixed(1)} KB)
              </Typography>
            )}

            <Button
              variant="contained"
              size="small"
              startIcon={<ArrowUpload24Regular style={{ fontSize: 16 }} />}
              onClick={importPlaybackReportingBackup}
              disabled={!playbackFile || playbackImporting}
              sx={{ ml: 'auto' }}
            >
              {t('settings.importBackupRun')}
            </Button>
          </Box>

          {playbackImporting && <LinearProgress sx={{ mt: 2, borderRadius: 1 }} />}

          {playbackImportResult && !playbackImporting && (
            <Alert
              severity={playbackImportResult.error ? 'error' : 'success'}
              sx={{ mt: 2, borderRadius: 2 }}
            >
              {playbackImportResult.error
                ? playbackImportResult.error
                : t('settings.importBackupSuccess', { count: playbackImportResult.count })}
            </Alert>
          )}
        </CardContent>
      </Card>

      {/* Jellystat import */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 0.5 }}>{t('settings.importJellystatBackup')}</Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>{t('settings.importJellystatBackupDesc')}</Typography>

          <input
            ref={jellystatFileInputRef}
            type="file"
            accept=".json,application/json"
            style={{ display: 'none' }}
            onChange={(e) => {
              setJellystatFile(e.target.files?.[0] ?? null)
              setJellystatImportResult(null)
              e.target.value = ''
            }}
          />

          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, flexWrap: 'wrap' }}>
            <Button
              variant="outlined"
              size="small"
              startIcon={<Document24Regular style={{ fontSize: 16 }} />}
              onClick={() => jellystatFileInputRef.current?.click()}
              disabled={jellystatImporting}
            >
              {t('settings.importBackupSelect')}
            </Button>

            {jellystatFile && (
              <Typography variant="body2" color="text.secondary" sx={{ fontSize: 12 }} noWrap>
                {jellystatFile.name} ({(jellystatFile.size / 1024 / 1024).toFixed(1)} MB)
              </Typography>
            )}

            <Button
              variant="contained"
              size="small"
              startIcon={<ArrowUpload24Regular style={{ fontSize: 16 }} />}
              onClick={importJellystatBackup}
              disabled={!jellystatFile || jellystatImporting}
              sx={{ ml: 'auto' }}
            >
              {t('settings.importBackupRun')}
            </Button>
          </Box>

          {jellystatImporting && <LinearProgress sx={{ mt: 2, borderRadius: 1 }} />}

          {jellystatImportResult && !jellystatImporting && (
            <Alert
              severity={jellystatImportResult.error ? 'error' : 'success'}
              sx={{ mt: 2, borderRadius: 2 }}
            >
              {jellystatImportResult.error
                ? jellystatImportResult.error
                : t('settings.jellystatImportSuccess', { count: jellystatImportResult.count })}
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
