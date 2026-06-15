import { useState, useEffect, useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  Grid, Alert, Card, CardActionArea, CardContent, Typography, Tabs, Tab, Box,
  Chip, List, ListItem, ListItemText, Skeleton, TextField, InputAdornment,
} from '@mui/material'
import { useTranslation } from 'react-i18next'
import { PieChart, Pie, Cell, Tooltip, ResponsiveContainer, Legend } from 'recharts'
import PageHeader from '@/shared/components/PageHeader/PageHeader'
import StatCard from '@/shared/components/StatCard/StatCard'
import ChartCard from '@/shared/components/ChartCard/ChartCard'
import api from '@/lib/axios'
import type { LibraryItem, LibraryStats, GenreStat } from '@/shared/types/library'
import {
  Play24Regular, Clock24Regular, Star24Regular,
  Search20Regular, VideoClip24Regular,
} from '@fluentui/react-icons'
import { formatWatchTime } from '@/shared/utils/formatWatchTime'

const COLORS = ['#a78bfa', '#7c3aed', '#6d28d9', '#5b21b6', '#4c1d95', '#8b5cf6', '#c4b5fd']

function formatSize(bytes?: number): string | null {
  if (!bytes) return null
  const gb = bytes / 1024 / 1024 / 1024
  if (gb >= 1) return `${gb.toFixed(gb >= 10 ? 1 : 2)} GB`
  const mb = bytes / 1024 / 1024
  return `${Math.round(mb)} MB`
}

function posterUrl(item: LibraryItem): string {
  return `/proxy/Items/Images/Primary/?id=${encodeURIComponent(item.Id)}&fillWidth=360&quality=90`
}

export default function LibraryDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const [tab, setTab] = useState(0)
  const [items, setItems] = useState<LibraryItem[]>([])
  const [stats, setStats] = useState<LibraryStats | null>(null)
  const [genres, setGenres] = useState<GenreStat[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [search, setSearch] = useState('')

  useEffect(() => {
    if (!id) return
    const load = (showLoading = true) => {
      if (showLoading) setLoading(true)
      Promise.all([
        api.get(`/stats/getLibraryStats?libraryId=${id}`),
        api.get(`/stats/getLibraryItems?libraryId=${id}`),
        api.get(`/stats/getGenreStats?libraryId=${id}`),
      ])
        .then(([statsRes, itemsRes, genresRes]) => {
          setStats(statsRes.data)
          setItems(itemsRes.data ?? [])
          setGenres(genresRes.data ?? [])
        })
        .catch(() => setError(t('common.loadError')))
        .finally(() => setLoading(false))
    }

    load()
    const interval = window.setInterval(() => load(false), 15000)
    return () => window.clearInterval(interval)
  }, [id, t])

  const filteredItems = useMemo(() => {
    const term = search.trim().toLowerCase()
    if (!term) return items
    return items.filter((item) =>
      item.Name.toLowerCase().includes(term) ||
      item.Type.toLowerCase().includes(term) ||
      String(item.ProductionYear ?? '').includes(term)
    )
  }, [items, search])

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
        <Tabs value={tab} onChange={(_, v) => setTab(v as number)}>
          <Tab label={t('library.items')} />
          <Tab label={t('library.genres')} />
        </Tabs>
      </Box>

      {tab === 0 && (
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
                        height: '100%',
                        overflow: 'hidden',
                        borderRadius: 2,
                        bgcolor: 'background.paper',
                        border: '1px solid',
                        borderColor: 'divider',
                        transition: 'transform 160ms ease, border-color 160ms ease',
                        '&:hover': {
                          transform: 'translateY(-3px)',
                          borderColor: 'primary.main',
                        },
                      }}
                    >
                      <CardActionArea onClick={() => navigate(`/libraries/${id}/items/${item.Id}`)} sx={{ height: '100%' }}>
                        <Box
                          sx={{
                            position: 'relative',
                            aspectRatio: '2 / 3',
                            bgcolor: 'rgba(255,255,255,0.04)',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            color: 'text.secondary',
                          }}
                        >
                          <VideoClip24Regular style={{ fontSize: 44, opacity: 0.45 }} />
                          <Box
                            component="img"
                            src={posterUrl(item)}
                            alt={item.Name}
                            loading="lazy"
                            onError={(event) => { event.currentTarget.style.display = 'none' }}
                            sx={{
                              position: 'absolute',
                              inset: 0,
                              width: '100%',
                              height: '100%',
                              objectFit: 'cover',
                            }}
                          />
                          {size && (
                            <Chip
                              label={size}
                              size="small"
                              sx={{
                                position: 'absolute',
                                right: 6,
                                bottom: 6,
                                height: 20,
                                fontSize: 10,
                                bgcolor: 'primary.main',
                                color: 'primary.contrastText',
                              }}
                            />
                          )}
                        </Box>
                        <CardContent sx={{ p: 1.25, '&:last-child': { pb: 1.25 } }}>
                          <Typography variant="body2" sx={{ fontWeight: 600, lineHeight: 1.25 }} title={item.Name}>
                            {item.Name}
                          </Typography>
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, mt: 0.75, flexWrap: 'wrap' }}>
                            {item.ProductionYear && (
                              <Typography variant="caption" color="text.secondary">
                                {item.ProductionYear}
                              </Typography>
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

      {tab === 1 && (
        <Grid container spacing={2}>
          <Grid size={{ xs: 12, md: 6 }}>
            <ChartCard title={t('library.genreDistribution')} loading={loading} height={320}>
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie data={genres} dataKey="Count" nameKey="Genre" cx="50%" cy="50%" outerRadius={120} label={true}>
                    {genres.map((_, i) => <Cell key={i} fill={COLORS[i % COLORS.length]} />)}
                  </Pie>
                  <Tooltip />
                  <Legend />
                </PieChart>
              </ResponsiveContainer>
            </ChartCard>
          </Grid>
          <Grid size={{ xs: 12, md: 6 }}>
            <Card>
              <CardContent>
                <Typography variant="subtitle1" sx={{ fontWeight: 600 }} gutterBottom>{t('library.genreList')}</Typography>
                {loading ? (
                  Array.from({ length: 8 }).map((_, i) => <Skeleton key={i} variant="text" sx={{ mb: 0.5 }} />)
                ) : (
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
    </>
  )
}
