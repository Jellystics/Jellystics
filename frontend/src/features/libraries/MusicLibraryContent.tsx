import { useState, useEffect, useMemo, useCallback } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import {
  Grid, Alert, Card, CardActionArea, CardContent, Typography, Tabs, Tab, Box,
  Chip, List, ListItem, ListItemText, Skeleton,
  IconButton, Tooltip,
} from '@mui/material'
import { createColumnHelper } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'
import StatCard from '@/shared/components/StatCard/StatCard'
import DataTable from '@/shared/components/DataTable/DataTable'
import MediaPoster from '@/shared/components/MediaPoster/MediaPoster'
import api from '@/lib/axios'
import type { LibraryStats, MusicTrack, MusicAlbum, MusicArtist } from '@/shared/types/library'
import type { Activity } from '@/shared/types/activity'
import {
  Play24Regular, Clock24Regular, Star24Regular,
  MusicNote224Regular, Person24Regular, ArrowLeft24Regular,
} from '@fluentui/react-icons'
import { formatWatchTime } from '@/shared/utils/formatWatchTime'
import { formatTicks } from '@/shared/utils/formatTicks'
import { formatDateOnly } from '@/shared/utils/formatDate'
import { getItemImageUrl } from '@/shared/utils/imageUrl'

import LibraryActivityTab from './components/LibraryActivityTab'
import LibrarySharedStats from './components/LibrarySharedStats'
import type { HistoryPoint, PlayMethodStat, LastPlayedRow, TimeToWatchData, UnwatchedContentData } from './components/types'

const VIEWS = ['albums', 'artists', 'tracks', 'stats', 'activity'] as const

// ─── Album grid ────────────────────────────────────────────────────────────

function AlbumGrid({ albums, libraryId, loading }: {
  albums: MusicAlbum[]; libraryId: string; loading: boolean
}) {
  const { t } = useTranslation()
  const navigate = useNavigate()

  return (
    <Grid container spacing={2}>
      {loading ? (
        Array.from({ length: 12 }).map((_, i) => (
          <Grid key={i} size={{ xs: 6, sm: 4, md: 3, lg: 2 }}>
            <Skeleton variant="rectangular" height={220} sx={{ borderRadius: 2 }} />
          </Grid>
        ))
      ) : albums.length === 0 ? (
        <Grid size={{ xs: 12 }}>
          <Typography variant="body2" color="text.secondary" sx={{ py: 4, textAlign: 'center' }}>
            {t('common.noData')}
          </Typography>
        </Grid>
      ) : (
        albums.map((album) => (
          <Grid key={album.Id} size={{ xs: 6, sm: 4, md: 3, lg: 2 }}>
            <Card sx={{ height: '100%', overflow: 'hidden', borderRadius: 2, border: '1px solid', borderColor: 'divider', transition: 'border-color 160ms ease', '&:hover': { borderColor: 'primary.main' } }}>
              <CardActionArea onClick={() => navigate(`/libraries/${libraryId}/albums/${album.Id}`)} sx={{ height: '100%' }}>
                <Box sx={{ position: 'relative', aspectRatio: '1 / 1', bgcolor: 'rgba(255,255,255,0.04)', display: 'flex', alignItems: 'center', justifyContent: 'center', overflow: 'hidden' }}>
                  <MusicNote224Regular style={{ fontSize: 44, opacity: 0.45 }} />
                  <Box
                    component="img" src={getItemImageUrl(album.Id, 360)} alt={album.Name} loading="lazy"
                    onError={(e) => { e.currentTarget.style.display = 'none' }}
                    sx={{ position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover', transition: 'transform 300ms ease', '.MuiCard-root:hover &': { transform: 'scale(1.06)' } }}
                  />
                </Box>
                <CardContent sx={{ p: 1.25, '&:last-child': { pb: 1.25 } }}>
                  <Typography variant="body2" sx={{ fontWeight: 600, lineHeight: 1.25 }} noWrap title={album.Name}>{album.Name}</Typography>
                  {album.AlbumArtist && (
                    <Typography variant="caption" color="text.secondary" noWrap sx={{ display: 'block', mt: 0.25 }}>{album.AlbumArtist}</Typography>
                  )}
                  <Box sx={{ display: 'flex', gap: 0.75, mt: 0.5, flexWrap: 'wrap' }}>
                    {album.ProductionYear && <Typography variant="caption" color="text.secondary">{album.ProductionYear}</Typography>}
                    {album.TrackCount > 0 && <Typography variant="caption" color="text.secondary">· {album.TrackCount} {t('library.tracks', 'titres')}</Typography>}
                  </Box>
                </CardContent>
              </CardActionArea>
            </Card>
          </Grid>
        ))
      )}
    </Grid>
  )
}

// ─── Tracks table ──────────────────────────────────────────────────────────

const colHelper = createColumnHelper<MusicTrack>()

function TracksTable({ tracks, loading, libraryId }: {
  tracks: MusicTrack[]; loading: boolean; libraryId: string
}) {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const columns = useMemo(() => [
    colHelper.display({
      id: 'poster', size: 44, header: () => null, meta: { hideFromColumnsMenu: true },
      cell: (info) => {
        const row = info.row.original
        return row.AlbumId
          ? <MediaPoster src={getItemImageUrl(row.AlbumId, 56, 80)} alt={row.AlbumName ?? ''} type="Audio" width={28} height={28} sx={{ borderRadius: '50%' }} />
          : null
      },
    }),
    colHelper.accessor('IndexNumber', {
      header: '#',
      cell: (info) => <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600 }}>{info.getValue() ?? '—'}</Typography>,
    }),
    colHelper.accessor('Name', {
      header: t('library.trackTitle', 'Titre'),
      cell: (info) => <Typography variant="body2" sx={{ fontWeight: 500 }} noWrap>{info.getValue()}</Typography>,
    }),
    colHelper.accessor('Artist', {
      header: t('library.artist', 'Artiste'),
      cell: (info) => <Typography variant="body2" color="text.secondary" noWrap>{info.getValue() ?? '—'}</Typography>,
    }),
    colHelper.accessor('AlbumName', {
      header: t('library.album', 'Album'),
      cell: (info) => {
        const row = info.row.original
        return (
          <Typography
            variant="body2" color="text.secondary" noWrap
            sx={row.AlbumId ? { cursor: 'pointer', '&:hover': { color: 'primary.light', textDecoration: 'underline' } } : {}}
            onClick={row.AlbumId ? (e) => { e.stopPropagation(); navigate(`/libraries/${libraryId}/albums/${row.AlbumId}`) } : undefined}
          >
            {info.getValue() ?? '—'}
          </Typography>
        )
      },
    }),
    colHelper.accessor('RunTimeTicks', {
      header: t('library.duration', 'Durée'),
      cell: (info) => <Typography variant="caption" color="text.secondary">{formatTicks(info.getValue())}</Typography>,
    }),
    colHelper.accessor('PlayCount', {
      header: t('common.plays'),
      cell: (info) => {
        const v = info.getValue()
        return v > 0
          ? <Chip label={v} size="small" sx={{ fontSize: 11, height: 20 }} />
          : <Typography variant="caption" color="text.disabled">0</Typography>
      },
    }),
  ], [t, navigate, libraryId])

  return (
    <DataTable data={tracks} columns={columns} loading={loading} searchPlaceholder={`${t('common.search')}…`} defaultPageSize={50} />
  )
}

// ─── Artists list ──────────────────────────────────────────────────────────

function ArtistsList({ artists, loading, onSelectArtist }: {
  artists: MusicArtist[]; loading: boolean
  onSelectArtist: (id: string, name: string) => void
}) {
  const { t } = useTranslation()

  return (
    <Card>
      <CardContent sx={{ p: 0, '&:last-child': { pb: 0 } }}>
        {loading ? (
          Array.from({ length: 8 }).map((_, i) => (
            <Box key={i} sx={{ display: 'flex', gap: 2, px: 2, py: 1.5, alignItems: 'center' }}>
              <Skeleton variant="circular" width={40} height={40} />
              <Box sx={{ flex: 1 }}>
                <Skeleton variant="text" width="45%" />
                <Skeleton variant="text" width="25%" />
              </Box>
            </Box>
          ))
        ) : artists.length === 0 ? (
          <Typography variant="body2" color="text.secondary" sx={{ p: 3, textAlign: 'center' }}>
            {t('common.noData')}
          </Typography>
        ) : (
          <List disablePadding>
            {artists.map((artist) => (
              <ListItem
                key={artist.Name}
                disablePadding
                sx={{
                  px: 2, py: 1, borderBottom: '1px solid', borderColor: 'divider',
                  '&:last-child': { borderBottom: 0 },
                  cursor: 'pointer', transition: 'background 150ms',
                  '&:hover': { bgcolor: 'action.hover' },
                  gap: 1.5, alignItems: 'center',
                }}
                onClick={() => onSelectArtist(artist.Id, artist.Name)}
              >
                <Box sx={{ width: 40, height: 40, borderRadius: '50%', bgcolor: 'rgba(255,255,255,0.07)', overflow: 'hidden', position: 'relative', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                  <Person24Regular style={{ fontSize: 20, opacity: 0.6 }} />
                  {artist.ImageTagsPrimary && (
                    <Box
                      component="img" src={getItemImageUrl(artist.Id, 80, 80)}
                      onError={(e: React.SyntheticEvent<HTMLImageElement>) => { e.currentTarget.style.display = 'none' }}
                      sx={{ position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover' }}
                    />
                  )}
                </Box>
                <ListItemText
                  primary={artist.Name}
                  secondary={`${artist.AlbumCount} ${t('library.albums', 'albums')} · ${artist.TrackCount} ${t('library.tracks', 'titres')}`}
                  slotProps={{
                    primary: { style: { fontWeight: 600, fontSize: 14 } },
                    secondary: { style: { fontSize: 12 } },
                  }}
                />
                {artist.PlayCount > 0 && (
                  <Chip label={`${artist.PlayCount} ${t('common.plays')}`} size="small" sx={{ fontSize: 11, height: 20, flexShrink: 0 }} />
                )}
              </ListItem>
            ))}
          </List>
        )}
      </CardContent>
    </Card>
  )
}

// ─── Main content ──────────────────────────────────────────────────────────

interface MusicLibraryContentProps {
  libraryId: string
  libraryName: string
}

export default function MusicLibraryContent({ libraryId }: MusicLibraryContentProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()

  const [tracks, setTracks] = useState<MusicTrack[]>([])
  const [albums, setAlbums] = useState<MusicAlbum[]>([])
  const [artists, setArtists] = useState<MusicArtist[]>([])
  const [artistAlbums, setArtistAlbums] = useState<MusicAlbum[]>([])
  const [selectedArtist, setSelectedArtist] = useState<string | null>(null)
  const [artistLoading, setArtistLoading] = useState(false)
  const [stats, setStats] = useState<LibraryStats | null>(null)
  const [loading, setLoading] = useState(true)

  const [historyData, setHistoryData] = useState<HistoryPoint[]>([])
  const [activityHistory, setActivityHistory] = useState<Activity[]>([])
  const [playMethodStats, setPlayMethodStats] = useState<PlayMethodStat[]>([])
  const [lastPlayed, setLastPlayed] = useState<LastPlayedRow[]>([])
  const [timeToWatch, setTimeToWatch] = useState<TimeToWatchData | null>(null)
  const [timeToWatchLoading, setTimeToWatchLoading] = useState(false)
  const [unwatchedContent, setUnwatchedContent] = useState<UnwatchedContentData | null>(null)
  const [unwatchedLoading, setUnwatchedLoading] = useState(false)

  const albumsSynced = albums.length > 0

  const load = useCallback((showLoading = true) => {
    if (showLoading) setLoading(true)
    Promise.allSettled([
      api.get(`/stats/getLibraryStats?libraryId=${libraryId}`),
      api.get(`/stats/getLibraryAlbums?libraryId=${libraryId}`),
      api.get(`/stats/getLibraryArtists?libraryId=${libraryId}`),
      api.get(`/stats/getLibraryTracks?libraryId=${libraryId}`),
      api.post('/api/getLibraryHistory', { libraryid: libraryId }),
      api.post('/stats/getLibraryItemsPlayMethodStats', { libraryid: libraryId }),
      api.post('/stats/getLibraryLastPlayed', { libraryid: libraryId }),
    ]).then(([statsRes, albumsRes, artistsRes, tracksRes, historyRes, methodRes, lastPlayedRes]) => {
      if (statsRes.status === 'fulfilled') setStats(statsRes.value.data)
      if (albumsRes.status === 'fulfilled') setAlbums(albumsRes.value.data ?? [])
      if (artistsRes.status === 'fulfilled') setArtists(artistsRes.value.data ?? [])
      if (tracksRes.status === 'fulfilled') setTracks(tracksRes.value.data ?? [])

      if (historyRes.status === 'fulfilled') {
        const raw: Activity[] = historyRes.value.data?.results ?? []
        const byDate: Record<string, number> = {}
        raw.forEach((row) => {
          const day = formatDateOnly(row.ActivityDateInserted)
          if (day !== '—') byDate[day] = (byDate[day] ?? 0) + 1
        })
        setHistoryData(Object.entries(byDate).map(([date, plays]) => ({ date, plays })).sort((a, b) => a.date.localeCompare(b.date)))
        setActivityHistory(raw)
      }

      if (methodRes.status === 'fulfilled') setPlayMethodStats(methodRes.value.data?.stats ?? [])
      if (lastPlayedRes.status === 'fulfilled') setLastPlayed(lastPlayedRes.value.data ?? [])
    }).finally(() => setLoading(false))
  }, [libraryId])

  useEffect(() => {
    load()
    const interval = window.setInterval(() => load(false), 15000)
    return () => window.clearInterval(interval)
  }, [load])

  useEffect(() => {
    setTimeToWatchLoading(true)
    api.get(`/stats/getTimeToWatch?libraryId=${libraryId}&limit=10`)
      .then((res) => setTimeToWatch(res.data))
      .catch(() => setTimeToWatch(null))
      .finally(() => setTimeToWatchLoading(false))
  }, [libraryId])

  useEffect(() => {
    setUnwatchedLoading(true)
    api.get(`/stats/getUnwatchedContent?libraryId=${libraryId}&pageSize=10`)
      .then((res) => setUnwatchedContent(res.data))
      .catch(() => setUnwatchedContent(null))
      .finally(() => setUnwatchedLoading(false))
  }, [libraryId])

  const handleSelectArtist = async (artistId: string, artistName: string) => {
    setSelectedArtist(artistName)
    setArtistLoading(true)
    try {
      const res = await api.get(`/stats/getArtistAlbums?libraryId=${libraryId}&artistId=${encodeURIComponent(artistId)}`)
      setArtistAlbums(res.data ?? [])
    } catch {
      setArtistAlbums([])
    } finally {
      setArtistLoading(false)
    }
  }

  const topTracks = useMemo(() =>
    [...tracks].sort((a, b) => b.PlayCount - a.PlayCount).slice(0, 10).filter((t) => t.PlayCount > 0),
    [tracks]
  )

  const tab = Math.max(0, VIEWS.indexOf((searchParams.get('view') ?? '') as typeof VIEWS[number]))
  const setTab = (idx: number) => {
    setSearchParams({ view: VIEWS[idx] ?? VIEWS[0] }, { replace: true })
    setSelectedArtist(null)
  }

  return (
    <>
      {/* Stat cards */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard label={t('stats.totalItems')} value={stats?.TotalItems ?? '—'} icon={<Play24Regular />} loading={loading} />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard label={t('stats.totalPlays')} value={stats?.TotalPlayCount ?? '—'} icon={<Play24Regular />} loading={loading} />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard label={t('stats.watchTime')} value={stats ? formatWatchTime(stats.TotalWatchTime) : '—'} icon={<Clock24Regular />} loading={loading} />
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <StatCard label={t('library.topItem')} value={stats?.MostPlayedItem?.Name ?? '—'} icon={<Star24Regular />} loading={loading} />
        </Grid>
      </Grid>

      {/* Tabs */}
      <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 3 }}>
        <Tabs value={tab} onChange={(_, v) => setTab(v as number)} variant="scrollable" scrollButtons="auto">
          <Tab label={t('library.albums', 'Albums')} />
          <Tab label={t('library.artists', 'Artistes')} />
          <Tab label={t('library.tracks', 'Titres')} />
          <Tab label={t('library.stats', 'Stats')} />
          <Tab label={t('library.activityHistory', "Historique d'activité")} />
        </Tabs>
      </Box>

      {/* Albums tab */}
      {tab === 0 && (
        albumsSynced
          ? <AlbumGrid albums={albums} libraryId={libraryId} loading={loading} />
          : !loading && (
            <Alert severity="info">
              {t('library.syncRequired', 'Lancez une synchronisation complète pour afficher les albums et les artistes.')}
            </Alert>
          )
      )}

      {/* Artists tab */}
      {tab === 1 && (
        !albumsSynced && !loading ? (
          <Alert severity="info">
            {t('library.syncRequired', 'Lancez une synchronisation complète pour afficher les albums et les artistes.')}
          </Alert>
        ) : selectedArtist ? (
          <>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
              <Tooltip title={t('common.back', 'Retour')}>
                <IconButton size="small" onClick={() => setSelectedArtist(null)}>
                  <ArrowLeft24Regular />
                </IconButton>
              </Tooltip>
              <Typography variant="h6" sx={{ fontWeight: 600 }}>{selectedArtist}</Typography>
            </Box>
            <AlbumGrid albums={artistAlbums} libraryId={libraryId} loading={artistLoading} />
          </>
        ) : (
          <ArtistsList artists={artists} loading={loading} onSelectArtist={handleSelectArtist} />
        )
      )}

      {/* Tracks tab */}
      {tab === 2 && (
        albumsSynced
          ? <TracksTable tracks={tracks} loading={loading} libraryId={libraryId} />
          : !loading && (
            <Alert severity="info">
              {t('library.syncRequired', 'Lancez une synchronisation complète pour afficher les titres.')}
            </Alert>
          )
      )}

      {/* Stats tab */}
      {tab === 3 && (
        <Grid container spacing={2}>
          {/* Top played tracks */}
          {topTracks.length > 0 && (
            <Grid size={{ xs: 12 }}>
              <Card>
                <CardContent sx={{ p: 0, '&:last-child': { pb: 0 } }}>
                  <Typography variant="subtitle1" sx={{ fontWeight: 600, px: 2, pt: 2, pb: 1 }}>
                    {t('library.topTracks', 'Titres les plus écoutés')}
                  </Typography>
                  <List disablePadding>
                    {topTracks.map((track, i) => (
                      <ListItem
                        key={track.Id}
                        disablePadding
                        sx={{
                          px: 2, py: 0.75, borderTop: '1px solid', borderColor: 'divider',
                          gap: 1.5, alignItems: 'center',
                          cursor: track.AlbumId ? 'pointer' : 'default',
                          transition: 'background 150ms',
                          '&:hover': track.AlbumId ? { bgcolor: 'action.hover' } : {},
                        }}
                        onClick={() => track.AlbumId && navigate(`/libraries/${libraryId}/albums/${track.AlbumId}`)}
                      >
                        <Typography variant="caption" color="text.secondary" sx={{ minWidth: 20, textAlign: 'right', fontWeight: 700, flexShrink: 0 }}>
                          {i + 1}
                        </Typography>
                        {track.AlbumId && (
                          <MediaPoster src={getItemImageUrl(track.AlbumId, 72, 85)} alt={track.AlbumName ?? ''} type="Audio" width={32} height={32} sx={{ borderRadius: '50%' }} />
                        )}
                        <Box sx={{ flex: 1, minWidth: 0 }}>
                          <Typography variant="body2" sx={{ fontWeight: 500 }} noWrap>{track.Name}</Typography>
                          <Typography variant="caption" color="text.secondary" noWrap>
                            {[track.Artist, track.AlbumName].filter(Boolean).join(' — ')}
                          </Typography>
                        </Box>
                        <Chip label={`${track.PlayCount} ${t('common.plays')}`} size="small" sx={{ fontSize: 11, height: 20, flexShrink: 0 }} />
                      </ListItem>
                    ))}
                  </List>
                </CardContent>
              </Card>
            </Grid>
          )}

          <LibrarySharedStats
            historyData={historyData} playMethodStats={playMethodStats} lastPlayed={lastPlayed}
            timeToWatch={timeToWatch} timeToWatchLoading={timeToWatchLoading}
            unwatchedContent={unwatchedContent} unwatchedLoading={unwatchedLoading}
            loading={loading}
          />
        </Grid>
      )}

      {/* Activity tab */}
      {tab === 4 && (
        <LibraryActivityTab data={activityHistory} loading={loading} onRefresh={() => load(false)} />
      )}
    </>
  )
}
