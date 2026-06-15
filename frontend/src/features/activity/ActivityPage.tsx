import { useState, useEffect } from 'react'
import { Alert } from '@mui/material'
import { createColumnHelper } from '@tanstack/react-table'
import type { ColumnDef } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'
import { format, parseISO } from 'date-fns'
import PageHeader from '@/shared/components/PageHeader/PageHeader'
import DataTable from '@/shared/components/DataTable/DataTable'
import api from '@/lib/axios'
import type { Activity } from '@/shared/types/activity'

const col = createColumnHelper<Activity>()

export default function ActivityPage() {
  const { t } = useTranslation()
  const [data, setData] = useState<Activity[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const load = (showLoading = true) => {
      if (showLoading) setLoading(true)
      api
        .get('/stats/getAllUserActivity')
        .then((r) => setData(r.data ?? []))
        .catch(() => setError(t('common.loadError')))
        .finally(() => setLoading(false))
    }

    load()
    const interval = window.setInterval(() => load(false), 15000)
    return () => window.clearInterval(interval)
  }, [t])

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const columns: ColumnDef<Activity, any>[] = [
    col.accessor('UserName', { header: t('activity.user') }),
    col.accessor('NowPlayingItemName', {
      header: t('activity.item'),
      cell: (i) => {
        const row = i.row.original
        return row.SeriesName ? `${row.SeriesName} — ${i.getValue()}` : i.getValue()
      },
    }),
    col.accessor('Client', { header: t('activity.client') }),
    col.accessor('DeviceName', { header: t('activity.device') }),
    col.accessor('PlayMethod', {
      header: t('activity.method'),
      cell: (i) => (i.getValue() as string) ?? '—',
    }),
    col.accessor('ActivityDateInserted', {
      header: t('activity.date'),
      cell: (i) => {
        try { return format(parseISO(i.getValue() as string), 'dd/MM/yyyy HH:mm') }
        catch { return i.getValue() as string }
      },
    }),
    col.accessor('PlayDuration', {
      header: t('activity.duration'),
      cell: (i) => {
        const seconds = Math.floor(((i.getValue() as number) ?? 0) / 10_000_000)
        const h = Math.floor(seconds / 3600)
        const m = Math.floor((seconds % 3600) / 60)
        const s = seconds % 60
        if (h > 0) return `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
        return `${m}:${String(s).padStart(2, '0')}`
      },
    }),
    col.accessor('RemoteEndPoint', {
      header: t('activity.ip'),
      cell: (i) => (i.getValue() as string) ?? '—',
    }),
  ]

  return (
    <>
      <PageHeader title={t('nav.activity')} />
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
      <DataTable
        data={data}
        columns={columns}
        loading={loading}
        searchPlaceholder={t('activity.search')}
      />
    </>
  )
}
