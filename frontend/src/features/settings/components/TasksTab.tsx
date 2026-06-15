import { useState, useEffect, useRef } from 'react'
import {
  Box, Card, CardContent, Button, Typography, Chip,
  List, ListItem, ListItemText, ListItemSecondaryAction, Skeleton,
} from '@mui/material'
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

      {/* Terminal output */}
      <Card>
        <CardContent>
          <Typography variant="subtitle1" sx={{ fontWeight: 600 }} gutterBottom>{t('settings.taskOutput')}</Typography>
          <Box
            ref={logsRef}
            sx={{
              bgcolor: '#0a0a0f',
              borderRadius: 1,
              p: 1.5,
              height: 300,
              overflowY: 'auto',
              fontFamily: 'monospace',
              fontSize: 12,
              color: '#d4d4d8',
              border: '1px solid',
              borderColor: 'divider',
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
        </CardContent>
      </Card>
    </Box>
  )
}
