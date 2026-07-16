import { useState, useEffect, useMemo, useCallback, useRef } from 'react'
import { Alert, Box, Chip } from '@mui/material'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { createColumnHelper } from '@tanstack/react-table'
import type { ColumnDef } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'
import PageHeader from '@/shared/components/PageHeader/PageHeader'
import UserAvatar from '@/shared/components/UserAvatar/UserAvatar'
import MediaPoster from '@/shared/components/MediaPoster/MediaPoster'
import DataTable, { type FilterDef, type FilterState } from '@/shared/components/DataTable/DataTable'
import api from '@/lib/axios'
import type { Activity } from '@/shared/types/activity'
import { formatDateTime } from '@/shared/utils/formatDate'
import { formatDuration, ticksToMinutes } from '@/shared/utils/formatTicks'
import { getActivityImageUrl } from '@/shared/utils/activityImage'

const FILTER_URL_KEYS: Record<string, string> = {
  Client: 'client',
  PlayMethod: 'playMethod',
  NowPlayingItemType: 'mediaType',
  DeviceName: 'deviceName',
  UserName: 'userName',
}
const URL_FILTER_IDS = Object.fromEntries(Object.entries(FILTER_URL_KEYS).map(([id, key]) => [key, id]))

function parseFiltersFromParams(params: URLSearchParams): FilterState {
  const filters: FilterState = {}
  for (const [urlKey, filterId] of Object.entries(URL_FILTER_IDS)) {
    const val = params.get(urlKey)
    if (val) filters[filterId] = val.split(',')
  }
  const drFrom = params.get('date_from') ?? undefined
  const drTo = params.get('date_to') ?? undefined
  if (drFrom || drTo) filters['ActivityDateInserted'] = [drFrom, drTo]
  const durMin = params.get('dur_min')
  const durMax = params.get('dur_max')
  if (durMin || durMax) filters['PlayDuration'] = [durMin ? Number(durMin) : undefined, durMax ? Number(durMax) : undefined]
  return filters
}

function serializeFiltersToParams(filters: FilterState, params: URLSearchParams) {
  for (const urlKey of Object.values(FILTER_URL_KEYS)) params.delete(urlKey)
  params.delete('date_from'); params.delete('date_to')
  params.delete('dur_min'); params.delete('dur_max')

  for (const [id, value] of Object.entries(filters)) {
    const urlKey = FILTER_URL_KEYS[id]
    if (urlKey) {
      const selected = value as string[]
      if (selected.length) params.set(urlKey, selected.join(','))
    } else if (id === 'ActivityDateInserted') {
      const [from, to] = value as [string | undefined, string | undefined]
      if (from) params.set('date_from', from)
      if (to) params.set('date_to', to)
    } else if (id === 'PlayDuration') {
      const [min, max] = value as [number | undefined, number | undefined]
      if (min !== undefined) params.set('dur_min', String(min))
      if (max !== undefined) params.set('dur_max', String(max))
    }
  }
}

const col = createColumnHelper<Activity>()

export default function ActivityPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const [data, setData] = useState<Activity[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // eslint-disable-next-line react-hooks/exhaustive-deps
  const initialFilters = useMemo(() => parseFiltersFromParams(searchParams), [])
  // eslint-disable-next-line react-hooks/exhaustive-deps
  const initialSearch = useMemo(() => searchParams.get('q') ?? '', [])

  const activeFiltersRef = useRef<FilterState>(initialFilters)

  const load = useCallback((showLoading = true) => {
    if (showLoading) setLoading(true)
    const filters = activeFiltersRef.current
    const params = new URLSearchParams()

    const toCSV = (arr: string[]) => arr.join(',')

    if (filters['Client']) params.set('client', toCSV(filters['Client'] as string[]))
    if (filters['PlayMethod']) params.set('playMethod', toCSV(filters['PlayMethod'] as string[]))
    if (filters['NowPlayingItemType']) params.set('mediaType', toCSV(filters['NowPlayingItemType'] as string[]))
    if (filters['DeviceName']) params.set('deviceName', toCSV(filters['DeviceName'] as string[]))
    if (filters['UserName']) params.set('userName', toCSV(filters['UserName'] as string[]))

    const dr = filters['ActivityDateInserted'] as [string | undefined, string | undefined] | undefined
    if (dr?.[0]) params.set('dateFrom', dr[0])
    if (dr?.[1]) params.set('dateTo', dr[1])

    const dur = filters['PlayDuration'] as [number | undefined, number | undefined] | undefined
    if (dur?.[0] !== undefined) params.set('durMin', String(dur[0]))
    if (dur?.[1] !== undefined) params.set('durMax', String(dur[1]))

    const query = params.size ? `?${params.toString()}` : ''
    api
      .get(`/stats/getAllUserActivity${query}`)
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
            <UserAvatar userId={UserId} userName={UserName} size={32} />
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
            <MediaPoster src={getActivityImageUrl(row, 90)} type={row.NowPlayingItemType} width={45} height={30} sx={{ borderRadius: 0.75 }} />
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
      cell: (i) => formatDateTime(i.getValue() as string),
    }),
    col.accessor('PlayDuration', {
      header: t('activity.duration'),
      cell: (i) => formatDuration(i.getValue() as number),
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
      transform: (ticks: number) => ticksToMinutes(ticks),
    },
    { id: 'ActivityDateInserted', label: t('activity.date'), type: 'daterange' },
  ], [t])

  const handleFiltersChange = useCallback((filters: FilterState) => {
    activeFiltersRef.current = filters
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      serializeFiltersToParams(filters, next)
      return next
    }, { replace: true })
    load(true)
  }, [setSearchParams, load])

  const handleSearchChange = useCallback((search: string) => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      if (search) next.set('q', search)
      else next.delete('q')
      return next
    }, { replace: true })
  }, [setSearchParams])

  return (
    <>
      <PageHeader title={t('nav.activity')} onRefresh={() => load()} loading={loading} />
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
      <DataTable
        data={data}
        columns={columns}
        loading={loading}
        searchPlaceholder={t('activity.search')}
        initialColumnVisibility={{ ApplicationVersion: false }}
        filterDefs={filterDefs}
        onRefresh={() => load(true)}
        initialFilters={initialFilters}
        onFiltersChange={handleFiltersChange}
        initialSearch={initialSearch}
        onSearchChange={handleSearchChange}
        manualFiltering
      />
    </>
  )
}
