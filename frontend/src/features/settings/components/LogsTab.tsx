import { useState, useEffect } from 'react'
import {
  Box, Chip, Select, MenuItem, FormControl, InputLabel,
  Typography, Alert,
} from '@mui/material'
import { createColumnHelper } from '@tanstack/react-table'
import { format, parseISO } from 'date-fns'
import { useTranslation } from 'react-i18next'
import DataTable from '@/shared/components/DataTable/DataTable'
import api from '@/lib/axios'
import { getDateLocale } from '@/lib/dateLocale'

interface LogEntry {
  id: number
  level: 'info' | 'warn' | 'error'
  message: string
  timestamp: string
  task?: string
}

const col = createColumnHelper<LogEntry>()

const LEVEL_COLORS = {
  info: 'default',
  warn: 'warning',
  error: 'error',
} as const

export default function LogsTab() {
  const { t } = useTranslation()
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [level, setLevel] = useState<string>('all')
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api
      .get('/logs/getLogs')
      .then((r) => {
        setLogs((r.data ?? []).map((row: Record<string, unknown>, i: number) => ({
          id: i,
          level: String(row.Result ?? 'info').toLowerCase() === 'failed' ? 'error' as const
            : String(row.Result ?? 'info').toLowerCase() === 'running' ? 'warn' as const
            : 'info' as const,
          message: (() => {
            const raw = String(row.Log ?? row.Result ?? '')
            try {
              const parsed = JSON.parse(raw)
              if (Array.isArray(parsed)) {
                return parsed.map((e: { Message?: string }) => e.Message ?? '').filter(Boolean).join(' | ')
              }
            } catch { /* not JSON, use raw */ }
            return raw
          })(),
          timestamp: String(row.TimeRun ?? ''),
          task: String(row.Name ?? ''),
        })))
      })
      .catch(() => setError(t('common.loadError')))
      .finally(() => setLoading(false))
  }, [t])

  const filtered = level === 'all' ? logs : logs.filter((l) => l.level === level)

  const columns = [
    col.accessor('level', {
      header: t('settings.logLevel'),
      cell: (i) => (
        <Chip
          label={i.getValue()}
          size="small"
          color={LEVEL_COLORS[i.getValue()] ?? 'default'}
          sx={{ fontSize: 11, height: 20, textTransform: 'uppercase' }}
        />
      ),
    }),
    col.accessor('timestamp', {
      header: t('activity.date'),
      cell: (i) => {
        try { return format(parseISO(i.getValue()), 'dd/MM/yyyy HH:mm:ss', { locale: getDateLocale() }) } catch { return i.getValue() }
      },
    }),
    col.accessor('task', {
      header: t('settings.task'),
      cell: (i) => i.getValue() ?? '—',
    }),
    col.accessor('message', {
      header: t('settings.logMessage'),
      cell: (i) => (
        <Typography variant="caption" sx={{ fontFamily: 'monospace', color: 'text.primary' }}>
          {i.getValue()}
        </Typography>
      ),
    }),
  ]

  return (
    <Box>
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      <FormControl size="small" sx={{ mb: 2, minWidth: 140 }}>
        <InputLabel>{t('settings.logLevel')}</InputLabel>
        <Select value={level} label={t('settings.logLevel')} onChange={(e) => setLevel(e.target.value)}>
          <MenuItem value="all">{t('common.all')}</MenuItem>
          <MenuItem value="info">{t('logLevel.info')}</MenuItem>
          <MenuItem value="warn">{t('logLevel.warn')}</MenuItem>
          <MenuItem value="error">{t('logLevel.error')}</MenuItem>
        </Select>
      </FormControl>

      <DataTable
        data={filtered}
        columns={columns}
        loading={loading}
        searchPlaceholder={t('settings.searchLogs')}
      />
    </Box>
  )
}
