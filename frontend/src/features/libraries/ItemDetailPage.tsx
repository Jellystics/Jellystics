import React, { useEffect, useState, useCallback, useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  Alert, Box, Card, CardContent, Chip, Grid, Skeleton, Typography,
} from '@mui/material'
import { createColumnHelper, type ColumnDef } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'
import {
  ArrowLeft24Regular, Clock24Regular, People24Regular,
  Play24Regular, Star24Regular,
} from '@fluentui/react-icons'
import PageHeader from '@/shared/components/PageHeader/PageHeader'
import StatCard from '@/shared/components/StatCard/StatCard'
import DataTable, { type FilterDef } from '@/shared/components/DataTable/DataTable'
import api from '@/lib/axios'
import { formatWatchTime } from '@/shared/utils/formatWatchTime'
import type { ItemDetails, ItemWatchHistory, ItemWatchUser } from '@/shared/types/library'
import { formatDateTime } from '@/shared/utils/formatDate'

const userCol = createColumnHelper<ItemWatchUser>()
const historyCol = createColumnHelper<ItemWatchHistory>()

import { getItemImageUrl, getUserImageUrl } from '@/shared/utils/imageUrl'

function posterUrl(itemId: string, fallbackId?: string): string {
  return getItemImageUrl(fallbackId ?? itemId, 420, 95)
}

export default function ItemDetailPage() {
  const { t } = useTranslation()
  const { itemId } = useParams<{ itemId: string }>()
  const navigate = useNavigate()
  const [details, setDetails] = useState<ItemDetails | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback((showLoading = true) => {
    if (!itemId) return
    if (showLoading) setLoading(true)
    api.get(`/stats/getItemDetails?itemId=${itemId}`)
      .then((res) => setDetails(res.data))
      .catch(() => setError(t('common.loadError')))
      .finally(() => setLoading(false))
  }, [itemId, t])

  useEffect(() => {
    load()
    const interval = window.setInterval(() => load(false), 15000)
    return () => window.clearInterval(interval)
  }, [load])

  const item = details?.item

  const userColumns: ColumnDef<ItemWatchUser, any>[] = [
    userCol.accessor('UserName', {
      header: t('activity.user'),
      cell: (info) => {
        const { UserId, UserName } = info.row.original
        return (
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Box sx={{ position: 'relative', width: 28, height: 28, flexShrink: 0 }}>
              <Box
                sx={{
                  position: 'absolute', inset: 0, borderRadius: '50%',
                  bgcolor: 'primary.main', display: 'flex', alignItems: 'center',
                  justifyContent: 'center', fontSize: 11, fontWeight: 700, color: 'primary.contrastText',
                }}
              >
                {(UserName || '?').charAt(0).toUpperCase()}
              </Box>
              <Box
                component="img"
                src={getUserImageUrl(UserId, 56, 80)}
                onError={(e: React.SyntheticEvent<HTMLImageElement>) => { e.currentTarget.style.display = 'none' }}
                sx={{ position: 'absolute', inset: 0, width: '100%', height: '100%', borderRadius: '50%', objectFit: 'cover' }}
              />
            </Box>
            <Typography variant="body2" sx={{ fontWeight: 500 }}>{UserName}</Typography>
            {info.row.original.IsActive && <Chip label={t('status.playing')} size="small" color="primary" sx={{ height: 20, fontSize: 11 }} />}
          </Box>
        )
      },
    }),
    userCol.accessor('PlayCount', { header: t('common.plays') }),
    userCol.accessor('TotalWatchTime', { header: t('stats.watchTime'), cell: (info) => formatWatchTime(info.getValue()) }),
    userCol.accessor('LastWatched', { header: t('users.lastSeen'), cell: (info) => formatDateTime(info.getValue()) }),
  ]

  const historyColumns: ColumnDef<ItemWatchHistory, any>[] = [
    historyCol.accessor('UserName', {
      header: t('activity.user'),
      cell: (info) => {
        const { UserId, UserName } = info.row.original
        return (
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Box sx={{ position: 'relative', width: 28, height: 28, flexShrink: 0 }}>
              <Box
                sx={{
                  position: 'absolute', inset: 0, borderRadius: '50%',
                  bgcolor: 'primary.main', display: 'flex', alignItems: 'center',
                  justifyContent: 'center', fontSize: 11, fontWeight: 700, color: 'primary.contrastText',
                }}
              >
                {(UserName || '?').charAt(0).toUpperCase()}
              </Box>
              <Box
                component="img"
                src={getUserImageUrl(UserId, 56, 80)}
                onError={(e: React.SyntheticEvent<HTMLImageElement>) => { e.currentTarget.style.display = 'none' }}
                sx={{ position: 'absolute', inset: 0, width: '100%', height: '100%', borderRadius: '50%', objectFit: 'cover' }}
              />
            </Box>
            {UserName}
          </Box>
        )
      },
    }),
    historyCol.accessor('ActivityDateInserted', { header: t('activity.date'), cell: (info) => formatDateTime(info.getValue()) }),
    historyCol.accessor('PlaybackDuration', { header: t('activity.duration'), cell: (info) => formatWatchTime(info.getValue()) }),
    historyCol.accessor('Client', { header: t('activity.client'), cell: (info) => info.getValue() ?? '—' }),
    historyCol.accessor('DeviceName', { header: t('activity.device'), cell: (info) => info.getValue() ?? '—' }),
    historyCol.accessor('PlayMethod', { header: t('activity.method'), cell: (info) => info.getValue() ?? '—' }),
    historyCol.accessor('IsActive', {
      header: t('status.status'),
      cell: (info) => info.getValue()
        ? <Chip label={t('status.playing')} size="small" color="primary" sx={{ height: 20, fontSize: 11 }} />
        : <Chip label={t('status.finished')} size="small" sx={{ height: 20, fontSize: 11 }} />,
    }),
  ]

  const historyFilterDefs = useMemo<FilterDef[]>(() => [
    { id: 'UserName', label: t('activity.user'), type: 'select' },
    { id: 'Client', label: t('activity.client'), type: 'select' },
    { id: 'DeviceName', label: t('activity.device'), type: 'select' },
    { id: 'PlayMethod', label: t('activity.method'), type: 'select' },
  ], [t])

  return (
    <>
      <Box
        component="button"
        onClick={() => navigate(-1 as any)}
        style={{ all: 'unset', cursor: 'pointer' }}
      >
        <Typography variant="body2" color="primary.main" sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mb: 2 }}>
          <ArrowLeft24Regular style={{ fontSize: 18 }} />
          {t('common.back', 'Back')}
        </Typography>
      </Box>

      <PageHeader title={item?.Name ?? t('item.mediaFallback')} />
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      <Grid container spacing={3} sx={{ mb: 3 }}>
        <Grid size={{ xs: 12, md: 3 }}>
          {loading ? (
            <Skeleton variant="rectangular" height={360} sx={{ borderRadius: 2 }} />
          ) : item ? (
            <Card sx={{ overflow: 'hidden' }}>
              <Box
                component="img"
                src={posterUrl(item.Id, item.Type === 'Audio' ? item.AlbumId : undefined)}
                alt={item.Name}
                onError={(e: React.SyntheticEvent<HTMLImageElement>) => {
                  // Fallback to item's own image if album image fails
                  if (item.AlbumId && e.currentTarget.src.includes(item.AlbumId)) {
                    e.currentTarget.src = posterUrl(item.Id)
                  } else {
                    e.currentTarget.style.display = 'none'
                  }
                }}
                sx={{ width: '100%', aspectRatio: item.Type === 'Audio' ? '1 / 1' : '2 / 3', objectFit: 'cover', display: 'block' }}
              />
            </Card>
          ) : null}
        </Grid>

        <Grid size={{ xs: 12, md: 9 }}>
          <Grid container spacing={2} sx={{ mb: 2 }}>
            <Grid size={{ xs: 6, md: 3 }}>
              <StatCard label={t('common.plays')} value={details?.stats.TotalPlays ?? '—'} icon={<Play24Regular />} loading={loading} />
            </Grid>
            <Grid size={{ xs: 6, md: 3 }}>
              <StatCard label={t('stats.watchTime')} value={details ? formatWatchTime(details.stats.TotalWatchTime) : '—'} icon={<Clock24Regular />} loading={loading} />
            </Grid>
            <Grid size={{ xs: 6, md: 3 }}>
              <StatCard label={t('stats.totalUsers')} value={details?.stats.UniqueUsers ?? '—'} icon={<People24Regular />} loading={loading} />
            </Grid>
            <Grid size={{ xs: 6, md: 3 }}>
              <StatCard label={t('library.rating')} value={item?.CommunityRating ? `★ ${item.CommunityRating.toFixed(1)}` : '—'} icon={<Star24Regular />} loading={loading} />
            </Grid>
          </Grid>

          <Card>
            <CardContent>
              {loading ? (
                <>
                  <Skeleton width="50%" height={28} />
                  <Skeleton width="80%" />
                  <Skeleton width="70%" />
                </>
              ) : item ? (
                <Box>
                  <Typography variant="h6" sx={{ fontWeight: 700, mb: 0.5 }}>{item.Name}</Typography>
                  {item.Artist && (
                    <Typography variant="body1" color="text.secondary" sx={{ fontWeight: 500, mb: 0.5 }}>
                      {item.Artist}
                    </Typography>
                  )}
                  {item.AlbumName && (
                    <Typography
                      variant="body2"
                      color="text.secondary"
                      sx={{
                        mb: 1,
                        ...(item.AlbumId ? { cursor: 'pointer', '&:hover': { textDecoration: 'underline' } } : {}),
                      }}
                      onClick={() => item.AlbumId && navigate(`/libraries/${item.ParentId}/albums/${item.AlbumId}`)}
                    >
                      {item.AlbumName}
                    </Typography>
                  )}
                  <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', mb: 2 }}>
                    {item.Type && <Chip label={item.Type} size="small" />}
                    {item.ProductionYear && <Chip label={item.ProductionYear} size="small" />}
                    {details?.stats.IsActive && <Chip label={t('status.nowPlaying')} color="primary" size="small" />}
                    {item.Genres?.map((genre) => <Chip key={genre} label={genre} size="small" variant="outlined" />)}
                  </Box>
                  <Typography variant="body2" color="text.secondary">
                    {t('item.lastWatched')}: {formatDateTime(details?.stats.LastWatched)}
                  </Typography>
                  {item.Path && (
                    <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 1, wordBreak: 'break-all' }}>
                      {item.Path}
                    </Typography>
                  )}
                </Box>
              ) : (
                <Typography variant="body2" color="text.secondary">{t('item.notFound')}</Typography>
              )}
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="h6" sx={{ fontWeight: 700, mb: 2 }}>{t('item.whoWatched')}</Typography>
          <DataTable data={details?.users ?? []} columns={userColumns} loading={loading} searchable={false} onRefresh={() => load(false)} />
        </CardContent>
      </Card>

      <Card>
        <CardContent>
          <Typography variant="h6" sx={{ fontWeight: 700, mb: 2 }}>{t('item.watchHistory')}</Typography>
          <DataTable data={details?.history ?? []} columns={historyColumns} loading={loading} searchPlaceholder={t('item.searchHistory')} filterDefs={historyFilterDefs} onRefresh={() => load(false)} />
        </CardContent>
      </Card>
    </>
  )
}
