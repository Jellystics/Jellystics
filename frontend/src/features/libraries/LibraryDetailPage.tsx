import { useState, useEffect, useMemo, useCallback } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import {
  Grid, Alert, Card, CardActionArea, CardContent, Typography, Tabs, Tab, Box,
  Chip, List, ListItem, ListItemText, Skeleton, TextField, InputAdornment,
  IconButton, Tooltip, CircularProgress,
  TablePagination, ToggleButtonGroup, ToggleButton,
  Select, MenuItem, FormControl, InputLabel,
} from '@mui/material'
import { createColumnHelper } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'
import { format, parseISO } from 'date-fns'
import { alpha } from '@mui/material/styles'
import { PieChart } from '@mui/x-charts/PieChart'
import { LineChart } from '@mui/x-charts/LineChart'
import { BarChart } from '@mui/x-charts/BarChart'
import PageHeader from '@/shared/components/PageHeader/PageHeader'
import StatCard from '@/shared/components/StatCard/StatCard'
import ChartCard from '@/shared/components/ChartCard/ChartCard'
import api from '@/lib/axios'
import type { LibraryItem, LibraryStats, GenreStat, MusicTrack, MusicAlbum, MusicArtist } from '@/shared/types/library'
import type { Activity } from '@/shared/types/activity'
import {
  Play24Regular, Clock24Regular, Star24Regular,
  Search20Regular, VideoClip24Regular, MusicNote224Regular,
  Person24Regular, ArrowLeft24Regular, Grid24Regular, TableSimple24Regular, ArrowSync24Regular,
} from '@fluentui/react-icons'
import { formatWatchTime } from '@/shared/utils/formatWatchTime'
import { formatTicks, formatDuration } from '@/shared/utils/formatTicks'
import { formatSize } from '@/shared/utils/formatSize'
import DataTable, { type FilterDef } from '@/shared/components/DataTable/DataTable'
import { getDateLocale } from '@/lib/dateLocale'
import { useChartColors } from '@/lib/chartColors'
import { getActivityImageUrl } from '@/shared/utils/activityImage'

type Track = MusicTrack
type Album = MusicAlbum
type Artist = MusicArtist




type HistoryPoint = { date: string; plays: number }
type ItemWithStats = { Name: string; times_played: number; total_play_time: number }
type PlayMethodStat = { Key: string; Transcodes: number; DirectPlays: number }
type LastPlayedRow = { NowPlayingItemName: string; ActivityDateInserted: string; UserName: string }

type TimeToWatchData = {
  avgDaysToWatch: number
  medianDaysToWatch: number
  distribution: { bucket: string; count: number }[]
  slowestItems: { id: string; name: string; type: string; daysToWatch: number; dateAdded: string; firstWatched: string }[]
  fastestItems: { id: string; name: string; type: string; daysToWatch: number; dateAdded: string; firstWatched: string }[]
}

type UnwatchedContentData = {
  summary: { totalItems: number; unwatchedItems: number; unwatchedPercent: number; byType: { type: string; count: number }[] }
  items: { current_page: number; pages: number; size: number; results: { id: string; name: string; type: string; dateAdded: string; genres: string[]; libraryName: string }[] }
}

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
                transition: 'border-color 160ms ease',
                '&:hover': { borderColor: 'primary.main' },
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
                    overflow: 'hidden',
                  }}
                >
                  <MusicNote224Regular style={{ fontSize: 44, opacity: 0.45 }} />
                  <Box
                    component="img"
                    src={`/proxy/Items/Images/Primary/?id=${encodeURIComponent(album.Id)}&fillWidth=360&quality=90`}
                    alt={album.Name}
                    loading="lazy"
                    onError={(e) => { e.currentTarget.style.display = 'none' }}
                    sx={{
                      position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover',
                      transition: 'transform 300ms ease',
                      '.MuiCard-root:hover &': { transform: 'scale(1.06)' },
                    }}
                  />
                </Box>
                <CardContent sx={{ p: 1.25, '&:last-child': { pb: 1.25 } }}>
                  <Typography variant="body2" sx={{ fontWeight: 600, lineHeight: 1.25 }} noWrap title={album.Name}>
                    {album.Name}
                  </Typography>
                  {album.AlbumArtist && (
                    <Typography variant="caption" color="text.secondary" noWrap sx={{ display: 'block', mt: 0.25 }}>
                      {album.AlbumArtist}
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
  const columns = useMemo(() => [
    colHelper.accessor('IndexNumber', {
      header: '#',
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
    colHelper.accessor('AlbumName', {
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
      cell: (info) => (
        <Typography variant="caption" color="text.secondary">
          {formatTicks(info.getValue())}
        </Typography>
      ),
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
    <DataTable
      data={tracks}
      columns={columns}
      loading={loading}
      searchPlaceholder={`${t('common.search')}…`}
      defaultPageSize={50}
    />
  )
}

// ─── Genre Distribution ─────────────────────────────────────────────────────

const PIE_LIMITS = [8, 12, 20, 9999] as const

function GenreDistributionCard({ genres, loading, t }: {
  genres: GenreStat[]
  loading: boolean
  t: (k: string, fb?: string) => string
}) {
  const COLORS = useChartColors()
  const [pieLimit, setPieLimit] = useState<number>(8)

  const pieGenres = genres.slice(0, pieLimit)
  const total = genres.reduce((s, g) => s + g.Count, 0)

  return (
    <Card>
      <CardContent>
        <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 2 }}>
          {t('library.genreDistribution')}
        </Typography>

        {loading ? (
          <Grid container spacing={2}>
            <Grid size={{ xs: 12, md: 6 }}>
              <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 2 }}>
                <Skeleton variant="circular" width={260} height={260} />
                <Skeleton variant="rectangular" width={180} height={36} sx={{ borderRadius: 1 }} />
              </Box>
            </Grid>
            <Grid size={{ xs: 12, md: 6 }}>
              {Array.from({ length: 10 }).map((_, i) => (
                <Skeleton key={i} variant="text" sx={{ mb: 0.75, height: 28 }} />
              ))}
            </Grid>
          </Grid>
        ) : genres.length === 0 ? (
          <Typography variant="body2" color="text.secondary" sx={{ py: 3, textAlign: 'center' }}>
            {t('common.noData')}
          </Typography>
        ) : (
          <Grid container spacing={3}>
            {/* Left — Pie + dropdown */}
            <Grid size={{ xs: 12, md: 6 }}>
              <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 2 }}>
                <PieChart
                  width={260}
                  height={260}
                  series={[{
                    data: pieGenres.map((g, i) => ({
                      id: i,
                      value: g.Count,
                      label: g.Genre,
                      color: COLORS[i % COLORS.length],
                    })),
                    outerRadius: 110,
                    paddingAngle: 2,
                    cornerRadius: 3,
                    highlightScope: { fade: 'global', highlight: 'item' },
                  }]}
                  slotProps={{ legend: { hidden: true } as any }}
                />

                <FormControl size="small" sx={{ minWidth: 180 }}>
                  <InputLabel>{t('library.showGenres', 'Genres affichés')}</InputLabel>
                  <Select
                    value={pieLimit}
                    label={t('library.showGenres', 'Genres affichés')}
                    onChange={(e) => setPieLimit(Number(e.target.value))}
                  >
                    {PIE_LIMITS.map((n) => (
                      <MenuItem key={n} value={n}>
                        {n >= 9999 ? t('library.allGenres', 'Tous les genres') : `Top ${n}`}
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
              </Box>
            </Grid>

            {/* Right — full genre list scrollable */}
            <Grid size={{ xs: 12, md: 6 }}>
              <Box
                sx={{
                  height: 320,
                  overflowY: 'auto',
                  scrollbarWidth: 'thin',
                  pr: 0.5,
                }}
              >
                {genres.map((g, i) => {
                  const pct = total > 0 ? (g.Count / total) * 100 : 0
                  return (
                    <Box
                      key={g.Genre}
                      sx={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: 1.25,
                        py: 0.65,
                        borderBottom: '1px solid',
                        borderColor: 'divider',
                        '&:last-child': { borderBottom: 0 },
                      }}
                    >
                      <Box
                        sx={{
                          width: 10, height: 10, borderRadius: '50%',
                          bgcolor: COLORS[i % COLORS.length],
                          flexShrink: 0,
                        }}
                      />
                      <Typography variant="body2" sx={{ flex: 1, fontWeight: 500 }} noWrap>
                        {g.Genre}
                      </Typography>
                      {/* Progress bar */}
                      <Box sx={{ width: 60, flexShrink: 0, display: { xs: 'none', sm: 'block' } }}>
                        <Box sx={{ height: 4, borderRadius: 2, bgcolor: 'action.hover', overflow: 'hidden' }}>
                          <Box
                            sx={{
                              height: '100%',
                              borderRadius: 2,
                              width: `${pct}%`,
                              bgcolor: COLORS[i % COLORS.length],
                            }}
                          />
                        </Box>
                      </Box>
                      <Typography
                        variant="caption"
                        color="text.secondary"
                        sx={{ flexShrink: 0, minWidth: 28, textAlign: 'right', fontVariantNumeric: 'tabular-nums' }}
                      >
                        {g.Count}
                      </Typography>
                    </Box>
                  )
                })}
              </Box>
            </Grid>
          </Grid>
        )}
      </CardContent>
    </Card>
  )
}

// ─── Items grid with pagination ────────────────────────────────────────────

const GRID_PAGE_SIZES = [30, 50, 100] as const

function ItemsGridView({ items, loading, navigate, libraryId, t, cols = 6 }: {
  items: LibraryItem[]
  loading: boolean
  navigate: ReturnType<typeof useNavigate>
  libraryId: string
  t: (k: string, fb?: string) => string
  cols?: 6 | 8 | 10
}) {
  const [page, setPage] = useState(0)
  const [pageSize, setPageSize] = useState<typeof GRID_PAGE_SIZES[number]>(30)
  const paged = items.slice(page * pageSize, (page + 1) * pageSize)

  // Reset to page 0 when items or pageSize change
  useEffect(() => { setPage(0) }, [items.length, pageSize])

  return (
    <Box>
      <Box sx={{ display: 'grid', gridTemplateColumns: `repeat(${cols}, 1fr)`, gap: 2, alignItems: 'start' }}>
        {loading ? (
          Array.from({ length: pageSize }).map((_, index) => (
            <Skeleton key={index} variant="rectangular" height={260} sx={{ borderRadius: 2 }} />
          ))
        ) : paged.length === 0 ? (
          <Box sx={{ gridColumn: `1 / -1`, py: 4, textAlign: 'center' }}>
            <Typography variant="body2" color="text.secondary">
              {t('common.noData')}
            </Typography>
          </Box>
        ) : (
          paged.map((item) => {
            const size = formatSize(item.Size)
            return (
              <Card
                key={item.Id}
                sx={{
                  overflow: 'hidden', borderRadius: 2,
                  border: '1px solid', borderColor: 'divider',
                  transition: 'border-color 160ms ease',
                  '&:hover': { borderColor: 'primary.main' },
                }}
              >
                <CardActionArea onClick={() => navigate(`/libraries/${libraryId}/items/${item.Id}`)}>
                  <Box
                    sx={{
                      position: 'relative', aspectRatio: '2 / 3',
                      bgcolor: 'rgba(255,255,255,0.04)',
                      display: 'flex', alignItems: 'center', justifyContent: 'center',
                      overflow: 'hidden',
                    }}
                  >
                    <VideoClip24Regular style={{ fontSize: 44, opacity: 0.45 }} />
                    <Box
                      component="img"
                      src={`/proxy/Items/Images/Primary/?id=${encodeURIComponent(item.Id)}&fillWidth=360&quality=90`}
                      alt={item.Name}
                      loading="lazy"
                      onError={(e) => { e.currentTarget.style.display = 'none' }}
                      sx={{
                        position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover',
                        transition: 'transform 300ms ease',
                        '.MuiCard-root:hover &': { transform: 'scale(1.06)' },
                      }}
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
            )
          })
        )}
      </Box>

      {!loading && (
        <TablePagination
          component="div"
          count={items.length}
          page={page}
          rowsPerPage={pageSize}
          rowsPerPageOptions={[...GRID_PAGE_SIZES]}
          onPageChange={(_, p) => { setPage(p); window.scrollTo({ top: 0, behavior: 'smooth' }) }}
          onRowsPerPageChange={(e) => setPageSize(Number(e.target.value) as typeof GRID_PAGE_SIZES[number])}
          labelRowsPerPage={t('common.rowsPerPage', 'Lignes/page')}
        />
      )}
    </Box>
  )
}

// ─── Items table with poster preview ───────────────────────────────────────

const itemColHelper = createColumnHelper<LibraryItem>()

function ItemsTableView({ items, loading, navigate, libraryId, t }: {
  items: LibraryItem[]
  loading: boolean
  navigate: ReturnType<typeof useNavigate>
  libraryId: string
  t: (k: string, fb?: string) => string
}) {
  const [hovered, setHovered] = useState<LibraryItem | null>(null)

  const columns = useMemo(() => [
    itemColHelper.display({
      id: 'poster',
      size: 52,
      header: () => null,
      meta: { hideFromColumnsMenu: true },
      cell: (info) => {
        const item = info.row.original
        return (
          <Box
            sx={{
              width: 36, height: 52, borderRadius: 1, overflow: 'hidden',
              bgcolor: 'rgba(255,255,255,0.06)', flexShrink: 0, position: 'relative',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
            }}
          >
            <VideoClip24Regular style={{ fontSize: 16, opacity: 0.35, position: 'absolute' }} />
            <Box
              component="img"
              src={`/proxy/Items/Images/Primary/?id=${encodeURIComponent(item.Id)}&fillWidth=80&quality=80`}
              alt={item.Name}
              loading="lazy"
              onError={(e) => { e.currentTarget.style.display = 'none' }}
              sx={{ position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover' }}
            />
          </Box>
        )
      },
    }),
    itemColHelper.accessor('Name', {
      header: t('library.title', 'Titre'),
      cell: (info) => {
        const item = info.row.original
        return (
          <Box>
            <Typography variant="body2" sx={{ fontWeight: 600, lineHeight: 1.3 }}>
              {info.getValue()}
            </Typography>
            {item.SeriesName && (
              <Typography variant="caption" color="text.secondary" sx={{ display: 'block' }} noWrap>
                {item.SeriesName}
                {item.ParentIndexNumber != null && item.IndexNumber != null
                  ? ` — S${item.ParentIndexNumber}E${item.IndexNumber}`
                  : ''}
              </Typography>
            )}
          </Box>
        )
      },
    }),
    itemColHelper.accessor('Type', {
      header: t('library.type', 'Type'),
      size: 90,
      cell: (info) => (
        <Chip label={info.getValue()} size="small" variant="outlined" sx={{ fontSize: 11, height: 20 }} />
      ),
    }),
    itemColHelper.accessor('ProductionYear', {
      header: t('library.year', 'Année'),
      size: 70,
      cell: (info) => (
        <Typography variant="caption" color="text.secondary">
          {info.getValue() ?? '—'}
        </Typography>
      ),
    }),
    itemColHelper.accessor('CommunityRating', {
      header: '★',
      size: 60,
      cell: (info) => {
        const v = info.getValue()
        return v
          ? <Typography variant="caption" color="warning.main" sx={{ fontWeight: 600 }}>★ {v.toFixed(1)}</Typography>
          : <Typography variant="caption" color="text.disabled">—</Typography>
      },
    }),
    itemColHelper.accessor('PlayCount', {
      header: t('common.plays'),
      size: 80,
      cell: (info) => {
        const v = info.getValue()
        return v > 0
          ? <Chip label={v} size="small" sx={{ fontSize: 11, height: 20 }} />
          : <Typography variant="caption" color="text.disabled">0</Typography>
      },
    }),
    itemColHelper.accessor('Size', {
      header: t('library.size', 'Taille'),
      size: 90,
      cell: (info) => (
        <Typography variant="caption" color="text.secondary">
          {formatSize(info.getValue()) ?? '—'}
        </Typography>
      ),
    }),
  ], [t])

  return (
    <Box sx={{ display: 'flex', gap: 2, alignItems: 'flex-start' }}>
      {/* Table */}
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <DataTable
          data={items}
          columns={columns}
          loading={loading}
          searchable={false}
          onRowClick={(row) => navigate(`/libraries/${libraryId}/items/${row.Id}`)}
          onRowHover={setHovered}
        />
      </Box>

      {/* Poster preview panel */}
      <Box
        sx={{
          width: 200,
          flexShrink: 0,
          position: 'sticky',
          top: 80,
          display: { xs: 'none', lg: 'block' },
        }}
        onMouseLeave={() => setHovered(null)}
      >
        <Card variant="outlined" sx={{ borderRadius: 2, overflow: 'hidden', transition: 'opacity 200ms', opacity: hovered ? 1 : 0.35 }}>
          <Box
            sx={{
              position: 'relative', aspectRatio: '2 / 3',
              bgcolor: 'rgba(255,255,255,0.05)',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
            }}
          >
            <VideoClip24Regular style={{ fontSize: 48, opacity: 0.25 }} />
            {hovered && (
              <Box
                component="img"
                key={hovered.Id}
                src={`/proxy/Items/Images/Primary/?id=${encodeURIComponent(hovered.Id)}&fillWidth=400&quality=90`}
                alt={hovered.Name}
                onError={(e) => { e.currentTarget.style.display = 'none' }}
                sx={{ position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover', transition: 'opacity 200ms' }}
              />
            )}
          </Box>
          <CardContent sx={{ p: 1.5, '&:last-child': { pb: 1.5 } }}>
            {hovered ? (
              <>
                <Typography variant="body2" sx={{ fontWeight: 700, lineHeight: 1.3, mb: 0.75 }}>
                  {hovered.Name}
                </Typography>
                <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap', mb: 0.75 }}>
                  <Chip label={hovered.Type} size="small" variant="outlined" sx={{ fontSize: 10, height: 18 }} />
                  {hovered.ProductionYear && (
                    <Chip label={hovered.ProductionYear} size="small" variant="outlined" sx={{ fontSize: 10, height: 18 }} />
                  )}
                </Box>
                {hovered.CommunityRating != null && (
                  <Typography variant="caption" color="warning.main" sx={{ display: 'block', fontWeight: 600, mb: 0.5 }}>
                    ★ {hovered.CommunityRating.toFixed(1)}
                  </Typography>
                )}
                <Typography variant="caption" color="text.secondary" sx={{ display: 'block' }}>
                  {hovered.PlayCount > 0
                    ? `${hovered.PlayCount} ${t('common.plays')}`
                    : t('library.neverPlayed', 'Jamais lu')}
                </Typography>
                {hovered.Size != null && formatSize(hovered.Size) && (
                  <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.25 }}>
                    {formatSize(hovered.Size)}
                  </Typography>
                )}
              </>
            ) : (
              <Typography variant="caption" color="text.disabled" sx={{ display: 'block', textAlign: 'center', py: 1 }}>
                {t('library.hoverToPreview', 'Survolez une ligne')}
              </Typography>
            )}
          </CardContent>
        </Card>
      </Box>
    </Box>
  )
}

// ─── Activity tab ───────────────────────────────────────────────────────────

const actColHelper = createColumnHelper<Activity>()


function LibraryActivityTab({ data, loading, onRefresh, t }: {
  data: Activity[]
  loading: boolean
  onRefresh: () => void
  t: (k: string, fb?: string) => string
}) {
  const columns = useMemo(() => [
    actColHelper.accessor('UserName', {
      header: t('activity.user', 'User'),
      cell: (info) => {
        const row = info.row.original
        return (
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Box
              sx={{
                width: 32, height: 32, borderRadius: '50%', overflow: 'hidden', flexShrink: 0,
                bgcolor: 'rgba(255,255,255,0.08)', display: 'flex', alignItems: 'center', justifyContent: 'center', position: 'relative',
              }}
            >
              <Typography variant="caption" sx={{ fontWeight: 700, fontSize: 11, lineHeight: 1, position: 'absolute' }}>
                {row.UserName.slice(0, 2).toUpperCase()}
              </Typography>
              <Box
                component="img"
                src={`/proxy/Users/${encodeURIComponent(row.UserId)}/Images/Primary?fillWidth=32&quality=80`}
                alt={row.UserName}
                loading="lazy"
                onError={(e) => { e.currentTarget.style.display = 'none' }}
                sx={{ position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover' }}
              />
            </Box>
            <Typography variant="body2" noWrap>{row.UserName}</Typography>
          </Box>
        )
      },
    }),
    actColHelper.accessor('NowPlayingItemName', {
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
    actColHelper.accessor('Client', {
      header: t('activity.client', 'Client'),
      cell: (info) => <Typography variant="body2" noWrap>{info.getValue()}</Typography>,
    }),
    actColHelper.accessor('DeviceName', {
      header: t('activity.device', 'Device'),
      cell: (info) => <Typography variant="body2" noWrap>{info.getValue()}</Typography>,
    }),
    actColHelper.accessor('PlayMethod', {
      header: t('activity.method', 'Method'),
      cell: (info) => {
        const v = info.getValue()
        return v
          ? <Chip label={v} size="small" variant="outlined" sx={{ fontSize: 11, height: 20 }} />
          : <Typography variant="caption" color="text.disabled">—</Typography>
      },
    }),
    actColHelper.accessor('ActivityDateInserted', {
      header: t('activity.date', 'Date'),
      cell: (info) => {
        const v = info.getValue()
        try {
          return (
            <Typography variant="body2" sx={{ whiteSpace: 'nowrap' }}>
              {format(parseISO(v), 'dd/MM/yyyy HH:mm', { locale: getDateLocale() })}
            </Typography>
          )
        } catch {
          return <Typography variant="body2">{v}</Typography>
        }
      },
    }),
    actColHelper.accessor('PlayDuration', {
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
    actColHelper.accessor('RemoteEndPoint', {
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

// ─── Main page ─────────────────────────────────────────────────────────────

export default function LibraryDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const CHART_COLORS = useChartColors()
  const CHART_BAR = CHART_COLORS[0]
  const [searchParams, setSearchParams] = useSearchParams()

  // view ↔ tab index mappings
  const REGULAR_VIEWS = ['items', 'stats', 'activity'] as const
  const MUSIC_VIEWS   = ['albums', 'artists', 'tracks', 'stats', 'activity'] as const

  const viewToTab = (view: string, music: boolean): number => {
    const list = music ? MUSIC_VIEWS : REGULAR_VIEWS
    const idx = list.indexOf(view as never)
    return idx >= 0 ? idx : 0
  }
  const tabToView = (idx: number, music: boolean): string =>
    (music ? MUSIC_VIEWS : REGULAR_VIEWS)[idx] ?? (music ? MUSIC_VIEWS[0] : REGULAR_VIEWS[0])
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
  const [viewMode, setViewMode] = useState<'grid' | 'table'>('grid')
  const [gridCols, setGridCols] = useState<6 | 8 | 10>(8)

  const [historyData, setHistoryData] = useState<HistoryPoint[]>([])
  const [activityHistory, setActivityHistory] = useState<Activity[]>([])
  const [itemsWithStats, setItemsWithStats] = useState<ItemWithStats[]>([])
  const [playMethodStats, setPlayMethodStats] = useState<PlayMethodStat[]>([])
  const [lastPlayed, setLastPlayed] = useState<LastPlayedRow[]>([])
  const [timeToWatch, setTimeToWatch] = useState<TimeToWatchData | null>(null)
  const [timeToWatchLoading, setTimeToWatchLoading] = useState(false)
  const [unwatchedContent, setUnwatchedContent] = useState<UnwatchedContentData | null>(null)
  const [unwatchedLoading, setUnwatchedLoading] = useState(false)

  const isMusicLibrary = !loading && (tracks.length > 0 || artists.length > 0 || albums.length > 0)
  const albumsSynced = albums.length > 0

  const load = useCallback((showLoading = true) => {
    if (!id) return
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
        const raw: Activity[] = historyRes.value.data?.results ?? []
        const byDate: Record<string, number> = {}
        raw.forEach((row) => {
          try {
            const day = format(parseISO(row.ActivityDateInserted), 'dd/MM/yyyy')
            byDate[day] = (byDate[day] ?? 0) + 1
          } catch { /* ignore */ }
        })
        setHistoryData(Object.entries(byDate).map(([date, plays]) => ({ date, plays })).sort((a, b) => a.date.localeCompare(b.date)))
        setActivityHistory(raw)
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
  }, [id, t])

  useEffect(() => {
    load()
    const interval = window.setInterval(() => load(false), 15000)
    return () => window.clearInterval(interval)
  }, [load])

  // Fetch Time to Watch data
  useEffect(() => {
    if (!id) return
    setTimeToWatchLoading(true)
    api.get(`/stats/getTimeToWatch?libraryId=${id}&limit=10`)
      .then((res) => setTimeToWatch(res.data))
      .catch(() => setTimeToWatch(null))
      .finally(() => setTimeToWatchLoading(false))
  }, [id])

  // Fetch Unwatched Content data
  useEffect(() => {
    if (!id) return
    setUnwatchedLoading(true)
    api.get(`/stats/getUnwatchedContent?libraryId=${id}&pageSize=10`)
      .then((res) => setUnwatchedContent(res.data))
      .catch(() => setUnwatchedContent(null))
      .finally(() => setUnwatchedLoading(false))
  }, [id])

  const handleSelectArtist = async (artistId: string, artistName: string) => {
    setSelectedArtist(artistName)
    setArtistLoading(true)
    try {
      const res = await api.get(`/stats/getArtistAlbums?libraryId=${id}&artistId=${encodeURIComponent(artistId)}`)
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

  const statsTabIndex = isMusicLibrary ? 3 : 1
  const activityTabIndex = isMusicLibrary ? 4 : 2

  const tab = viewToTab(searchParams.get('view') ?? '', isMusicLibrary)
  const setTab = (idx: number) => {
    setSearchParams({ view: tabToView(idx, isMusicLibrary) }, { replace: true })
  }

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
      <PageHeader title={stats?.Name ?? (id ?? '')} onRefresh={() => load(false)} loading={loading} />
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
          <Tabs value={tab} onChange={(_, v) => { setTab(v as number); setSelectedArtist(null) }} variant="scrollable" scrollButtons="auto">
            <Tab label={t('library.albums', 'Albums')} />
            <Tab label={t('library.artists', 'Artistes')} />
            <Tab label={t('library.tracks', 'Titres')} />
            <Tab label={t('library.stats', 'Stats')} />
            <Tab label={t('library.activityHistory', "Historique d'activité")} />
          </Tabs>
        ) : (
          <Tabs value={tab} onChange={(_, v) => setTab(v as number)} variant="scrollable" scrollButtons="auto">
            <Tab label={t('library.items')} />
            <Tab label={t('library.stats', 'Stats')} />
            <Tab label={t('library.activityHistory', "Historique d'activité")} />
          </Tabs>
        )}
      </Box>

      {/* ── MUSIC: Albums ── */}
      {isMusicLibrary && tab === 0 && (
        albumsSynced
          ? <AlbumGrid albums={albums} libraryId={id!} loading={loading} navigate={navigate} t={(k, fb) => t(k, { defaultValue: fb })} />
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
              <AlbumGrid albums={artistAlbums} libraryId={id!} loading={artistLoading} navigate={navigate} t={(k, fb) => t(k, { defaultValue: fb })} />
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
                        onClick={() => handleSelectArtist(artist.Id, artist.Name)}
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
          ? <TracksTable tracks={tracks} loading={loading} navigate={navigate} libraryId={id!} t={(k, fb) => t(k, { defaultValue: fb })} />
          : !loading && (
            <Alert severity="info">
              {t('library.syncRequired', 'Lancez une synchronisation complète pour afficher les titres.')}
            </Alert>
          )
      )}

      {/* ── REGULAR: Items ── */}
      {!isMusicLibrary && tab === 0 && (
        <Box>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2, flexWrap: 'wrap' }}>
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
            <Typography variant="caption" color="text.secondary" sx={{ ml: 0.5 }}>
              {filteredItems.length} {t('common.items')}
            </Typography>
            <Box sx={{ ml: 'auto', display: 'flex', gap: 1, alignItems: 'center' }}>
              <Tooltip title={t('common.refresh')}>
                <span>
                  <IconButton size="small" onClick={() => load()} disabled={loading}>
                    {loading ? <CircularProgress size={16} /> : <ArrowSync24Regular style={{ fontSize: 18 }} />}
                  </IconButton>
                </span>
              </Tooltip>
              {viewMode === 'grid' && (
                <ToggleButtonGroup
                  value={gridCols}
                  exclusive
                  onChange={(_, v) => v && setGridCols(v)}
                  size="small"
                >
                  {([6, 8, 10] as const).map((n) => (
                    <ToggleButton key={n} value={n} sx={{ px: 1.25, fontSize: 12, fontWeight: 600, minWidth: 32 }}>
                      {n}
                    </ToggleButton>
                  ))}
                </ToggleButtonGroup>
              )}
              <ToggleButtonGroup
                value={viewMode}
                exclusive
                onChange={(_, v) => v && setViewMode(v)}
                size="small"
              >
                <ToggleButton value="grid" aria-label="grid view">
                  <Grid24Regular style={{ fontSize: 18 }} />
                </ToggleButton>
                <ToggleButton value="table" aria-label="table view">
                  <TableSimple24Regular style={{ fontSize: 18 }} />
                </ToggleButton>
              </ToggleButtonGroup>
            </Box>
          </Box>

          {viewMode === 'grid' ? (
            <ItemsGridView items={filteredItems} loading={loading} navigate={navigate} libraryId={id!} t={(k, fb) => t(k, { defaultValue: fb })} cols={gridCols} />
          ) : (
            <ItemsTableView items={filteredItems} loading={loading} navigate={navigate} libraryId={id!} t={(k, fb) => t(k, { defaultValue: fb })} />
          )}
        </Box>
      )}

      {/* ── Activity tab ── */}
      {tab === activityTabIndex && (
        <LibraryActivityTab data={activityHistory} loading={loading} onRefresh={() => load(false)} t={(k, fb) => t(k, { defaultValue: fb })} />
      )}

      {/* ── Stats tab (tab 3 for music, tab 1 for regular) ── */}
      {tab === statsTabIndex && (
        <Grid container spacing={2}>
          {/* Genre Distribution (non-music only) */}
          {!isMusicLibrary && (
            <Grid size={{ xs: 12 }}>
              <GenreDistributionCard genres={genres} loading={loading} t={(k, fb) => t(k, { defaultValue: fb })} />
            </Grid>
          )}

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
                            {[track.Artist, track.AlbumName].filter(Boolean).join(' — ')}
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
                slotProps={{ legend: { hidden: true } as any }}
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
                  slotProps={{ legend: { hidden: true } as any }}
                />
              </ChartCard>
            </Grid>
          )}

          <Grid size={{ xs: 12, md: 6 }}>
            {(() => {
              const totalTranscodes = playMethodStats.reduce((sum, s) => sum + (s.Transcodes ?? 0), 0)
              const totalDirectPlays = playMethodStats.reduce((sum, s) => sum + (s.DirectPlays ?? 0), 0)
              const pieData = [
                { id: 0, value: totalTranscodes, label: t('activity.transcodes', 'Transcodes'), color: CHART_COLORS[0] },
                { id: 1, value: totalDirectPlays, label: t('activity.directPlays', 'Direct Plays'), color: CHART_COLORS[1] },
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

          {/* Time to Watch */}
          <Grid size={{ xs: 12, md: 6 }}>
            <Card sx={{ height: '100%' }}>
              <CardContent>
                <Typography variant="subtitle1" sx={{ fontWeight: 700 }} gutterBottom>
                  {t('library.timeToWatch', 'Time to Watch')}
                </Typography>
                {timeToWatchLoading ? (
                  Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} variant="text" sx={{ mb: 0.5 }} />)
                ) : !timeToWatch ? (
                  <Typography variant="body2" color="text.secondary" sx={{ py: 2, textAlign: 'center' }}>{t('common.noData')}</Typography>
                ) : (
                  <>
                    <Box sx={{ display: 'flex', gap: 3, mb: 2 }}>
                      <Box sx={{ textAlign: 'center', flex: 1, p: 1.5, borderRadius: 1, bgcolor: (theme) => alpha(theme.palette.primary.main, 0.08) }}>
                        <Typography variant="h5" sx={{ fontWeight: 700 }}>{timeToWatch.avgDaysToWatch.toFixed(1)}</Typography>
                        <Typography variant="caption" color="text.secondary">{t('library.avgDays', 'Avg days')}</Typography>
                      </Box>
                      <Box sx={{ textAlign: 'center', flex: 1, p: 1.5, borderRadius: 1, bgcolor: (theme) => alpha(theme.palette.primary.main, 0.08) }}>
                        <Typography variant="h5" sx={{ fontWeight: 700 }}>{timeToWatch.medianDaysToWatch.toFixed(1)}</Typography>
                        <Typography variant="caption" color="text.secondary">{t('library.medianDays', 'Median days')}</Typography>
                      </Box>
                    </Box>

                    {timeToWatch.distribution.length > 0 && (
                      <Box sx={{ mb: 2 }}>
                        <BarChart
                          xAxis={[{ data: timeToWatch.distribution.map((d) => d.bucket), scaleType: 'band' }]}
                          series={[{ data: timeToWatch.distribution.map((d) => d.count), label: t('common.count', 'Count'), color: CHART_BAR }]}
                          height={180}
                          sx={{ width: '100%' }}
                          grid={{ horizontal: true }}
                          slotProps={{ legend: { hidden: true } as any }}
                        />
                      </Box>
                    )}

                    <Grid container spacing={2}>
                      <Grid size={{ xs: 6 }}>
                        <Typography variant="caption" sx={{ fontWeight: 700, display: 'block', mb: 0.5 }} color="success.main">
                          {t('library.fastestWatched', 'Fastest watched')}
                        </Typography>
                        <List dense disablePadding>
                          {timeToWatch.fastestItems.slice(0, 5).map((item) => (
                            <ListItem key={item.id} disablePadding sx={{ py: 0.25 }}>
                              <ListItemText
                                primary={item.name}
                                secondary={`${item.daysToWatch} ${t('library.days', 'days')}`}
                                slotProps={{
                                  primary: { style: { fontSize: 12, fontWeight: 500 } },
                                  secondary: { style: { fontSize: 11 } },
                                }}
                              />
                            </ListItem>
                          ))}
                        </List>
                      </Grid>
                      <Grid size={{ xs: 6 }}>
                        <Typography variant="caption" sx={{ fontWeight: 700, display: 'block', mb: 0.5 }} color="warning.main">
                          {t('library.slowestWatched', 'Slowest watched')}
                        </Typography>
                        <List dense disablePadding>
                          {timeToWatch.slowestItems.slice(0, 5).map((item) => (
                            <ListItem key={item.id} disablePadding sx={{ py: 0.25 }}>
                              <ListItemText
                                primary={item.name}
                                secondary={`${item.daysToWatch} ${t('library.days', 'days')}`}
                                slotProps={{
                                  primary: { style: { fontSize: 12, fontWeight: 500 } },
                                  secondary: { style: { fontSize: 11 } },
                                }}
                              />
                            </ListItem>
                          ))}
                        </List>
                      </Grid>
                    </Grid>
                  </>
                )}
              </CardContent>
            </Card>
          </Grid>

          {/* Unwatched Content Summary */}
          <Grid size={{ xs: 12, md: 6 }}>
            <Card sx={{ height: '100%' }}>
              <CardContent>
                <Typography variant="subtitle1" sx={{ fontWeight: 700 }} gutterBottom>
                  {t('library.unwatchedContent', 'Unwatched Content')}
                </Typography>
                {unwatchedLoading ? (
                  Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} variant="text" sx={{ mb: 0.5 }} />)
                ) : !unwatchedContent ? (
                  <Typography variant="body2" color="text.secondary" sx={{ py: 2, textAlign: 'center' }}>{t('common.noData')}</Typography>
                ) : (
                  <>
                    <Box sx={{ display: 'flex', gap: 3, mb: 2 }}>
                      <Box sx={{ textAlign: 'center', flex: 1, p: 1.5, borderRadius: 1, bgcolor: (theme) => alpha(theme.palette.warning.main, 0.08) }}>
                        <Typography variant="h5" sx={{ fontWeight: 700 }}>{unwatchedContent.summary.unwatchedItems}</Typography>
                        <Typography variant="caption" color="text.secondary">{t('library.unwatchedItems', 'Unwatched')}</Typography>
                      </Box>
                      <Box sx={{ textAlign: 'center', flex: 1, p: 1.5, borderRadius: 1, bgcolor: (theme) => alpha(theme.palette.warning.main, 0.08) }}>
                        <Typography variant="h5" sx={{ fontWeight: 700 }}>{unwatchedContent.summary.unwatchedPercent.toFixed(1)}%</Typography>
                        <Typography variant="caption" color="text.secondary">{t('library.unwatchedPercent', 'of library')}</Typography>
                      </Box>
                    </Box>

                    <Box sx={{ mb: 2 }}>
                      <PieChart
                        series={[{
                          data: [
                            { id: 0, value: unwatchedContent.summary.totalItems - unwatchedContent.summary.unwatchedItems, label: t('library.watched', 'Watched'), color: CHART_COLORS[1] },
                            { id: 1, value: unwatchedContent.summary.unwatchedItems, label: t('library.unwatched', 'Unwatched'), color: CHART_COLORS[0] },
                          ].filter((d) => d.value > 0),
                          innerRadius: 35,
                          outerRadius: 80,
                          paddingAngle: 2,
                          cornerRadius: 3,
                        }]}
                        height={200}
                        sx={{ width: '100%' }}
                      />
                    </Box>

                    {unwatchedContent.items.results.length > 0 && (
                      <>
                        <Typography variant="caption" sx={{ fontWeight: 700, display: 'block', mb: 0.5 }} color="text.secondary">
                          {t('library.unwatchedItemsList', 'Unwatched items')}
                        </Typography>
                        <List dense disablePadding>
                          {unwatchedContent.items.results.map((item) => (
                            <ListItem key={item.id} disablePadding sx={{ py: 0.25 }}>
                              <ListItemText
                                primary={item.name}
                                secondary={item.type}
                                slotProps={{
                                  primary: { style: { fontSize: 12, fontWeight: 500 } },
                                  secondary: { style: { fontSize: 11 } },
                                }}
                              />
                            </ListItem>
                          ))}
                        </List>
                      </>
                    )}
                  </>
                )}
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      )}
    </>
  )
}
