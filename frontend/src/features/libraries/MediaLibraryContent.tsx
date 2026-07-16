import { useState, useEffect, useMemo, useCallback } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import {
  Grid, Card, CardActionArea, CardContent, Typography, Tabs, Tab, Box,
  Chip, Skeleton, TextField, InputAdornment,
  IconButton, Tooltip, CircularProgress,
  TablePagination, ToggleButtonGroup, ToggleButton,
  Select, MenuItem, FormControl, InputLabel,
} from '@mui/material'
import { createColumnHelper } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'
import { PieChart } from '@mui/x-charts/PieChart'
import { BarChart } from '@mui/x-charts/BarChart'
import StatCard from '@/shared/components/StatCard/StatCard'
import ChartCard from '@/shared/components/ChartCard/ChartCard'
import DataTable from '@/shared/components/DataTable/DataTable'
import MediaPoster from '@/shared/components/MediaPoster/MediaPoster'
import api from '@/lib/axios'
import type { LibraryItem, LibraryStats, GenreStat } from '@/shared/types/library'
import type { Activity } from '@/shared/types/activity'
import {
  Play24Regular, Clock24Regular, Star24Regular,
  Search20Regular, VideoClip24Regular,
  Grid24Regular, TableSimple24Regular, ArrowSync24Regular,
} from '@fluentui/react-icons'
import { useChartColors } from '@/lib/chartColors'
import { formatWatchTime } from '@/shared/utils/formatWatchTime'
import { formatSize } from '@/shared/utils/formatSize'
import { formatDateOnly } from '@/shared/utils/formatDate'
import { getItemImageUrl } from '@/shared/utils/imageUrl'

import LibraryActivityTab from './components/LibraryActivityTab'
import LibrarySharedStats from './components/LibrarySharedStats'
import type { HistoryPoint, ItemWithStats, PlayMethodStat, LastPlayedRow, TimeToWatchData, UnwatchedContentData } from './components/types'

const VIEWS = ['items', 'stats', 'activity'] as const

// ─── Genre Distribution ─────────────────────────────────────────────────────

const PIE_LIMITS = [8, 12, 20, 9999] as const

function GenreDistributionCard({ genres, loading }: { genres: GenreStat[]; loading: boolean }) {
  const { t } = useTranslation()
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
            <Grid size={{ xs: 12, md: 6 }}>
              <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 2 }}>
                <PieChart
                  width={260}
                  height={260}
                  series={[{
                    data: pieGenres.map((g, i) => ({
                      id: i, value: g.Count, label: g.Genre, color: COLORS[i % COLORS.length],
                    })),
                    outerRadius: 110, paddingAngle: 2, cornerRadius: 3,
                    highlightScope: { fade: 'global', highlight: 'item' },
                  }]}
                  slotProps={{ legend: { hidden: true } as any }}
                />
                <FormControl size="small" sx={{ minWidth: 180 }}>
                  <InputLabel>{t('library.showGenres', 'Genres affichés')}</InputLabel>
                  <Select value={pieLimit} label={t('library.showGenres', 'Genres affichés')} onChange={(e) => setPieLimit(Number(e.target.value))}>
                    {PIE_LIMITS.map((n) => (
                      <MenuItem key={n} value={n}>
                        {n >= 9999 ? t('library.allGenres', 'Tous les genres') : `Top ${n}`}
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
              </Box>
            </Grid>
            <Grid size={{ xs: 12, md: 6 }}>
              <Box sx={{ height: 320, overflowY: 'auto', scrollbarWidth: 'thin', pr: 0.5 }}>
                {genres.map((g, i) => {
                  const pct = total > 0 ? (g.Count / total) * 100 : 0
                  return (
                    <Box key={g.Genre} sx={{ display: 'flex', alignItems: 'center', gap: 1.25, py: 0.65, borderBottom: '1px solid', borderColor: 'divider', '&:last-child': { borderBottom: 0 } }}>
                      <Box sx={{ width: 10, height: 10, borderRadius: '50%', bgcolor: COLORS[i % COLORS.length], flexShrink: 0 }} />
                      <Typography variant="body2" sx={{ flex: 1, fontWeight: 500 }} noWrap>{g.Genre}</Typography>
                      <Box sx={{ width: 60, flexShrink: 0, display: { xs: 'none', sm: 'block' } }}>
                        <Box sx={{ height: 4, borderRadius: 2, bgcolor: 'action.hover', overflow: 'hidden' }}>
                          <Box sx={{ height: '100%', borderRadius: 2, width: `${pct}%`, bgcolor: COLORS[i % COLORS.length] }} />
                        </Box>
                      </Box>
                      <Typography variant="caption" color="text.secondary" sx={{ flexShrink: 0, minWidth: 28, textAlign: 'right', fontVariantNumeric: 'tabular-nums' }}>
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

function ItemsGridView({ items, loading, navigate, libraryId, cols = 6 }: {
  items: LibraryItem[]; loading: boolean; navigate: ReturnType<typeof useNavigate>; libraryId: string; cols?: 6 | 8 | 10
}) {
  const { t } = useTranslation()
  const [page, setPage] = useState(0)
  const [pageSize, setPageSize] = useState<typeof GRID_PAGE_SIZES[number]>(30)
  const paged = items.slice(page * pageSize, (page + 1) * pageSize)

  useEffect(() => { setPage(0) }, [items.length, pageSize])

  return (
    <Box>
      <Box sx={{ display: 'grid', gridTemplateColumns: `repeat(${cols}, 1fr)`, gap: 2, alignItems: 'start' }}>
        {loading ? (
          Array.from({ length: pageSize }).map((_, i) => (
            <Skeleton key={i} variant="rectangular" height={260} sx={{ borderRadius: 2 }} />
          ))
        ) : paged.length === 0 ? (
          <Box sx={{ gridColumn: '1 / -1', py: 4, textAlign: 'center' }}>
            <Typography variant="body2" color="text.secondary">{t('common.noData')}</Typography>
          </Box>
        ) : (
          paged.map((item) => {
            const size = formatSize(item.Size)
            return (
              <Card key={item.Id} sx={{ overflow: 'hidden', borderRadius: 2, border: '1px solid', borderColor: 'divider', transition: 'border-color 160ms ease', '&:hover': { borderColor: 'primary.main' } }}>
                <CardActionArea onClick={() => navigate(`/libraries/${libraryId}/items/${item.Id}`)}>
                  <Box sx={{ position: 'relative', aspectRatio: '2 / 3', bgcolor: 'rgba(255,255,255,0.04)', display: 'flex', alignItems: 'center', justifyContent: 'center', overflow: 'hidden' }}>
                    <VideoClip24Regular style={{ fontSize: 44, opacity: 0.45 }} />
                    <Box
                      component="img" src={getItemImageUrl(item.Id, 360)} alt={item.Name} loading="lazy"
                      onError={(e) => { e.currentTarget.style.display = 'none' }}
                      sx={{ position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover', transition: 'transform 300ms ease', '.MuiCard-root:hover &': { transform: 'scale(1.06)' } }}
                    />
                    {size && (
                      <Chip label={size} size="small" sx={{ position: 'absolute', right: 6, bottom: 6, height: 20, fontSize: 10, bgcolor: 'primary.main', color: 'primary.contrastText' }} />
                    )}
                  </Box>
                  <CardContent sx={{ p: 1.25, '&:last-child': { pb: 1.25 } }}>
                    <Typography variant="body2" sx={{ fontWeight: 600, lineHeight: 1.25 }} title={item.Name}>{item.Name}</Typography>
                    <Box sx={{ display: 'flex', gap: 0.75, mt: 0.75, flexWrap: 'wrap' }}>
                      {item.ProductionYear && <Typography variant="caption" color="text.secondary">{item.ProductionYear}</Typography>}
                      {item.CommunityRating && <Typography variant="caption" color="warning.main">★ {item.CommunityRating.toFixed(1)}</Typography>}
                      {item.PlayCount > 0 && <Typography variant="caption" color="text.secondary">{item.PlayCount} {t('common.plays')}</Typography>}
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
          component="div" count={items.length} page={page} rowsPerPage={pageSize}
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

function ItemsTableView({ items, loading, navigate, libraryId }: {
  items: LibraryItem[]; loading: boolean; navigate: ReturnType<typeof useNavigate>; libraryId: string
}) {
  const { t } = useTranslation()
  const [hovered, setHovered] = useState<LibraryItem | null>(null)

  const columns = useMemo(() => [
    itemColHelper.display({
      id: 'poster', size: 52, header: () => null, meta: { hideFromColumnsMenu: true },
      cell: (info) => <MediaPoster src={getItemImageUrl(info.row.original.Id, 80, 80)} alt={info.row.original.Name} type={info.row.original.Type} />,
    }),
    itemColHelper.accessor('Name', {
      header: t('library.title', 'Titre'),
      cell: (info) => {
        const item = info.row.original
        return (
          <Box>
            <Typography variant="body2" sx={{ fontWeight: 600, lineHeight: 1.3 }}>{info.getValue()}</Typography>
            {item.SeriesName && (
              <Typography variant="caption" color="text.secondary" sx={{ display: 'block' }} noWrap>
                {item.SeriesName}{item.ParentIndexNumber != null && item.IndexNumber != null ? ` — S${item.ParentIndexNumber}E${item.IndexNumber}` : ''}
              </Typography>
            )}
          </Box>
        )
      },
    }),
    itemColHelper.accessor('Type', {
      header: t('library.type', 'Type'), size: 90,
      cell: (info) => <Chip label={info.getValue()} size="small" variant="outlined" sx={{ fontSize: 11, height: 20 }} />,
    }),
    itemColHelper.accessor('ProductionYear', {
      header: t('library.year', 'Année'), size: 70,
      cell: (info) => <Typography variant="caption" color="text.secondary">{info.getValue() ?? '—'}</Typography>,
    }),
    itemColHelper.accessor('CommunityRating', {
      header: '★', size: 60,
      cell: (info) => {
        const v = info.getValue()
        return v
          ? <Typography variant="caption" color="warning.main" sx={{ fontWeight: 600 }}>★ {v.toFixed(1)}</Typography>
          : <Typography variant="caption" color="text.disabled">—</Typography>
      },
    }),
    itemColHelper.accessor('PlayCount', {
      header: t('common.plays'), size: 80,
      cell: (info) => {
        const v = info.getValue()
        return v > 0
          ? <Chip label={v} size="small" sx={{ fontSize: 11, height: 20 }} />
          : <Typography variant="caption" color="text.disabled">0</Typography>
      },
    }),
    itemColHelper.accessor('Size', {
      header: t('library.size', 'Taille'), size: 90,
      cell: (info) => <Typography variant="caption" color="text.secondary">{formatSize(info.getValue()) ?? '—'}</Typography>,
    }),
  ], [t])

  return (
    <Box sx={{ display: 'flex', gap: 2, alignItems: 'flex-start' }}>
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <DataTable data={items} columns={columns} loading={loading} searchable={false} onRowClick={(row) => navigate(`/libraries/${libraryId}/items/${row.Id}`)} onRowHover={setHovered} />
      </Box>
      <Box sx={{ width: 200, flexShrink: 0, position: 'sticky', top: 80, display: { xs: 'none', lg: 'block' } }} onMouseLeave={() => setHovered(null)}>
        <Card variant="outlined" sx={{ borderRadius: 2, overflow: 'hidden', transition: 'opacity 200ms', opacity: hovered ? 1 : 0.35 }}>
          <Box sx={{ position: 'relative', aspectRatio: '2 / 3', bgcolor: 'rgba(255,255,255,0.05)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <VideoClip24Regular style={{ fontSize: 48, opacity: 0.25 }} />
            {hovered && (
              <Box component="img" key={hovered.Id} src={getItemImageUrl(hovered.Id, 400)} alt={hovered.Name}
                onError={(e) => { e.currentTarget.style.display = 'none' }}
                sx={{ position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover', transition: 'opacity 200ms' }}
              />
            )}
          </Box>
          <CardContent sx={{ p: 1.5, '&:last-child': { pb: 1.5 } }}>
            {hovered ? (
              <>
                <Typography variant="body2" sx={{ fontWeight: 700, lineHeight: 1.3, mb: 0.75 }}>{hovered.Name}</Typography>
                <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap', mb: 0.75 }}>
                  <Chip label={hovered.Type} size="small" variant="outlined" sx={{ fontSize: 10, height: 18 }} />
                  {hovered.ProductionYear && <Chip label={hovered.ProductionYear} size="small" variant="outlined" sx={{ fontSize: 10, height: 18 }} />}
                </Box>
                {hovered.CommunityRating != null && (
                  <Typography variant="caption" color="warning.main" sx={{ display: 'block', fontWeight: 600, mb: 0.5 }}>★ {hovered.CommunityRating.toFixed(1)}</Typography>
                )}
                <Typography variant="caption" color="text.secondary" sx={{ display: 'block' }}>
                  {hovered.PlayCount > 0 ? `${hovered.PlayCount} ${t('common.plays')}` : t('library.neverPlayed', 'Jamais lu')}
                </Typography>
                {hovered.Size != null && formatSize(hovered.Size) && (
                  <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.25 }}>{formatSize(hovered.Size)}</Typography>
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

// ─── Main content ──────────────────────────────────────────────────────────

interface MediaLibraryContentProps {
  libraryId: string
  libraryName: string
}

export default function MediaLibraryContent({ libraryId }: MediaLibraryContentProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const CHART_COLORS = useChartColors()
  const CHART_BAR = CHART_COLORS[0]
  const [searchParams, setSearchParams] = useSearchParams()

  const [items, setItems] = useState<LibraryItem[]>([])
  const [stats, setStats] = useState<LibraryStats | null>(null)
  const [genres, setGenres] = useState<GenreStat[]>([])
  const [loading, setLoading] = useState(true)
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

  const load = useCallback((showLoading = true) => {
    if (showLoading) setLoading(true)
    Promise.allSettled([
      api.get(`/stats/getLibraryStats?libraryId=${libraryId}`),
      api.get(`/stats/getLibraryItems?libraryId=${libraryId}`),
      api.get(`/stats/getGenreStats?libraryId=${libraryId}`),
      api.post('/api/getLibraryHistory', { libraryid: libraryId }),
      api.post('/stats/getLibraryItemsWithStats', { libraryid: libraryId }),
      api.post('/stats/getLibraryItemsPlayMethodStats', { libraryid: libraryId }),
      api.post('/stats/getLibraryLastPlayed', { libraryid: libraryId }),
    ]).then(([statsRes, itemsRes, genresRes, historyRes, itemsStatsRes, methodRes, lastPlayedRes]) => {
      if (statsRes.status === 'fulfilled') setStats(statsRes.value.data)
      if (itemsRes.status === 'fulfilled') setItems(itemsRes.value.data ?? [])
      if (genresRes.status === 'fulfilled') setGenres(genresRes.value.data ?? [])

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

      if (itemsStatsRes.status === 'fulfilled') {
        const results: ItemWithStats[] = itemsStatsRes.value.data?.results ?? []
        setItemsWithStats([...results].sort((a, b) => b.times_played - a.times_played).slice(0, 10))
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

  const filteredItems = useMemo(() => {
    const term = search.trim().toLowerCase()
    if (!term) return items
    return items.filter((item) =>
      item.Name.toLowerCase().includes(term) ||
      item.Type.toLowerCase().includes(term) ||
      String(item.ProductionYear ?? '').includes(term)
    )
  }, [items, search])

  const tab = Math.max(0, VIEWS.indexOf((searchParams.get('view') ?? '') as typeof VIEWS[number]))
  const setTab = (idx: number) => setSearchParams({ view: VIEWS[idx] ?? VIEWS[0] }, { replace: true })

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
          <Tab label={t('library.items')} />
          <Tab label={t('library.stats', 'Stats')} />
          <Tab label={t('library.activityHistory', "Historique d'activité")} />
        </Tabs>
      </Box>

      {/* Items tab */}
      {tab === 0 && (
        <Box>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2, flexWrap: 'wrap' }}>
            <TextField
              size="small" placeholder={t('common.search')} value={search}
              onChange={(e) => setSearch(e.target.value)}
              slotProps={{ input: { startAdornment: <InputAdornment position="start"><Search20Regular style={{ fontSize: 16 }} /></InputAdornment> } }}
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
                <ToggleButtonGroup value={gridCols} exclusive onChange={(_, v) => v && setGridCols(v)} size="small">
                  {([6, 8, 10] as const).map((n) => (
                    <ToggleButton key={n} value={n} sx={{ px: 1.25, fontSize: 12, fontWeight: 600, minWidth: 32 }}>{n}</ToggleButton>
                  ))}
                </ToggleButtonGroup>
              )}
              <ToggleButtonGroup value={viewMode} exclusive onChange={(_, v) => v && setViewMode(v)} size="small">
                <ToggleButton value="grid" aria-label="grid view"><Grid24Regular style={{ fontSize: 18 }} /></ToggleButton>
                <ToggleButton value="table" aria-label="table view"><TableSimple24Regular style={{ fontSize: 18 }} /></ToggleButton>
              </ToggleButtonGroup>
            </Box>
          </Box>
          {viewMode === 'grid'
            ? <ItemsGridView items={filteredItems} loading={loading} navigate={navigate} libraryId={libraryId} cols={gridCols} />
            : <ItemsTableView items={filteredItems} loading={loading} navigate={navigate} libraryId={libraryId} />
          }
        </Box>
      )}

      {/* Stats tab */}
      {tab === 1 && (
        <Grid container spacing={2}>
          <Grid size={{ xs: 12 }}>
            <GenreDistributionCard genres={genres} loading={loading} />
          </Grid>
          <Grid size={{ xs: 12 }}>
            <ChartCard title={t('library.topItems', 'Items les plus regardés')} loading={loading} empty={itemsWithStats.length === 0} height={280}>
              <BarChart
                layout="horizontal"
                yAxis={[{ data: itemsWithStats.map((item) => item.Name), scaleType: 'band' }]}
                xAxis={[{ label: t('common.plays') }]}
                series={[{ data: itemsWithStats.map((item) => item.times_played), label: t('common.plays'), color: CHART_BAR }]}
                height={280} sx={{ width: '100%' }} grid={{ vertical: true }}
                slotProps={{ legend: { hidden: true } as any }}
              />
            </ChartCard>
          </Grid>
          <LibrarySharedStats
            historyData={historyData} playMethodStats={playMethodStats} lastPlayed={lastPlayed}
            timeToWatch={timeToWatch} timeToWatchLoading={timeToWatchLoading}
            unwatchedContent={unwatchedContent} unwatchedLoading={unwatchedLoading}
            loading={loading}
          />
        </Grid>
      )}

      {/* Activity tab */}
      {tab === 2 && (
        <LibraryActivityTab data={activityHistory} loading={loading} onRefresh={() => load(false)} />
      )}
    </>
  )
}
