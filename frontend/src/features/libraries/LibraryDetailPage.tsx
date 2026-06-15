import { useState, useEffect, useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  Grid, Alert, Card, CardActionArea, CardContent, Typography, Tabs, Tab, Box,
  Chip, List, ListItem, ListItemText, Skeleton, TextField, InputAdornment,
  IconButton, Tooltip, Table, TableBody, TableCell, TableHead, TableRow,
  TableSortLabel, TablePagination, Paper,
} from '@mui/material'
import {
  useReactTable, getCoreRowModel, getSortedRowModel, getFilteredRowModel,
  getPaginationRowModel, flexRender, createColumnHelper, type SortingState,
} from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'
import i18next from 'i18next'
import { format, parseISO } from 'date-fns'
import { PieChart } from '@mui/x-charts/PieChart'
import { LineChart } from '@mui/x-charts/LineChart'
import { BarChart } from '@mui/x-charts/BarChart'
import PageHeader from '@/shared/components/PageHeader/PageHeader'
import StatCard from '@/shared/components/StatCard/StatCard'
import ChartCard from '@/shared/components/ChartCard/ChartCard'
import api from '@/lib/axios'
import type { LibraryItem, LibraryStats, GenreStat } from '@/shared/types/library'
import {
  Play24Regular, Clock24Regular, Star24Regular,
  Search20Regular, VideoClip24Regular, MusicNote224Regular,
  Person24Regular, ArrowLeft24Regular,
} from '@fluentui/react-icons'
import { formatWatchTime } from '@/shared/utils/formatWatchTime'

const COLORS = ['#60a5fa', '#34d399', '#fb923c', '#f472b6', '#a78bfa', '#facc15', '#38bdf8', '#4ade80']
const CHART_BAR = '#60a5fa'

interface Track {
  Id: string
  Name: string
  Artist: string | null
  Album: string | null
  AlbumId: string | null
  IndexNumber: number | null
  RunTimeTicks: number | null
  PlayCount: number
}

interface Album {
  Id: string
  Name: string
  Artist: string | null
  ProductionYear: number | null
  TrackCount: number
  PlayCount: number
}

interface Artist {
  Name: string
  AlbumCount: number
  TrackCount: number
  PlayCount: number
}

function formatSize(bytes?: number): string | null {
  if (!bytes) return null
  const gb = bytes / 1024 / 1024 / 1024
  if (gb >= 1) return `${gb.toFixed(gb >= 10 ? 1 : 2)} ${i18next.t('units.gigabytes')}`
  const mb = bytes / 1024 / 1024
  return `${Math.round(mb)} ${i18next.t('units.megabytes')}`
}

function formatTicks(ticks: number | null): string {
  if (!ticks) return '—'
  const s = Math.floor(ticks / 10_000_000)
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`
}

type HistoryPoint = { date: string; plays: number }
type ItemWithStats = { Name: string; times_played: number; total_play_time: number }
type PlayMethodStat = { Key: string; Transcodes: number; DirectPlays: number }
type LastPlayedRow = { NowPlayingItemName: string; ActivityDateInserted: string; UserName: string }

// ─── Album grid ────────────────────────────────────────────────────────────

function AlbumGrid({ albums, libraryId, loading, navigate, t }: {
  albums: Album[]
  libraryId: string
  loading: boolean
  navigate: ReturnType<typeof useNavigate>
  t: (k: string, fb?: string) => string
}) {
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
            <Card
              sx={{
                height: '100%',
                overflow: 'hidden',
                borderRadius: 2,
                border: '1px solid',
                borderColor: 'divider',
                transition: 'transform 160ms ease, border-color 160ms ease',
                '&:hover': { transform: 'translateY(-3px)', borderColor: 'primary.main' },
              }}
            >
              <CardActionArea onClick={() => navigate(`/libraries/${libraryId}/albums/${album.Id}`)} sx={{ height: '100%' }}>
                <Box
                  sx={{
                    position: 'relative',
                    aspectRatio: '1 / 1',
                    bgcolor: 'rgba(255,255,255,0.04)',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                  }}
                >
                  <MusicNote224Regular style={{ fontSize: 44, opacity: 0.45 }} />
                  <Box
                    component="img"
                    src={`/proxy/Items/Images/Primary/?id=${encodeURIComponent(album.Id)}&fillWidth=360&quality=90`}
                    alt={album.Name}
                    loading="lazy"
                    onError={(e) => { e.currentTarget.style.display = 'none' }}
                    sx={{ position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover' }}
                  />
                </Box>
                <CardContent sx={{ p: 1.25, '&:last-child': { pb: 1.25 } }}>
                  <Typography variant="body2" sx={{ fontWeight: 600, lineHeight: 1.25 }} noWrap title={album.Name}>
                    {album.Name}
                  </Typography>
                  {album.Artist && (
                    <Typography variant="caption" color="text.secondary" noWrap sx={{ display: 'block', mt: 0.25 }}>
                      {album.Artist}
                    </Typography>
                  )}
                  <Box sx={{ display: 'flex', gap: 0.75, mt: 0.5, flexWrap: 'wrap' }}>
                    {album.ProductionYear && (
                      <Typography variant="caption" color="text.secondary">{album.ProductionYear}</Typography>
                    )}
                    {album.TrackCount > 0 && (
                      <Typography variant="caption" color="text.secondary">
                        · {album.TrackCount} {t('library.tracks', 'titres')}
                      </Typography>
                    )}
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

const colHelper = createColumnHelper<Track>()

function TracksTable({ tracks, loading, navigate, libraryId, t }: {
  tracks: Track[]
  loading: boolean
  navigate: ReturnType<typeof useNavigate>
  libraryId: string
  t: (k: string, fb?: string) => string
}) {
  const [sorting, setSorting] = useState<SortingState>([])
  const [globalFilter, setGlobalFilter] = useState('')

  const columns = useMemo(() => [
    colHelper.accessor('IndexNumber', {
      header: '#',
      size: 48,
      cell: (info) => (
        <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600 }}>
          {info.getValue() ?? '—'}
        </Typography>
      ),
    }),
    colHelper.accessor('Name', {
      header: t('library.trackTitle', 'Titre'),
      cell: (info) => (
        <Typography variant="body2" sx={{ fontWeight: 500 }} noWrap>
          {info.getValue()}
        </Typography>
      ),
    }),
    colHelper.accessor('Artist', {
      header: t('library.artist', 'Artiste'),
      cell: (info) => (
        <Typography variant="body2" color="text.secondary" noWrap>
          {info.getValue() ?? '—'}
        </Typography>
      ),
    }),
    colHelper.accessor('Album', {
      header: t('library.album', 'Album'),
      cell: (info) => {
        const row = info.row.original
        return (
          <Typography
            variant="body2"
            color="text.secondary"
            noWrap
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
      size: 72,
      cell: (info) => (
        <Typography variant="caption" color="text.secondary">
          {formatTicks(info.getValue())}
        </Typography>
      ),
    }),
    colHelper.accessor('PlayCount', {
      header: t('common.plays'),
      size: 80,
      cell: (info) => {
        const v = info.getValue()
        return v > 0
          ? <Chip label={v} size="small" sx={{ fontSize: 11, height: 20 }} />
          : <Typography variant="caption" color="text.disabled">0</Typography>
      },
    }),
  ], [t, navigate, libraryId])

  const table = useReactTable({
    data: tracks,
    columns,
    state: { sorting, globalFilter },
    onSortingChange: setSorting,
    onGlobalFilterChange: setGlobalFilter,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    initialState: { pagination: { pageSize: 50 } },
    globalFilterFn: (row, _colId, filterValue) => {
      const v = filterValue.toLowerCase()
      return (
        row.original.Name.toLowerCase().includes(v) ||
        (row.original.Artist ?? '').toLowerCase().includes(v) ||
        (row.original.Album ?? '').toLowerCase().includes(v)
      )
    },
  })

  if (loading) {
    return (
      <Paper variant="outlined" sx={{ borderRadius: 2 }}>
        {Array.from({ length: 8 }).map((_, i) => (
          <Box key={i} sx={{ display: 'flex', gap: 2, px: 2, py: 1.5, borderBottom: '1px solid', borderColor: 'divider' }}>
            <Skeleton variant="text" width={24} />
            <Skeleton variant="text" sx={{ flex: 1 }} />
            <Skeleton variant="text" width={100} />
            <Skeleton variant="text" width={100} />
            <Skeleton variant="text" width={40} />
          </Box>
        ))}
      </Paper>
    )
  }

  return (
    <Box>
      <Box sx={{ mb: 2 }}>
        <TextField
          size="small"
          placeholder={`${t('common.search')}…`}
          value={globalFilter}
          onChange={(e) => setGlobalFilter(e.target.value)}
          slotProps={{
            input: {
              startAdornment: (
                <InputAdornment position="start">
                  <Search20Regular style={{ fontSize: 16 }} />
                </InputAdornment>
              ),
            },
          }}
          sx={{ width: { xs: '100%', sm: 320 } }}
        />
        <Typography variant="caption" color="text.secondary" sx={{ ml: 1.5 }}>
          {table.getFilteredRowModel().rows.length} {t('library.tracks', 'titres')}
        </Typography>
      </Box>

      <Paper variant="outlined" sx={{ borderRadius: 2, overflow: 'hidden' }}>
        <Table size="small">
          <TableHead>
            <TableRow>
              {table.getHeaderGroups().flatMap((hg) => hg.headers).map((header) => (
                <TableCell
                  key={header.id}
                  style={{ width: header.column.columnDef.size }}
                  sx={{ fontWeight: 600, fontSize: 12, whiteSpace: 'nowrap', bgcolor: 'rgba(255,255,255,0.03)' }}
                >
                  {header.column.getCanSort() ? (
                    <TableSortLabel
                      active={header.column.getIsSorted() !== false}
                      direction={header.column.getIsSorted() === 'asc' ? 'asc' : 'desc'}
                      onClick={header.column.getToggleSortingHandler()}
                    >
                      {flexRender(header.column.columnDef.header, header.getContext())}
                    </TableSortLabel>
                  ) : (
                    flexRender(header.column.columnDef.header, header.getContext())
                  )}
                </TableCell>
              ))}
            </TableRow>
          </TableHead>
          <TableBody>
            {table.getRowModel().rows.length === 0 ? (
              <TableRow>
                <TableCell colSpan={columns.length} sx={{ textAlign: 'center', py: 4, color: 'text.secondary' }}>
                  {t('common.noData')}
                </TableCell>
              </TableRow>
            ) : (
              table.getRowModel().rows.map((row) => (
                <TableRow
                  key={row.id}
                  hover
                  sx={{ '&:last-child td': { borderBottom: 0 } }}
                >
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id} sx={{ fontSize: 13, maxWidth: 240, overflow: 'hidden' }}>
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
        <TablePagination
          component="div"
          count={table.getFilteredRowModel().rows.length}
          page={table.getState().pagination.pageIndex}
          rowsPerPage={table.getState().pagination.pageSize}
          rowsPerPageOptions={[25, 50, 100]}
          onPageChange={(_, p) => table.setPageIndex(p)}
          onRowsPerPageChange={(e) => table.setPageSize(Number(e.target.value))}
          labelRowsPerPage={t('common.rowsPerPage', 'Lignes/page')}
        />
      </Paper>
    </Box>
  )
}

// ─── Main page ─────────────────────────────────────────────────────────────

export default function LibraryDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const [tab, setTab] = useState(0)
  const [items, setItems] = useState<LibraryItem[]>([])
  const [tracks, setTracks] = useState<Track[]>([])
  const [albums, setAlbums] = useState<Album[]>([])
  const [artists, setArtists] = useState<Artist[]>([])
  const [artistAlbums, setArtistAlbums] = useState<Album[]>([])
  const [selectedArtist, setSelectedArtist] = useState<string | null>(null)
  const [artistLoading, setArtistLoading] = useState(false)
  const [stats, setStats] = useState<LibraryStats | null>(null)
  const [genres, setGenres] = useState<GenreStat[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [search, setSearch] = useState('')

  const [historyData, setHistoryData] = useState<HistoryPoint[]>([])
  const [itemsWithStats, setItemsWithStats] = useState<ItemWithStats[]>([])
  const [playMethodStats, setPlayMethodStats] = useState<PlayMethodStat[]>([])
  const [lastPlayed, setLastPlayed] = useState<LastPlayedRow[]>([])

  const isMusicLibrary = !loading && items.length > 0 && items.some((item) => item.Type === 'Audio')
  const albumsSynced = albums.length > 0

  useEffect(() => {
    if (!id) return
    const load = (showLoading = true) => {
      if (showLoading) setLoading(true)
      Promise.allSettled([
        api.get(`/stats/getLibraryStats?libraryId=${id}`),
        api.get(`/stats/getLibraryItems?libraryId=${id}`),
        api.get(`/stats/getGenreStats?libraryId=${id}`),
        api.post(`/api/getLibraryHistory`, { libraryid: id }),
        api.post(`/stats/getLibraryItemsWithStats`, { libraryid: id }),
        api.post(`/stats/getLibraryItemsPlayMethodStats`, { libraryid: id }),
        api.post(`/stats/getLibraryLastPlayed`, { libraryid: id }),
        api.get(`/stats/getLibraryAlbums?libraryId=${id}`),
        api.get(`/stats/getLibraryArtists?libraryId=${id}`),
        api.get(`/stats/getLibraryTracks?libraryId=${id}`),
      ]).then(([statsRes, itemsRes, genresRes, historyRes, itemsStatsRes, methodRes, lastPlayedRes, albumsRes, artistsRes, tracksRes]) => {
        if (statsRes.status === 'fulfilled') setStats(statsRes.value.data)
        if (itemsRes.status === 'fulfilled') setItems(itemsRes.value.data ?? [])
        if (genresRes.status === 'fulfilled') setGenres(genresRes.value.data ?? [])
        if (albumsRes.status === 'fulfilled') setAlbums(albumsRes.value.data ?? [])
        if (artistsRes.status === 'fulfilled') setArtists(artistsRes.value.data ?? [])
        if (tracksRes.status === 'fulfilled') setTracks(tracksRes.value.data ?? [])

        if (historyRes.status === 'fulfilled') {
          const raw: { ActivityDateInserted: string }[] = historyRes.value.data?.results ?? []
          const byDate: Record<string, number> = {}
          raw.forEach((row) => {
            try {
              const day = format(parseISO(row.ActivityDateInserted), 'dd/MM/yyyy')
              byDate[day] = (byDate[day] ?? 0) + 1
            } catch { /* ignore */ }
          })
          setHistoryData(Object.entries(byDate).map(([date, plays]) => ({ date, plays })).sort((a, b) => a.date.localeCompare(b.date)))
        }

        if (itemsStatsRes.status === 'fulfilled') {
          const results: ItemWithStats[] = itemsStatsRes.value.data?.results ?? []
          setItemsWithStats([...results].sort((a, b) => b.times_played - a.times_played).slice(0, 10))
        }

        if (methodRes.status === 'fulfilled') setPlayMethodStats(methodRes.value.data?.stats ?? [])
        if (lastPlayedRes.status === 'fulfilled') setLastPlayed(lastPlayedRes.value.data ?? [])
      })
        .catch(() => setError(t('common.loadError')))
        .finally(() => setLoading(false))
    }

    load()
    const interval = window.setInterval(() => load(false), 15000)
    return () => window.clearInterval(interval)
  }, [id, t])

  const handleSelectArtist = async (artistName: string) => {
    setSelectedArtist(artistName)
    setArtistLoading(true)
    try {
      const res = await api.get(`/stats/getArtistAlbums?libraryId=${id}&artist=${encodeURIComponent(artistName)}`)
      setArtistAlbums(res.data ?? [])
    } catch {
      setArtistAlbums([])
    } finally {
      setArtistLoading(false)
    }
  }

  const filteredItems = useMemo(() => {
    const term = search.trim().toLowerCase()
    if (!term) return items
    return items.filter((item) =>
      item.Name.toLowerCase().includes(term) ||
      item.Type.toLowerCase().includes(term) ||
      String(item.ProductionYear ?? '').includes(term)
    )
  }, [items, search])

  // Top played tracks for music stats
  const topTracks = useMemo(() =>
    [...tracks].sort((a, b) => b.PlayCount - a.PlayCount).slice(0, 10).filter((t) => t.PlayCount > 0),
    [tracks]
  )

  const statsTabIndex = isMusicLibrary ? 3 : 2

  return (
    <>
      <PageHeader title={stats?.Name ?? (id ?? '')} />
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

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

      <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 3 }}>
        {isMusicLibrary ? (
          <Tabs value={tab} onChange={(_, v) => { setTab(v as number); setSelectedArtist(null) }}>
            <Tab label={t('library.albums', 'Albums')} />
            <Tab label={t('library.artists', 'Artistes')} />
            <Tab label={t('library.tracks', 'Titres')} />
            <Tab label={t('library.stats', 'Stats')} />
          </Tabs>
        ) : (
          <Tabs value={tab} onChange={(_, v) => setTab(v as number)}>
            <Tab label={t('library.items')} />
            <Tab label={t('library.genres')} />
            <Tab label={t('library.stats', 'Stats')} />
          </Tabs>
        )}
      </Box>

      {/* ── MUSIC: Albums ── */}
      {isMusicLibrary && tab === 0 && (
        albumsSynced
          ? <AlbumGrid albums={albums} libraryId={id!} loading={loading} navigate={navigate} t={t} />
          : !loading && (
            <Alert severity="info">
              {t('library.syncRequired', 'Lancez une synchronisation complète pour afficher les albums et les artistes.')}
            </Alert>
          )
      )}

      {/* ── MUSIC: Artistes ── */}
      {isMusicLibrary && tab === 1 && !albumsSynced && !loading && (
        <Alert severity="info">
          {t('library.syncRequired', 'Lancez une synchronisation complète pour afficher les albums et les artistes.')}
        </Alert>
      )}
      {isMusicLibrary && tab === 1 && albumsSynced && (
        <Box>
          {selectedArtist ? (
            <>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
                <Tooltip title={t('common.back', 'Retour')}>
                  <IconButton size="small" onClick={() => setSelectedArtist(null)}>
                    <ArrowLeft24Regular />
                  </IconButton>
                </Tooltip>
                <Typography variant="h6" sx={{ fontWeight: 600 }}>{selectedArtist}</Typography>
              </Box>
              <AlbumGrid albums={artistAlbums} libraryId={id!} loading={artistLoading} navigate={navigate} t={t} />
            </>
          ) : (
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
                          px: 2, py: 1,
                          borderBottom: '1px solid', borderColor: 'divider',
                          '&:last-child': { borderBottom: 0 },
                          cursor: 'pointer',
                          transition: 'background 150ms',
                          '&:hover': { bgcolor: 'action.hover' },
                          gap: 1.5, alignItems: 'center',
                        }}
                        onClick={() => handleSelectArtist(artist.Name)}
                      >
                        <Box
                          sx={{
                            width: 40, height: 40, borderRadius: '50%',
                            bgcolor: 'rgba(255,255,255,0.07)',
                            display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0,
                          }}
                        >
                          <Person24Regular style={{ fontSize: 20, opacity: 0.6 }} />
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
                          <Chip
                            label={`${artist.PlayCount} ${t('common.plays')}`}
                            size="small"
                            sx={{ fontSize: 11, height: 20, flexShrink: 0 }}
                          />
                        )}
                      </ListItem>
                    ))}
                  </List>
                )}
              </CardContent>
            </Card>
          )}
        </Box>
      )}

      {/* ── MUSIC: Titres ── */}
      {isMusicLibrary && tab === 2 && (
        albumsSynced
          ? <TracksTable tracks={tracks} loading={loading} navigate={navigate} libraryId={id!} t={t} />
          : !loading && (
            <Alert severity="info">
              {t('library.syncRequired', 'Lancez une synchronisation complète pour afficher les titres.')}
            </Alert>
          )
      )}

      {/* ── REGULAR: Items ── */}
      {!isMusicLibrary && tab === 0 && (
        <Box>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
            <TextField
              size="small"
              placeholder={t('common.search')}
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              slotProps={{
                input: {
                  startAdornment: (
                    <InputAdornment position="start">
                      <Search20Regular style={{ fontSize: 16 }} />
                    </InputAdornment>
                  ),
                },
              }}
              sx={{ width: { xs: '100%', sm: 260 } }}
            />
          </Box>
          <Grid container spacing={2}>
            {loading ? (
              Array.from({ length: 12 }).map((_, index) => (
                <Grid key={index} size={{ xs: 6, sm: 4, md: 3, lg: 2 }}>
                  <Skeleton variant="rectangular" height={260} sx={{ borderRadius: 2 }} />
                </Grid>
              ))
            ) : filteredItems.length === 0 ? (
              <Grid size={{ xs: 12 }}>
                <Typography variant="body2" color="text.secondary" sx={{ py: 4, textAlign: 'center' }}>
                  {t('common.noData')}
                </Typography>
              </Grid>
            ) : (
              filteredItems.map((item) => {
                const size = formatSize(item.Size)
                return (
                  <Grid key={item.Id} size={{ xs: 6, sm: 4, md: 3, lg: 2 }}>
                    <Card
                      sx={{
                        height: '100%', overflow: 'hidden', borderRadius: 2,
                        border: '1px solid', borderColor: 'divider',
                        transition: 'transform 160ms ease, border-color 160ms ease',
                        '&:hover': { transform: 'translateY(-3px)', borderColor: 'primary.main' },
                      }}
                    >
                      <CardActionArea onClick={() => navigate(`/libraries/${id}/items/${item.Id}`)} sx={{ height: '100%' }}>
                        <Box
                          sx={{
                            position: 'relative', aspectRatio: '2 / 3',
                            bgcolor: 'rgba(255,255,255,0.04)',
                            display: 'flex', alignItems: 'center', justifyContent: 'center',
                          }}
                        >
                          <VideoClip24Regular style={{ fontSize: 44, opacity: 0.45 }} />
                          <Box
                            component="img"
                            src={`/proxy/Items/Images/Primary/?id=${encodeURIComponent(item.Id)}&fillWidth=360&quality=90`}
                            alt={item.Name}
                            loading="lazy"
                            onError={(e) => { e.currentTarget.style.display = 'none' }}
                            sx={{ position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover' }}
                          />
                          {size && (
                            <Chip
                              label={size}
                              size="small"
                              sx={{
                                position: 'absolute', right: 6, bottom: 6, height: 20, fontSize: 10,
                                bgcolor: 'primary.main', color: 'primary.contrastText',
                              }}
                            />
                          )}
                        </Box>
                        <CardContent sx={{ p: 1.25, '&:last-child': { pb: 1.25 } }}>
                          <Typography variant="body2" sx={{ fontWeight: 600, lineHeight: 1.25 }} title={item.Name}>
                            {item.Name}
                          </Typography>
                          <Box sx={{ display: 'flex', gap: 0.75, mt: 0.75, flexWrap: 'wrap' }}>
                            {item.ProductionYear && (
                              <Typography variant="caption" color="text.secondary">{item.ProductionYear}</Typography>
                            )}
                            {item.CommunityRating && (
                              <Typography variant="caption" color="warning.main">
                                ★ {item.CommunityRating.toFixed(1)}
                              </Typography>
                            )}
                            {item.PlayCount > 0 && (
                              <Typography variant="caption" color="text.secondary">
                                {item.PlayCount} {t('common.plays')}
                              </Typography>
                            )}
                          </Box>
                        </CardContent>
                      </CardActionArea>
                    </Card>
                  </Grid>
                )
              })
            )}
          </Grid>
        </Box>
      )}

      {/* ── REGULAR: Genres ── */}
      {!isMusicLibrary && tab === 1 && (
        <Grid container spacing={2}>
          <Grid size={{ xs: 12, md: 6 }}>
            <ChartCard title={t('library.genreDistribution')} loading={loading} empty={genres.length === 0} height={320}>
              <PieChart
                series={[{
                  data: genres.map((g, i) => ({ id: i, value: g.Count, label: g.Genre, color: COLORS[i % COLORS.length] })),
                  outerRadius: 120, paddingAngle: 2, cornerRadius: 3,
                }]}
                height={320}
                sx={{ width: '100%' }}
              />
            </ChartCard>
          </Grid>
          <Grid size={{ xs: 12, md: 6 }}>
            <Card>
              <CardContent>
                <Typography variant="subtitle1" sx={{ fontWeight: 600 }} gutterBottom>{t('library.genreList')}</Typography>
                {loading
                  ? Array.from({ length: 8 }).map((_, i) => <Skeleton key={i} variant="text" sx={{ mb: 0.5 }} />)
                  : (
                    <List dense disablePadding>
                      {genres.map((g) => (
                        <ListItem key={g.Genre} disablePadding sx={{ py: 0.25 }}>
                          <ListItemText primary={g.Genre} slotProps={{ primary: { style: { fontSize: 13 } } }} />
                          <Chip label={g.Count} size="small" sx={{ fontSize: 11, height: 20 }} />
                        </ListItem>
                      ))}
                    </List>
                  )}
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      )}

      {/* ── Stats tab (tab 3 for music, tab 2 for regular) ── */}
      {tab === statsTabIndex && (
        <Grid container spacing={2}>
          {/* Top played tracks (music only) */}
          {isMusicLibrary && topTracks.length > 0 && (
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
                          px: 2, py: 0.75,
                          borderTop: '1px solid', borderColor: 'divider',
                          gap: 1.5, alignItems: 'center',
                          cursor: track.AlbumId ? 'pointer' : 'default',
                          transition: 'background 150ms',
                          '&:hover': track.AlbumId ? { bgcolor: 'action.hover' } : {},
                        }}
                        onClick={() => track.AlbumId && navigate(`/libraries/${id}/albums/${track.AlbumId}`)}
                      >
                        <Typography
                          variant="caption"
                          color="text.secondary"
                          sx={{ minWidth: 20, textAlign: 'right', fontWeight: 700, flexShrink: 0 }}
                        >
                          {i + 1}
                        </Typography>
                        <Box sx={{ flex: 1, minWidth: 0 }}>
                          <Typography variant="body2" sx={{ fontWeight: 500 }} noWrap>{track.Name}</Typography>
                          <Typography variant="caption" color="text.secondary" noWrap>
                            {[track.Artist, track.Album].filter(Boolean).join(' — ')}
                          </Typography>
                        </Box>
                        <Chip
                          label={`${track.PlayCount} ${t('common.plays')}`}
                          size="small"
                          sx={{ fontSize: 11, height: 20, flexShrink: 0 }}
                        />
                      </ListItem>
                    ))}
                  </List>
                </CardContent>
              </Card>
            </Grid>
          )}

          <Grid size={{ xs: 12 }}>
            <ChartCard title={t('library.activityOverTime', 'Activité dans le temps')} loading={loading} empty={historyData.length === 0} height={220}>
              <LineChart
                xAxis={[{ data: historyData.map((d) => d.date), scaleType: 'point' }]}
                series={[{ data: historyData.map((d) => d.plays), area: true, label: t('common.plays'), color: CHART_BAR, showMark: false }]}
                height={220}
                sx={{ width: '100%' }}
                grid={{ horizontal: true }}
                slotProps={{ legend: { hidden: true } }}
              />
            </ChartCard>
          </Grid>

          {!isMusicLibrary && (
            <Grid size={{ xs: 12 }}>
              <ChartCard title={t('library.topItems', 'Items les plus regardés')} loading={loading} empty={itemsWithStats.length === 0} height={280}>
                <BarChart
                  layout="horizontal"
                  yAxis={[{ data: itemsWithStats.map((item) => item.Name), scaleType: 'band' }]}
                  xAxis={[{ label: t('common.plays') }]}
                  series={[{ data: itemsWithStats.map((item) => item.times_played), label: t('common.plays'), color: CHART_BAR }]}
                  height={280}
                  sx={{ width: '100%' }}
                  grid={{ vertical: true }}
                  slotProps={{ legend: { hidden: true } }}
                />
              </ChartCard>
            </Grid>
          )}

          <Grid size={{ xs: 12, md: 6 }}>
            {(() => {
              const totalTranscodes = playMethodStats.reduce((sum, s) => sum + (s.Transcodes ?? 0), 0)
              const totalDirectPlays = playMethodStats.reduce((sum, s) => sum + (s.DirectPlays ?? 0), 0)
              const pieData = [
                { id: 0, value: totalTranscodes, label: t('activity.transcodes', 'Transcodes'), color: COLORS[0] },
                { id: 1, value: totalDirectPlays, label: t('activity.directPlays', 'Direct Plays'), color: COLORS[1] },
              ].filter((d) => d.value > 0)
              return (
                <ChartCard title={t('library.playMethod', 'Méthode de lecture')} loading={loading} empty={pieData.length === 0} height={240}>
                  <PieChart
                    series={[{ data: pieData, innerRadius: 40, outerRadius: 90, paddingAngle: 2, cornerRadius: 3 }]}
                    height={240}
                    sx={{ width: '100%' }}
                  />
                </ChartCard>
              )
            })()}
          </Grid>

          <Grid size={{ xs: 12, md: 6 }}>
            <Card sx={{ height: '100%' }}>
              <CardContent>
                <Typography variant="subtitle1" sx={{ fontWeight: 600 }} gutterBottom>
                  {t('library.lastPlayed', 'Derniers lus')}
                </Typography>
                {loading
                  ? Array.from({ length: 6 }).map((_, i) => <Skeleton key={i} variant="text" sx={{ mb: 0.5 }} />)
                  : lastPlayed.length === 0
                    ? <Typography variant="body2" color="text.secondary" sx={{ py: 2, textAlign: 'center' }}>{t('common.noData')}</Typography>
                    : (
                      <List dense disablePadding>
                        {lastPlayed.map((row, i) => {
                          let dateStr = row.ActivityDateInserted
                          try { dateStr = format(parseISO(row.ActivityDateInserted), 'dd/MM/yyyy HH:mm') } catch { /* keep raw */ }
                          return (
                            <ListItem key={i} disablePadding sx={{ py: 0.5 }}>
                              <ListItemText
                                primary={row.NowPlayingItemName}
                                secondary={`${row.UserName} — ${dateStr}`}
                                slotProps={{
                                  primary: { style: { fontSize: 13, fontWeight: 500 } },
                                  secondary: { style: { fontSize: 11 } },
                                }}
                              />
                            </ListItem>
                          )
                        })}
                      </List>
                    )}
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      )}
    </>
  )
}
