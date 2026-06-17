import { useState, useEffect, useMemo, useCallback } from 'react'
import { Alert, Box, Chip } from '@mui/material'
import { useNavigate } from 'react-router-dom'
import { createColumnHelper } from '@tanstack/react-table'
import type { ColumnDef } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'
import { format, parseISO } from 'date-fns'
import { VideoClip24Regular } from '@fluentui/react-icons'
import PageHeader from '@/shared/components/PageHeader/PageHeader'
import DataTable, { type FilterDef } from '@/shared/components/DataTable/DataTable'
import api from '@/lib/axios'
import type { Activity } from '@/shared/types/activity'
import { getDateLocale } from '@/lib/dateLocale'

const col = createColumnHelper<Activity>()

export default function ActivityPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [data, setData] = useState<Activity[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback((showLoading = true) => {
    if (showLoading) setLoading(true)
    api
      .get('/stats/getAllUserActivity')
      .then((r) => setData(r.data ?? []))
      .catch(() => setError(t('common.loadError')))
      .finally(() => setLoading(false))
  }, [t])

  useEffect(() => {
    load()
    const interval = window.setInterval(() => load(false), 15000)
    return () => window.clearInterval(interval)
  }, [load])

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const columns = useMemo<ColumnDef<Activity, any>[]>(() => [
    col.accessor('UserName', {
      header: t('activity.user'),
      cell: (i) => {
        const { UserName, UserId } = i.row.original
        return (
          <Box
            onClick={(e) => { e.stopPropagation(); navigate(`/users/${UserId}`) }}
            sx={{ display: 'flex', alignItems: 'center', gap: 1, cursor: 'pointer', '&:hover .username': { textDecoration: 'underline' } }}
          >
            <Box sx={{ position: 'relative', width: 32, height: 32, flexShrink: 0 }}>
              {/* Fallback always rendered behind */}
              <Box
                sx={{
                  position: 'absolute',
                  inset: 0,
                  borderRadius: '50%',
                  bgcolor: 'primary.main',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: 12,
                  fontWeight: 700,
                  color: 'primary.contrastText',
                }}
              >
                {UserName?.[0]?.toUpperCase()}
              </Box>
              <Box
                component="img"
                src={`/proxy/Users/${UserId}/Images/Primary?fillWidth=32&quality=80`}
                onError={(e: React.SyntheticEvent<HTMLImageElement>) => {
                  e.currentTarget.style.display = 'none'
                }}
                sx={{
                  position: 'absolute',
                  inset: 0,
                  width: '100%',
                  height: '100%',
                  borderRadius: '50%',
                  objectFit: 'cover',
                }}
              />
            </Box>
            <span className="username">{UserName}</span>
          </Box>
        )
      },
    }),
    col.accessor('NowPlayingItemName', {
      header: t('activity.item'),
      cell: (i) => {
        const row = i.row.original
        const label = row.SeriesName ? `${row.SeriesName} — ${i.getValue()}` : i.getValue()
        return (
          <Box
            onClick={(e) => { e.stopPropagation(); navigate(`/items/${row.ItemId}`) }}
            sx={{ display: 'flex', alignItems: 'center', gap: 1, cursor: 'pointer', '&:hover .itemname': { textDecoration: 'underline' } }}
          >
            <Box
              sx={{
                width: 45,
                height: 30,
                borderRadius: 0.75,
                overflow: 'hidden',
                bgcolor: 'rgba(255,255,255,0.06)',
                flexShrink: 0,
                position: 'relative',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <VideoClip24Regular style={{ opacity: 0.4, fontSize: 16 }} />
              <Box
                component="img"
                src={`/proxy/Items/Images/Primary/?id=${encodeURIComponent(row.ParentId ?? row.ItemId)}&fillWidth=90&quality=80`}
                onError={(e: React.SyntheticEvent<HTMLImageElement>) => {
                  if (row.ParentId && e.currentTarget.src.includes(encodeURIComponent(row.ParentId))) {
                    e.currentTarget.src = `/proxy/Items/Images/Primary/?id=${encodeURIComponent(row.ItemId)}&fillWidth=90&quality=80`
                  } else {
                    e.currentTarget.style.display = 'none'
                  }
                }}
                sx={{
                  position: 'absolute',
                  inset: 0,
                  width: '100%',
                  height: '100%',
                  objectFit: 'cover',
                }}
              />
            </Box>
            <span className="itemname">{label}</span>
          </Box>
        )
      },
    }),
    col.accessor('NowPlayingItemType', {
      header: t('activity.mediaType', 'Type'),
      cell: (i) => {
        const val = i.getValue() as string | undefined
        return val ? <Chip label={val} size="small" variant="outlined" /> : '—'
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
        try {
          return format(parseISO(i.getValue() as string), 'dd/MM/yyyy HH:mm', { locale: getDateLocale() })
        } catch {
          return i.getValue() as string
        }
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
    col.accessor('ApplicationVersion', {
      header: t('activity.version', 'Version'),
    }),
  ], [t])

  const filterDefs = useMemo<FilterDef[]>(() => [
    { id: 'Client', label: t('activity.client'), type: 'select' },
    { id: 'PlayMethod', label: t('activity.method'), type: 'select' },
    { id: 'NowPlayingItemType', label: t('activity.mediaType', 'Type'), type: 'select' },
    { id: 'DeviceName', label: t('activity.device'), type: 'select' },
    { id: 'UserName', label: t('activity.user'), type: 'select' },
    {
      id: 'PlayDuration',
      label: t('activity.duration'),
      type: 'range',
      unit: 'min',
      transform: (ticks: number) => Math.floor(ticks / 10_000_000 / 60),
    },
    { id: 'ActivityDateInserted', label: t('activity.date'), type: 'daterange' },
  ], [t])

  return (
    <>
      <PageHeader title={t('nav.activity')} />
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
      <DataTable
        data={data}
        columns={columns}
        loading={loading}
        searchPlaceholder={t('activity.search')}
        initialColumnVisibility={{ ApplicationVersion: false }}
        filterDefs={filterDefs}
        onRefresh={load}
      />
    </>
  )
}
