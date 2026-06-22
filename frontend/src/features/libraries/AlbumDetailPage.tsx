import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  Box, Card, CardContent, Typography, List, ListItem, Skeleton, Alert, Chip,
} from '@mui/material'
import { useTranslation } from 'react-i18next'
import { MusicNote224Regular, Play24Regular, ArrowLeft24Regular } from '@fluentui/react-icons'
import PageHeader from '@/shared/components/PageHeader/PageHeader'
import api from '@/lib/axios'
import { formatTicks } from '@/shared/utils/formatTicks'

interface Track {
  Id: string
  Name: string
  IndexNumber: number | null
  RunTimeTicks: number | null
  AlbumId: string
  AlbumName: string
  Artist: string | null
  PlayCount: number
}


export default function AlbumDetailPage() {
  const { albumId } = useParams<{ albumId: string }>()
  const navigate = useNavigate()
  const { t } = useTranslation()

  const [tracks, setTracks] = useState<Track[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!albumId) return
    setLoading(true)
    api.get(`/stats/getAlbumTracks?albumId=${albumId}`)
      .then((res) => setTracks(res.data ?? []))
      .catch(() => setError(t('common.loadError')))
      .finally(() => setLoading(false))
  }, [albumId, t])

  const albumName = tracks[0]?.AlbumName ?? ''
  const artistName = tracks[0]?.Artist ?? null

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
      <PageHeader title={albumName || '…'} />
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      <Box sx={{ display: 'flex', gap: 3, mb: 3, flexDirection: { xs: 'column', sm: 'row' }, alignItems: { sm: 'flex-start' } }}>
        {/* Album cover */}
        <Box
          sx={{
            flexShrink: 0,
            width: { xs: '100%', sm: 200 },
            maxWidth: 200,
            aspectRatio: '1 / 1',
            borderRadius: 2,
            overflow: 'hidden',
            bgcolor: 'rgba(255,255,255,0.05)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            position: 'relative',
          }}
        >
          <MusicNote224Regular style={{ fontSize: 56, opacity: 0.3 }} />
          {albumId && (
            <Box
              component="img"
              src={`/proxy/Items/Images/Primary/?id=${albumId}&fillWidth=400&quality=90`}
              alt={albumName}
              onError={(e) => { e.currentTarget.style.display = 'none' }}
              sx={{ position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover' }}
            />
          )}
        </Box>

        {/* Album info + track list */}
        <Box sx={{ flex: 1, minWidth: 0 }}>
          {artistName && (
            <Typography variant="body2" color="text.secondary" sx={{ mb: 1.5, fontWeight: 500 }}>
              {artistName}
            </Typography>
          )}
          <Card>
            <CardContent sx={{ p: 0, '&:last-child': { pb: 0 } }}>
              {loading ? (
                Array.from({ length: 8 }).map((_, i) => (
                  <Box key={i} sx={{ display: 'flex', gap: 2, px: 2, py: 1.5, alignItems: 'center' }}>
                    <Skeleton variant="text" width={20} />
                    <Skeleton variant="text" sx={{ flex: 1 }} />
                    <Skeleton variant="text" width={40} />
                  </Box>
                ))
              ) : tracks.length === 0 ? (
                <Typography variant="body2" color="text.secondary" sx={{ p: 3, textAlign: 'center' }}>
                  {t('common.noData')}
                </Typography>
              ) : (
                <List disablePadding>
                  {tracks.map((track, i) => (
                    <ListItem
                      key={track.Id || i}
                      disablePadding
                      sx={{
                        px: 2,
                        py: 1,
                        borderBottom: '1px solid',
                        borderColor: 'divider',
                        '&:last-child': { borderBottom: 0 },
                        gap: 2,
                        alignItems: 'center',
                        cursor: 'pointer',
                        transition: 'background 150ms',
                        '&:hover': { bgcolor: 'action.hover' },
                      }}
                      onClick={() => navigate(`/items/${track.Id}`)}
                    >
                      <Typography
                        variant="caption"
                        color="text.secondary"
                        sx={{ minWidth: 24, textAlign: 'right', flexShrink: 0, fontWeight: 600 }}
                      >
                        {track.IndexNumber ?? i + 1}
                      </Typography>
                      <Typography variant="body2" sx={{ flex: 1, fontWeight: 500 }} noWrap>
                        {track.Name}
                      </Typography>
                      {track.PlayCount > 0 && (
                        <Chip
                          icon={<Play24Regular style={{ fontSize: 12 }} />}
                          label={track.PlayCount}
                          size="small"
                          sx={{ fontSize: 11, height: 20, flexShrink: 0 }}
                        />
                      )}
                      <Typography
                        variant="caption"
                        color="text.secondary"
                        sx={{ minWidth: 36, textAlign: 'right', flexShrink: 0 }}
                      >
                        {formatTicks(track.RunTimeTicks)}
                      </Typography>
                    </ListItem>
                  ))}
                </List>
              )}
            </CardContent>
          </Card>
        </Box>
      </Box>
    </>
  )
}
