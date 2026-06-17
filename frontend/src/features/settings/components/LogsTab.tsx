import { useState, useEffect, useCallback, useMemo } from 'react'
import {
  Box, Card, CardContent, Chip, Typography, Alert,
} from '@mui/material'
import { createColumnHelper } from '@tanstack/react-table'
import { format, parseISO } from 'date-fns'
import { useTranslation } from 'react-i18next'
import DataTable, { type FilterDef } from '@/shared/components/DataTable/DataTable'
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
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(() => {
    setLoading(true)
    api.get('/logs/getLogs')
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
            } catch { /* not JSON */ }
            return raw
          })(),
          timestamp: String(row.TimeRun ?? ''),
          task: String(row.Name ?? ''),
        })))
      })
      .catch(() => setError(t('common.loadError')))
      .finally(() => setLoading(false))
  }, [t])

  useEffect(() => { load() }, [load])

  const columns = useMemo(() => [
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
  ], [t])

  const filterDefs = useMemo<FilterDef[]>(() => [
    { id: 'level', label: t('settings.logLevel'), type: 'select' },
    { id: 'task', label: t('settings.task'), type: 'select' },
  ], [t])

  return (
    <Box>
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
      <Card>
        <CardContent>
          <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 2 }}>{t('settings.logs')}</Typography>
          <DataTable
            data={logs}
            columns={columns}
            loading={loading}
            searchPlaceholder={t('settings.searchLogs')}
            filterDefs={filterDefs}
            onRefresh={load}
          />
        </CardContent>
      </Card>
    </Box>
  )
}
