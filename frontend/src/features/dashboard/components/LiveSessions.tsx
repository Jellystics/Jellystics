import { useCallback, useState } from 'react'
import {
  Box, Card, CardContent, CardHeader, Chip, Avatar, Typography,
  LinearProgress, Tooltip, Skeleton,
} from '@mui/material'
import { PersonCircle24Regular, Play24Regular, Pause24Regular } from '@fluentui/react-icons'
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

export default function LiveSessions({ initialSessions, loading }: LiveSessionsProps) {
  const { t } = useTranslation()
  const [sessions, setSessions] = useState<Session[]>(initialSessions)

  const handleSessionUpdate = useCallback((data: unknown) => {
    if (Array.isArray(data)) setSessions(data as Session[])
  }, [])

  useSocket('sessions', handleSessionUpdate)

  const activeSessions = sessions.filter((s) => s.NowPlayingItem)

  return (
    <Card>
      <CardHeader
        title={
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            {t('dashboard.liveSessions')}
            <Chip label={activeSessions.length} size="small" color={activeSessions.length > 0 ? 'primary' : 'default'} sx={{ height: 20, fontSize: 11 }} />
          </Box>
        }
        slotProps={{ title: { variant: 'subtitle1', sx: { fontWeight: 600 } } }}
      />
      <CardContent sx={{ pt: 0 }}>
        {loading ? (
          Array.from({ length: 2 }).map((_, i) => (
            <Skeleton key={i} variant="rectangular" height={72} sx={{ mb: 1, borderRadius: 1 }} />
          ))
        ) : activeSessions.length === 0 ? (
          <Typography variant="body2" color="text.secondary" sx={{ py: 2, textAlign: 'center' }}>
            {t('dashboard.noActiveSessions')}
          </Typography>
        ) : (
          activeSessions.map((session) => (
            <Box key={session.Id} sx={{ display: 'flex', alignItems: 'center', gap: 1.5, py: 1, borderBottom: '1px solid', borderColor: 'divider', '&:last-child': { borderBottom: 0 } }}>
              <Avatar
                src={`/proxy/Users/Images/Primary/?id=${session.UserId}&fillWidth=72&quality=90`}
                sx={{ width: 36, height: 36, bgcolor: 'primary.main', fontSize: 14 }}
              >
                <PersonCircle24Regular />
              </Avatar>
              <Box sx={{ flex: 1, minWidth: 0 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mb: 0.25 }}>
                  <Typography variant="body2" sx={{ fontWeight: 600 }} noWrap>{session.UserName}</Typography>
                  {session.PlayState?.IsPaused
                    ? <Pause24Regular style={{ fontSize: 14, color: 'var(--mui-palette-text-secondary)' }} />
                    : <Play24Regular style={{ fontSize: 14, color: 'var(--mui-palette-primary-main)' }} />}
                </Box>
                <Tooltip title={session.NowPlayingItem?.SeriesName ?? session.NowPlayingItem?.Name ?? ''}>
                  <Typography variant="caption" color="text.secondary" noWrap sx={{ display: 'block' }}>
                    {session.NowPlayingItem?.SeriesName
                      ? `${session.NowPlayingItem.SeriesName} · ${session.NowPlayingItem.Name}`
                      : session.NowPlayingItem?.Name}
                  </Typography>
                </Tooltip>
                <LinearProgress variant="determinate" value={0} sx={{ mt: 0.5, height: 2, borderRadius: 1 }} />
              </Box>
              <Box sx={{ textAlign: 'right', flexShrink: 0 }}>
                <Typography variant="caption" color="text.secondary">
                  {formatDuration(session.PlayState?.PositionTicks)}
                </Typography>
                <Typography variant="caption" color="text.secondary" sx={{ display: 'block' }}>{session.Client}</Typography>
              </Box>
            </Box>
          ))
        )}
      </CardContent>
    </Card>
  )
}
