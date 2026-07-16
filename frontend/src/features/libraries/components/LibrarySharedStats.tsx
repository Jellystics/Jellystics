import {
  Grid, Card, CardContent, Typography, List, ListItem, ListItemText,
  Box,
} from '@mui/material'
import { alpha } from '@mui/material/styles'
import { PieChart } from '@mui/x-charts/PieChart'
import { LineChart } from '@mui/x-charts/LineChart'
import { BarChart } from '@mui/x-charts/BarChart'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import SkeletonList from '@/shared/components/SkeletonList/SkeletonList'
import ChartCard from '@/shared/components/ChartCard/ChartCard'
import MediaPoster from '@/shared/components/MediaPoster/MediaPoster'
import { useChartColors } from '@/lib/chartColors'
import { formatDateTime } from '@/shared/utils/formatDate'
import { getItemImageUrl } from '@/shared/utils/imageUrl'
import type { HistoryPoint, PlayMethodStat, LastPlayedRow, TimeToWatchData, UnwatchedContentData } from './types'

interface SharedStatsProps {
  historyData: HistoryPoint[]
  playMethodStats: PlayMethodStat[]
  lastPlayed: LastPlayedRow[]
  timeToWatch: TimeToWatchData | null
  timeToWatchLoading: boolean
  unwatchedContent: UnwatchedContentData | null
  unwatchedLoading: boolean
  loading: boolean
}

export default function LibrarySharedStats({
  historyData, playMethodStats, lastPlayed,
  timeToWatch, timeToWatchLoading,
  unwatchedContent, unwatchedLoading,
  loading,
}: SharedStatsProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const CHART_COLORS = useChartColors()
  const CHART_BAR = CHART_COLORS[0]

  return (
    <>
      {/* Activity Over Time */}
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

      {/* Play Method Pie */}
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

      {/* Last Played */}
      <Grid size={{ xs: 12, md: 6 }}>
        <Card sx={{ height: '100%' }}>
          <CardContent>
            <Typography variant="subtitle1" sx={{ fontWeight: 600 }} gutterBottom>
              {t('library.lastPlayed', 'Derniers lus')}
            </Typography>
            {loading
              ? <SkeletonList count={6} variant="text" spacing={0.5} />
              : lastPlayed.length === 0
                ? <Typography variant="body2" color="text.secondary" sx={{ py: 2, textAlign: 'center' }}>{t('common.noData')}</Typography>
                : (
                  <List dense disablePadding>
                    {lastPlayed.map((row, i) => {
                      const dateStr = formatDateTime(row.ActivityDateInserted)
                      return (
                        <ListItem
                          key={i}
                          disablePadding
                          sx={{ py: 0.5, gap: 1.5, cursor: row.NowPlayingItemId ? 'pointer' : 'default', borderRadius: 1, '&:hover': row.NowPlayingItemId ? { bgcolor: 'action.hover' } : {} }}
                          onClick={() => row.NowPlayingItemId && navigate(`/items/${row.NowPlayingItemId}`)}
                        >
                          {row.NowPlayingItemId && (
                            <MediaPoster src={getItemImageUrl(row.NowPlayingItemId, 56, 80)} width={28} height={40} />
                          )}
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
              <SkeletonList count={4} variant="text" spacing={0.5} />
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
                        <ListItem key={item.id} disablePadding sx={{ py: 0.25, gap: 1, cursor: 'pointer', borderRadius: 1, '&:hover': { bgcolor: 'action.hover' } }} onClick={() => navigate(`/items/${item.id}`)}>
                          <MediaPoster src={getItemImageUrl(item.id, 48, 80)} type={item.type} width={24} height={34} />
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
                        <ListItem key={item.id} disablePadding sx={{ py: 0.25, gap: 1, cursor: 'pointer', borderRadius: 1, '&:hover': { bgcolor: 'action.hover' } }} onClick={() => navigate(`/items/${item.id}`)}>
                          <MediaPoster src={getItemImageUrl(item.id, 48, 80)} type={item.type} width={24} height={34} />
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

      {/* Unwatched Content */}
      <Grid size={{ xs: 12, md: 6 }}>
        <Card sx={{ height: '100%' }}>
          <CardContent>
            <Typography variant="subtitle1" sx={{ fontWeight: 700 }} gutterBottom>
              {t('library.unwatchedContent', 'Unwatched Content')}
            </Typography>
            {unwatchedLoading ? (
              <SkeletonList count={4} variant="text" spacing={0.5} />
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
                        <ListItem key={item.id} disablePadding sx={{ py: 0.25, gap: 1, cursor: 'pointer', borderRadius: 1, '&:hover': { bgcolor: 'action.hover' } }} onClick={() => navigate(`/items/${item.id}`)}>
                          <MediaPoster src={getItemImageUrl(item.id, 48, 80)} type={item.type} width={24} height={34} />
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
    </>
  )
}
