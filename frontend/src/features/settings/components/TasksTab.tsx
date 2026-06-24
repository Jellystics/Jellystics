import { useState, useEffect, useRef } from 'react'
import {
  Box, Card, CardContent, Button, Typography, Chip,
  List, ListItem, ListItemText,
  Alert, LinearProgress, TextField, InputAdornment, Tooltip, Switch, FormControlLabel,
} from '@mui/material'
import SkeletonList from '@/shared/components/SkeletonList/SkeletonList'
import { ArrowUpload24Regular, Document24Regular, ArrowClockwise24Regular, CheckmarkCircle24Regular } from '@fluentui/react-icons'
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

// Full sync is always active. The others require an explicit enable toggle.
const ALWAYS_ENABLED_TASK = 'Full Jellyfin Sync'

// Default cron expressions pre-filled in inputs
const DEFAULT_CRON: Record<string, string> = {
  'Full Jellyfin Sync': '0 0 * * *',
  'Recently Added Sync': '0 * * * *',
  'Backup': '0 3 * * *',
  'Jellyfin Playback Reporting Plugin Sync': '0 4 * * *',
}

export default function TasksTab() {
  const { t } = useTranslation()
  const { enqueueSnackbar } = useSnackbar()
  const [tasks, setTasks] = useState<Task[]>([])
  const [logs, setLogs] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const logsRef = useRef<HTMLDivElement>(null)
  const playbackFileInputRef = useRef<HTMLInputElement>(null)
  const jsonFileInputRef = useRef<HTMLInputElement>(null)
  const [playbackFile, setPlaybackFile] = useState<File | null>(null)
  const [jsonFile, setJsonFile] = useState<File | null>(null)
  const [playbackImporting, setPlaybackImporting] = useState(false)
  const [jsonImporting, setJsonImporting] = useState(false)
  const [playbackImportResult, setPlaybackImportResult] = useState<ImportResult | null>(null)

  // cron expressions keyed by task name
  const [cronInputs, setCronInputs] = useState<Record<string, string>>({})
  const [cronEnabled, setCronEnabled] = useState<Record<string, boolean>>({})
  const [cronSaving, setCronSaving] = useState<Record<string, boolean>>({})

  useEffect(() => {
    api
      .get('/api/getTasks')
      .then((r) => setTasks(r.data ?? []))
      .catch(() => {})
      .finally(() => setLoading(false))

    // Load existing cron settings, falling back to defaults
    api.get('/api/getTaskSettings').then((r) => {
      const settings = r.data as Record<string, { cronExpression?: string; enabled?: boolean; Interval?: number }> ?? {}
      const inputs: Record<string, string> = { ...DEFAULT_CRON }
      const enabled: Record<string, boolean> = {}
      for (const [name, cfg] of Object.entries(settings)) {
        if (cfg.cronExpression) inputs[name] = cfg.cronExpression
        if (cfg.enabled !== undefined) enabled[name] = cfg.enabled
      }
      setCronInputs(inputs)
      setCronEnabled(enabled)
    }).catch(() => { setCronInputs({ ...DEFAULT_CRON }) })
  }, [])

  useSocket('TaskLog', (msg) => {
    setLogs((prev) => [...prev.slice(-500), String(msg)])
    setTimeout(() => {
      if (logsRef.current) logsRef.current.scrollTop = logsRef.current.scrollHeight
    }, 50)
  })

  const saveCron = async (taskName: string) => {
    const expr = (cronInputs[taskName] ?? '').trim()
    if (!expr) return
    setCronSaving((prev) => ({ ...prev, [taskName]: true }))
    try {
      await api.post('/api/setTaskSettings', { taskname: taskName, cronExpression: expr })
      enqueueSnackbar(t('settings.cronSaved'), { variant: 'success' })
    } catch {
      enqueueSnackbar(t('common.error'), { variant: 'error' })
    } finally {
      setCronSaving((prev) => ({ ...prev, [taskName]: false }))
    }
  }

  const toggleCronEnabled = async (taskName: string, value: boolean) => {
    setCronEnabled((prev) => ({ ...prev, [taskName]: value }))
    try {
      await api.post('/api/setTaskSettings', { taskname: taskName, enabled: value })
    } catch {
      // revert on error
      setCronEnabled((prev) => ({ ...prev, [taskName]: !value }))
      enqueueSnackbar(t('common.error'), { variant: 'error' })
    }
  }

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

  const importJsonBackup = async () => {
    if (!jsonFile) return
    setJsonImporting(true)
    try {
      const form = new FormData()
      form.append('file', jsonFile)
      const uploaded = await api.post('/backup/upload', form, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })
      const fileName = uploaded.data.fileName as string
      const res = await api.get(`/backup/restore/${encodeURIComponent(fileName)}`)
      const restored = (res.data.restored ?? {}) as Record<string, number>
      enqueueSnackbar(t('settings.jellystatImportSuccess', { count: Object.values(restored).reduce((s, v) => s + Number(v || 0), 0) }), { variant: 'success' })
      setJsonFile(null)
    } catch (e: unknown) {
      const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error ?? t('settings.jellystatImportError')
      enqueueSnackbar(msg, { variant: 'error' })
    } finally {
      setJsonImporting(false)
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
            <SkeletonList count={4} height={80} />
          ) : (
            <List disablePadding>
              {tasks.map((task) => {
                const isAlwaysOn = task.name === ALWAYS_ENABLED_TASK
                const enabled = isAlwaysOn || !!cronEnabled[task.name]
                return (
                  <ListItem key={task.name} disablePadding sx={{ py: 1.5, borderBottom: '1px solid', borderColor: 'divider', '&:last-child': { borderBottom: 0 }, flexDirection: 'column', alignItems: 'stretch' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', width: '100%' }}>
                      <ListItemText
                        primary={task.displayName}
                        secondary={task.lastRun ? `${t('settings.lastRun')}: ${task.lastRun}` : t('settings.neverRun')}
                        slotProps={{ primary: { style: { fontSize: 14, fontWeight: 500 } }, secondary: { style: { fontSize: 12 } } }}
                      />
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, ml: 'auto', pl: 2 }}>
                        {!isAlwaysOn && (
                          <FormControlLabel
                            control={
                              <Switch
                                size="small"
                                checked={!!cronEnabled[task.name]}
                                onChange={(e) => toggleCronEnabled(task.name, e.target.checked)}
                              />
                            }
                            label={<Typography variant="caption">{t('settings.cronEnabled')}</Typography>}
                            labelPlacement="start"
                            sx={{ mr: 0, ml: 0 }}
                          />
                        )}
                        {task.running ? (
                          <Chip label={t('settings.running')} size="small" color="primary" />
                        ) : (
                          <Button size="small" variant="outlined" onClick={() => runTask(task.name)}>
                            {t('settings.run')}
                          </Button>
                        )}
                      </Box>
                    </Box>

                    {enabled && (
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mt: 0.75 }}>
                        <Tooltip title={t('settings.cronHelp')} placement="bottom-start">
                          <TextField
                            size="small"
                            label={t('settings.cronExpression')}
                            placeholder={DEFAULT_CRON[task.name] ?? '0 * * * *'}
                            value={cronInputs[task.name] ?? ''}
                            onChange={(e) => setCronInputs((prev) => ({ ...prev, [task.name]: e.target.value }))}
                            sx={{ flex: 1, maxWidth: 280 }}
                            slotProps={{
                              htmlInput: { style: { fontFamily: 'monospace', fontSize: 13 } },
                              input: {
                                startAdornment: (
                                  <InputAdornment position="start">
                                    <ArrowClockwise24Regular style={{ fontSize: 16, opacity: 0.5 }} />
                                  </InputAdornment>
                                ),
                              },
                            }}
                          />
                        </Tooltip>
                        <Button
                          size="small"
                          variant="contained"
                          onClick={() => saveCron(task.name)}
                          disabled={!cronInputs[task.name]?.trim() || cronSaving[task.name]}
                          startIcon={<CheckmarkCircle24Regular style={{ fontSize: 14 }} />}
                        >
                          {t('common.save')}
                        </Button>
                      </Box>
                    )}
                  </ListItem>
                )
              })}
            </List>
          )}
          <Typography variant="caption" color="text.secondary" sx={{ mt: 1.5, display: 'block' }}>
            {t('settings.cronHint')}
          </Typography>
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

      {/* JSON backup import (Jellystics / Jellystat) */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 0.5 }}>{t('settings.importJellystatBackup')}</Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>{t('settings.importJellystatBackupDesc')}</Typography>

          <input
            ref={jsonFileInputRef}
            type="file"
            accept=".json,application/json"
            style={{ display: 'none' }}
            onChange={(e) => {
              setJsonFile(e.target.files?.[0] ?? null)
              e.target.value = ''
            }}
          />

          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, flexWrap: 'wrap' }}>
            <Button
              variant="outlined"
              size="small"
              startIcon={<Document24Regular style={{ fontSize: 16 }} />}
              onClick={() => jsonFileInputRef.current?.click()}
              disabled={jsonImporting}
            >
              {t('settings.importBackupSelect')}
            </Button>

            {jsonFile && (
              <Typography variant="body2" color="text.secondary" sx={{ fontSize: 12 }} noWrap>
                {jsonFile.name} ({(jsonFile.size / 1024 / 1024).toFixed(1)} MB)
              </Typography>
            )}

            <Button
              variant="contained"
              size="small"
              startIcon={<ArrowUpload24Regular style={{ fontSize: 16 }} />}
              onClick={importJsonBackup}
              disabled={!jsonFile || jsonImporting}
              sx={{ ml: 'auto' }}
            >
              {t('settings.importBackupRun')}
            </Button>
          </Box>

          {jsonImporting && <LinearProgress sx={{ mt: 2, borderRadius: 1 }} />}
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
