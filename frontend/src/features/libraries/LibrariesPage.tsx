import { useState, useEffect, useCallback } from 'react'
import { Grid, Alert, Card, CardActionArea, Typography, Chip, Skeleton, Box } from '@mui/material'
import {
  Library24Regular, VideoClip24Regular, MusicNote224Regular,
  Image24Regular, Tv24Regular,
} from '@fluentui/react-icons'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { format, parseISO } from 'date-fns'
import PageHeader from '@/shared/components/PageHeader/PageHeader'
import api from '@/lib/axios'
import type { Library } from '@/shared/types/library'
import { formatWatchTime } from '@/shared/utils/formatWatchTime'
import { formatSize } from '@/shared/utils/formatSize'
import { getDateLocale } from '@/lib/dateLocale'

// Cache des item IDs utilisés pour les fonds de carte (5 min TTL)
const SAMPLE_TTL = 5 * 60 * 1000
let sampleCache: Record<string, string> = {}
let sampleCacheTime = 0

const TYPE_ICONS: Record<string, React.ReactNode> = {
  movies:  <VideoClip24Regular style={{ fontSize: 22 }} />,
  tvshows: <Tv24Regular style={{ fontSize: 22 }} />,
  music:   <MusicNote224Regular style={{ fontSize: 22 }} />,
  photos:  <Image24Regular style={{ fontSize: 22 }} />,
}

function getTypeIcon(type: string): React.ReactNode {
  return TYPE_ICONS[type.toLowerCase()] ?? <Library24Regular style={{ fontSize: 22 }} />
}

function LibraryCard({ lib, sampleItemId, onClick }: { lib: Library; sampleItemId?: string; onClick: () => void }) {
  const { t } = useTranslation()
  const icon = getTypeIcon(lib.CollectionType)
  const [backdropFailed, setBackdropFailed] = useState(false)
  const [primaryFailed, setPrimaryFailed] = useState(false)

  // Try item backdrop first, then item primary, then give up (shows solid bg)
  const bgSrc = sampleItemId
    ? (!backdropFailed
        ? `/proxy/Items/Images/Backdrop/?id=${encodeURIComponent(sampleItemId)}&fillWidth=900&quality=70`
        : !primaryFailed
          ? `/proxy/Items/Images/Primary/?id=${encodeURIComponent(sampleItemId)}&fillWidth=900&quality=70`
          : null)
    : null

  return (
    <Card
      sx={{
        position: 'relative',
        overflow: 'hidden',
        borderRadius: 2,
        border: '1px solid',
        borderColor: 'divider',
        transition: 'border-color 180ms ease, box-shadow 180ms ease',
        '&:hover': {
          borderColor: 'primary.main',
          boxShadow: '0 4px 20px rgba(0,0,0,0.3)',
        },
      }}
    >
      <CardActionArea onClick={onClick} sx={{ display: 'block' }}>
        {/* Image area */}
        <Box sx={{ position: 'relative', paddingTop: '52%', overflow: 'hidden' }}>
          {/* Solid fallback (theme-aware) */}
          <Box
            sx={{
              position: 'absolute',
              inset: 0,
              bgcolor: 'background.paper',
            }}
          />

          {/* Blurred media image */}
          {bgSrc && (
            <Box
              component="img"
              src={bgSrc}
              alt=""
              onError={() => {
                if (!backdropFailed) setBackdropFailed(true)
                else setPrimaryFailed(true)
              }}
              sx={{
                position: 'absolute',
                inset: 0,
                width: '100%',
                height: '100%',
                objectFit: 'cover',
                filter: 'blur(12px)',
                transform: 'scale(1.12)',
                display: 'block',
                transition: 'filter 400ms ease, transform 400ms ease',
                '.MuiCard-root:hover &': {
                  filter: 'blur(5px)',
                  transform: 'scale(1.18)',
                },
              }}
            />
          )}

          {/* Gradient overlay — always on top of image */}
          <Box
            sx={{
              position: 'absolute',
              inset: 0,
              background: 'linear-gradient(to top, rgba(0,0,0,0.88) 0%, rgba(0,0,0,0.3) 55%, rgba(0,0,0,0.05) 100%)',
            }}
          />

          {/* Hover stats overlay */}
          <Box
            sx={{
              position: 'absolute',
              inset: 0,
              bgcolor: 'rgba(0,0,0,0.78)',
              backdropFilter: 'blur(2px)',
              opacity: 0,
              transition: 'opacity 250ms ease',
              '.MuiCard-root:hover &': { opacity: 1 },
              display: 'flex',
              flexDirection: 'column',
              justifyContent: 'center',
              px: 2.5,
              gap: 1.25,
            }}
          >
            <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 1.25 }}>
              {[
                { label: 'Total Plays', value: lib.TotalPlayCount != null ? lib.TotalPlayCount.toLocaleString() : '—' },
                { label: 'Watch Time', value: lib.TotalWatchTime ? formatWatchTime(lib.TotalWatchTime) : '—' },
                { label: 'Size', value: formatSize(lib.TotalSize) ?? '—' },
                {
                  label: 'Last Activity',
                  value: lib.LastActivity
                    ? (() => { try { return format(parseISO(lib.LastActivity), 'dd/MM/yyyy', { locale: getDateLocale() }) } catch { return lib.LastActivity } })()
                    : '—',
                },
              ].map(({ label, value }) => (
                <Box key={label}>
                  <Typography variant="caption" sx={{ color: 'rgba(255,255,255,0.45)', display: 'block', lineHeight: 1.3 }}>
                    {label}
                  </Typography>
                  <Typography variant="body2" sx={{ color: 'white', fontWeight: 600, lineHeight: 1.4 }}>
                    {value}
                  </Typography>
                </Box>
              ))}
            </Box>
          </Box>

          {/* Type chip — top right */}
          <Chip
            label={lib.CollectionType}
            size="small"
            sx={{
              position: 'absolute',
              top: 10,
              right: 10,
              height: 22,
              fontSize: 10,
              fontWeight: 700,
              textTransform: 'capitalize',
              letterSpacing: 0.5,
              bgcolor: 'rgba(0,0,0,0.55)',
              color: 'rgba(255,255,255,0.9)',
              border: '1px solid rgba(255,255,255,0.15)',
              backdropFilter: 'blur(4px)',
            }}
          />

          {/* Bottom info overlay */}
          <Box
            sx={{
              position: 'absolute',
              bottom: 0,
              left: 0,
              right: 0,
              px: 2,
              pb: 2,
              pt: 1,
            }}
          >
            {/* Icon + Name */}
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 0.75 }}>
              <Box sx={{ color: 'rgba(255,255,255,0.75)', flexShrink: 0, display: 'flex' }}>
                {icon}
              </Box>
              <Typography
                variant="h6"
                sx={{
                  fontWeight: 700,
                  color: 'white',
                  lineHeight: 1.2,
                  textShadow: '0 1px 4px rgba(0,0,0,0.6)',
                }}
                noWrap
              >
                {lib.Name}
              </Typography>
            </Box>

            {/* Stats row */}
            <Box sx={{ display: 'flex', gap: 1.5, flexWrap: 'wrap' }}>
              <Box
                sx={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 0.4,
                  bgcolor: 'rgba(255,255,255,0.1)',
                  border: '1px solid rgba(255,255,255,0.12)',
                  borderRadius: 1,
                  px: 0.75,
                  py: 0.25,
                  backdropFilter: 'blur(4px)',
                }}
              >
                <Typography variant="caption" sx={{ color: 'rgba(255,255,255,0.9)', fontWeight: 600, fontVariantNumeric: 'tabular-nums' }}>
                  {lib.ItemCount.toLocaleString()}
                </Typography>
                <Typography variant="caption" sx={{ color: 'rgba(255,255,255,0.55)' }}>
                  {t('common.items')}
                </Typography>
              </Box>

              {lib.EpisodeCount ? (
                <Box
                  sx={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 0.4,
                    bgcolor: 'rgba(255,255,255,0.1)',
                    border: '1px solid rgba(255,255,255,0.12)',
                    borderRadius: 1,
                    px: 0.75,
                    py: 0.25,
                    backdropFilter: 'blur(4px)',
                  }}
                >
                  <Typography variant="caption" sx={{ color: 'rgba(255,255,255,0.9)', fontWeight: 600, fontVariantNumeric: 'tabular-nums' }}>
                    {lib.EpisodeCount.toLocaleString()}
                  </Typography>
                  <Typography variant="caption" sx={{ color: 'rgba(255,255,255,0.55)' }}>
                    {t('common.episodes')}
                  </Typography>
                </Box>
              ) : null}

              {lib.SeasonCount ? (
                <Box
                  sx={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 0.4,
                    bgcolor: 'rgba(255,255,255,0.1)',
                    border: '1px solid rgba(255,255,255,0.12)',
                    borderRadius: 1,
                    px: 0.75,
                    py: 0.25,
                    backdropFilter: 'blur(4px)',
                  }}
                >
                  <Typography variant="caption" sx={{ color: 'rgba(255,255,255,0.9)', fontWeight: 600, fontVariantNumeric: 'tabular-nums' }}>
                    {lib.SeasonCount.toLocaleString()}
                  </Typography>
                  <Typography variant="caption" sx={{ color: 'rgba(255,255,255,0.55)' }}>
                    {t('library.seasons', 'saisons')}
                  </Typography>
                </Box>
              ) : null}
            </Box>
          </Box>
        </Box>
      </CardActionArea>
    </Card>
  )
}

export default function LibrariesPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [libraries, setLibraries] = useState<Library[]>([])
  const [sampleIds, setSampleIds] = useState<Record<string, string>>({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(() => {
    setLoading(true)
    api
      .get('/stats/getLibraries')
      .then(async (r) => {
        const libs: Library[] = r.data ?? []
        setLibraries(libs)

        // Utilise le cache si encore valide
        if (Date.now() - sampleCacheTime < SAMPLE_TTL && Object.keys(sampleCache).length > 0) {
          setSampleIds(sampleCache)
          return
        }

        // Fetch un échantillon d'items par librairie en parallèle
        const results = await Promise.allSettled(
          libs.map((lib) => api.get(`/api/libraries/${encodeURIComponent(lib.Id)}/items`))
        )
        const ids: Record<string, string> = {}
        results.forEach((res, i) => {
          if (res.status !== 'fulfilled') return
          const data = res.value.data
          const arr: Array<{ Id: string }> = Array.isArray(data) ? data : (data?.results ?? data?.items ?? [])
          if (arr.length === 0) return
          const pool = arr.slice(0, 30)
          ids[libs[i].Id] = pool[Math.floor(Math.random() * pool.length)].Id
        })

        sampleCache = ids
        sampleCacheTime = Date.now()
        setSampleIds(ids)
      })
      .catch(() => setError(t('common.loadError')))
      .finally(() => setLoading(false))
  }, [t])

  useEffect(() => { load() }, [load])

  return (
    <>
      <PageHeader title={t('nav.libraries')} onRefresh={load} loading={loading} />
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      <Grid container spacing={2}>
        {loading
          ? Array.from({ length: 6 }).map((_, i) => (
              <Grid key={i} size={{ xs: 12, sm: 6, lg: 4 }}>
                <Skeleton variant="rectangular" sx={{ borderRadius: 2, paddingTop: '52%' }} />
              </Grid>
            ))
          : libraries.map((lib) => (
              <Grid key={lib.Id} size={{ xs: 12, sm: 6, lg: 4 }}>
                <LibraryCard lib={lib} sampleItemId={sampleIds[lib.Id]} onClick={() => navigate(`/libraries/${lib.Id}`)} />
              </Grid>
            ))}
      </Grid>
    </>
  )
}
