import { useCallback, useEffect, useState } from 'react'
import {
  Box, Card, CardContent, CardHeader, Chip, Avatar, Typography,
  LinearProgress, Tooltip, Skeleton,
} from '@mui/material'
import {
  PersonCircle24Regular, Play24Regular, Pause24Regular,
  Desktop24Regular, Phone24Regular, Tv24Regular,
} from '@fluentui/react-icons'
import { useTranslation } from 'react-i18next'
import { useSocket } from '@/shared/hooks/useSocket'
import type { Session } from '@/shared/types/activity'

interface LiveSessionsProps {
  initialSessions: Session[]
  loading: boolean
}

function formatDuration(ticks?: number): string {
  if (!ticks) return '0:00'
  const totalSeconds = Math.floor(ticks / 10_000_000)
  const h = Math.floor(totalSeconds / 3600)
  const m = Math.floor((totalSeconds % 3600) / 60)
  const s = totalSeconds % 60
  if (h > 0) return `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
  return `${m}:${String(s).padStart(2, '0')}`
}

function formatBitrate(bps?: number): string {
  if (!bps) return ''
  if (bps >= 1_000_000) return `${(bps / 1_000_000).toFixed(1)} Mbps`
  return `${Math.round(bps / 1000)} kbps`
}

function getPlayMethodChip(method?: string) {
  if (!method) return null
  const m = method.toLowerCase()
  if (m === 'directplay') return { label: 'Direct Play', color: 'success' as const }
  if (m === 'directstream') return { label: 'Direct Stream', color: 'info' as const }
  if (m === 'transcode') return { label: 'Transcode', color: 'primary' as const }
  return { label: method, color: 'default' as const }
}

function getDeviceIcon(client?: string) {
  const c = (client ?? '').toLowerCase()
  if (c.includes('mobile') || c.includes('android') || c.includes('ios')) return <Phone24Regular style={{ fontSize: 14 }} />
  if (c.includes('tv') || c.includes('roku') || c.includes('fire') || c.includes('androidtv')) return <Tv24Regular style={{ fontSize: 14 }} />
  return <Desktop24Regular style={{ fontSize: 14 }} />
}

function episodeLabel(item: NonNullable<Session['NowPlayingItem']>): string {
  const season = item.ParentIndexNumber
  const ep = item.IndexNumber
  if (season != null && ep != null) {
    return `S${String(season).padStart(2, '0')}E${String(ep).padStart(2, '0')}`
  }
  return ''
}

export default function LiveSessions({ initialSessions, loading }: LiveSessionsProps) {
  const { t } = useTranslation()
  const [sessions, setSessions] = useState<Session[]>(initialSessions)

  useEffect(() => { setSessions(initialSessions) }, [initialSessions])

  const handleSessionUpdate = useCallback((data: unknown) => {
    if (Array.isArray(data)) setSessions(data as Session[])
  }, [])

  useSocket('sessions', handleSessionUpdate)

  const activeSessions = sessions
    .filter((s) => s.NowPlayingItem)
    .sort((a, b) => a.Id.localeCompare(b.Id))

  return (
    <Card>
      <CardHeader
        title={
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            {t('dashboard.liveSessions')}
            <Chip
              label={activeSessions.length}
              size="small"
              color={activeSessions.length > 0 ? 'primary' : 'default'}
              sx={{ height: 20, fontSize: 11 }}
            />
          </Box>
        }
        slotProps={{ title: { variant: 'subtitle1', sx: { fontWeight: 600 } } }}
      />
      <CardContent sx={{ pt: 0 }}>
        {loading ? (
          Array.from({ length: 2 }).map((_, i) => (
            <Skeleton key={i} variant="rectangular" height={88} sx={{ mb: 1.5, borderRadius: 1 }} />
          ))
        ) : activeSessions.length === 0 ? (
          <Typography variant="body2" color="text.secondary" sx={{ py: 2, textAlign: 'center' }}>
            {t('dashboard.noActiveSessions')}
          </Typography>
        ) : (
          activeSessions.map((session) => {
            const item = session.NowPlayingItem!
            const positionTicks = session.PlayState?.PositionTicks
            const totalTicks = item.RunTimeTicks
            const progress = positionTicks && totalTicks ? Math.min((positionTicks / totalTicks) * 100, 100) : 0
            const isPaused = session.PlayState?.IsPaused ?? false
            const playMethod = session.PlayState?.PlayMethod ?? item.MediaType
            const methodChip = getPlayMethodChip(playMethod)
            const epLabel = episodeLabel(item)
            const posterItemId = item.SeriesId || item.Id
            const ti = session.TranscodingInfo

            const transcodingTooltip = ti
              ? [
                  ti.VideoCodec ? `Video: ${ti.VideoCodec}${ti.Width && ti.Height ? ` (${ti.Width}×${ti.Height})` : ''}` : null,
                  ti.AudioCodec ? `Audio: ${ti.AudioCodec}` : null,
                  ti.Bitrate ? `Bitrate: ${formatBitrate(ti.Bitrate)}` : null,
                  ti.IsVideoDirect ? 'Video: Direct' : null,
                  ti.IsAudioDirect ? 'Audio: Direct' : null,
                  ti.VideoDecoderIsHardware ? 'HW Decode' : null,
                  ti.VideoEncoderIsHardware ? 'HW Encode' : null,
                ].filter(Boolean).join(' · ')
              : ''

            return (
              <Box
                key={session.Id}
                sx={{
                  display: 'flex',
                  gap: 1.5,
                  py: 1.5,
                  borderBottom: '1px solid',
                  borderColor: 'divider',
                  '&:last-child': { borderBottom: 0 },
                }}
              >
                {/* Media poster */}
                <Box
                  sx={{
                    flexShrink: 0,
                    width: 52,
                    height: 78,
                    borderRadius: 1,
                    overflow: 'hidden',
                    bgcolor: 'action.hover',
                  }}
                >
                  <img
                    src={`/proxy/Items/Images/Primary/?id=${posterItemId}&fillWidth=104&quality=85`}
                    alt={item.Name}
                    style={{ width: '100%', height: '100%', objectFit: 'cover', display: 'block' }}
                    onError={(e) => { (e.currentTarget as HTMLImageElement).style.display = 'none' }}
                  />
                </Box>

                {/* Main content */}
                <Box sx={{ flex: 1, minWidth: 0 }}>
                  {/* User row */}
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, mb: 0.5 }}>
                    <Avatar
                      src={`/proxy/Users/Images/Primary/?id=${session.UserId}&fillWidth=48&quality=85`}
                      sx={{ width: 20, height: 20, bgcolor: 'primary.main', fontSize: 10 }}
                    >
                      <PersonCircle24Regular style={{ fontSize: 12 }} />
                    </Avatar>
                    <Typography variant="caption" sx={{ fontWeight: 600 }}>
                      {session.UserName}
                    </Typography>
                    {isPaused
                      ? <Pause24Regular style={{ fontSize: 12, color: 'var(--mui-palette-text-secondary)' }} />
                      : <Play24Regular style={{ fontSize: 12, color: 'var(--mui-palette-primary-main)' }} />
                    }
                  </Box>

                  {/* Media title */}
                  {item.SeriesName ? (
                    <>
                      <Typography variant="body2" sx={{ fontWeight: 600, lineHeight: 1.3 }} noWrap>
                        {item.SeriesName}
                      </Typography>
                      <Typography variant="caption" color="text.secondary" noWrap sx={{ display: 'block', lineHeight: 1.3 }}>
                        {epLabel ? `${epLabel} · ` : ''}{item.Name}
                      </Typography>
                    </>
                  ) : (
                    <Typography variant="body2" sx={{ fontWeight: 600, lineHeight: 1.3 }} noWrap>
                      {item.Name}{item.ProductionYear ? ` (${item.ProductionYear})` : ''}
                    </Typography>
                  )}

                  {/* Progress bar */}
                  <Box sx={{ mt: 0.75 }}>
                    <LinearProgress
                      variant="determinate"
                      value={progress}
                      sx={{ height: 3, borderRadius: 1.5 }}
                    />
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', mt: 0.25 }}>
                      <Typography variant="caption" color="text.secondary" sx={{ fontSize: 10 }}>
                        {formatDuration(positionTicks)}
                      </Typography>
                      {totalTicks && (
                        <Typography variant="caption" color="text.secondary" sx={{ fontSize: 10 }}>
                          {formatDuration(totalTicks)}
                        </Typography>
                      )}
                    </Box>
                  </Box>

                  {/* Chips row */}
                  <Box sx={{ display: 'flex', gap: 0.5, mt: 0.5, flexWrap: 'wrap', alignItems: 'center' }}>
                    {methodChip && (
                      <Tooltip title={transcodingTooltip || methodChip.label} placement="top">
                        <Chip
                          label={methodChip.label}
                          size="small"
                          color={methodChip.color}
                          sx={{ height: 18, fontSize: 10, '& .MuiChip-label': { px: 0.75 } }}
                        />
                      </Tooltip>
                    )}
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.25, ml: 'auto' }}>
                      {getDeviceIcon(session.Client)}
                      <Typography variant="caption" color="text.secondary" sx={{ fontSize: 10 }} noWrap>
                        {session.Client}{session.DeviceName && session.DeviceName !== session.Client ? ` · ${session.DeviceName}` : ''}
                      </Typography>
                    </Box>
                  </Box>
                </Box>
              </Box>
            )
          })
        )}
      </CardContent>
    </Card>
  )
}
