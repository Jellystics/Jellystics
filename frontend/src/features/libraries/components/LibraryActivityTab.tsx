import { useMemo } from 'react'
import { Box, Chip, Typography } from '@mui/material'
import { createColumnHelper } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'
import { VideoClip24Regular } from '@fluentui/react-icons'
import type { Activity } from '@/shared/types/activity'
import UserAvatar from '@/shared/components/UserAvatar/UserAvatar'
import DataTable, { type FilterDef } from '@/shared/components/DataTable/DataTable'
import { formatDateTime } from '@/shared/utils/formatDate'
import { formatDuration } from '@/shared/utils/formatTicks'
import { getActivityImageUrl } from '@/shared/utils/activityImage'

const colHelper = createColumnHelper<Activity>()

interface LibraryActivityTabProps {
  data: Activity[]
  loading: boolean
  onRefresh: () => void
}

export default function LibraryActivityTab({ data, loading, onRefresh }: LibraryActivityTabProps) {
  const { t } = useTranslation()

  const columns = useMemo(() => [
    colHelper.accessor('UserName', {
      header: t('activity.user', 'User'),
      cell: (info) => {
        const row = info.row.original
        return (
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <UserAvatar userId={row.UserId} userName={row.UserName} size={32} />
            <Typography variant="body2" noWrap>{row.UserName}</Typography>
          </Box>
        )
      },
    }),
    colHelper.accessor('NowPlayingItemName', {
      header: t('activity.item', 'Media'),
      cell: (info) => {
        const row = info.row.original
        const label = row.SeriesName ? `${row.SeriesName} — ${row.NowPlayingItemName}` : row.NowPlayingItemName
        const imgUrl = getActivityImageUrl(row, 80)
        return (
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Box
              sx={{
                width: 40, height: 40, borderRadius: 0.5, overflow: 'hidden', flexShrink: 0,
                bgcolor: 'rgba(255,255,255,0.06)', display: 'flex', alignItems: 'center', justifyContent: 'center', position: 'relative',
              }}
            >
              <VideoClip24Regular style={{ fontSize: 14, opacity: 0.4, position: 'absolute' }} />
              {imgUrl && (
                <Box
                  component="img"
                  src={imgUrl}
                  alt={row.NowPlayingItemName}
                  loading="lazy"
                  onError={(e) => { e.currentTarget.style.display = 'none' }}
                  sx={{ position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover' }}
                />
              )}
            </Box>
            <Typography variant="body2" noWrap title={label}>{label}</Typography>
          </Box>
        )
      },
    }),
    colHelper.accessor('Client', {
      header: t('activity.client', 'Client'),
      cell: (info) => <Typography variant="body2" noWrap>{info.getValue()}</Typography>,
    }),
    colHelper.accessor('DeviceName', {
      header: t('activity.device', 'Device'),
      cell: (info) => <Typography variant="body2" noWrap>{info.getValue()}</Typography>,
    }),
    colHelper.accessor('PlayMethod', {
      header: t('activity.method', 'Method'),
      cell: (info) => {
        const v = info.getValue()
        return v
          ? <Chip label={v} size="small" variant="outlined" sx={{ fontSize: 11, height: 20 }} />
          : <Typography variant="caption" color="text.disabled">—</Typography>
      },
    }),
    colHelper.accessor('ActivityDateInserted', {
      header: t('activity.date', 'Date'),
      cell: (info) => (
        <Typography variant="body2" sx={{ whiteSpace: 'nowrap' }}>
          {formatDateTime(info.getValue())}
        </Typography>
      ),
    }),
    colHelper.accessor('PlayDuration', {
      header: t('activity.duration', 'Duration'),
      cell: (info) => {
        const v = info.getValue()
        return (
          <Typography variant="body2" sx={{ fontVariantNumeric: 'tabular-nums', whiteSpace: 'nowrap' }}>
            {v > 0 ? formatDuration(v) : '—'}
          </Typography>
        )
      },
    }),
    colHelper.accessor('RemoteEndPoint', {
      header: t('activity.ip', 'IP'),
      cell: (info) => {
        const v = info.getValue()
        return <Typography variant="caption" color="text.secondary">{v ?? '—'}</Typography>
      },
    }),
  ], [t])

  const filterDefs = useMemo<FilterDef[]>(() => [
    { id: 'UserName', label: t('activity.user'), type: 'select' },
    { id: 'Client', label: t('activity.client'), type: 'select' },
    { id: 'PlayMethod', label: t('activity.method'), type: 'select' },
    { id: 'NowPlayingItemType', label: t('activity.mediaType', 'Type'), type: 'select' },
    { id: 'DeviceName', label: t('activity.device'), type: 'select' },
    {
      id: 'PlayDuration',
      label: t('activity.duration'),
      type: 'range',
      unit: 'min',
      transform: (ticks: number) => Math.floor(ticks / 10_000_000 / 60),
    },
  ], [t])

  return (
    <DataTable
      data={data}
      columns={columns}
      loading={loading}
      searchable
      searchPlaceholder={t('activity.search', 'Search activity...')}
      filterDefs={filterDefs}
      onRefresh={onRefresh}
    />
  )
}
