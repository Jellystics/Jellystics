import { useState, useMemo } from 'react'
import type { ReactNode } from 'react'
import {
  Card, CardContent, CardHeader, Typography, Chip, Skeleton, Box,
  ToggleButtonGroup, ToggleButton,
} from '@mui/material'
import { VideoClip24Regular, MusicNote224Regular, Library24Regular } from '@fluentui/react-icons'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'

interface TopItem { Id: string; Name: string; PlayCount: number; Type: string }
interface TopContentProps { items: TopItem[]; loading: boolean; timeRangeSelector?: ReactNode }

type TypeFilter = 'all' | 'Movie' | 'Series' | 'Audio'

function TypeFallback({ type }: { type: string }) {
  if (type === 'Audio') return <MusicNote224Regular style={{ fontSize: 18, opacity: 0.5 }} />
  if (type === 'Episode' || type === 'Series') return <VideoClip24Regular style={{ fontSize: 18, opacity: 0.5 }} />
  return <Library24Regular style={{ fontSize: 18, opacity: 0.5 }} />
}

export default function TopContent({ items, loading, timeRangeSelector }: TopContentProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [typeFilter, setTypeFilter] = useState<TypeFilter>('all')

  const filtered = useMemo(() => {
    if (typeFilter === 'all') return items.slice(0, 5)
    if (typeFilter === 'Series') return items.filter((it) => it.Type === 'Episode' || it.Type === 'Series').slice(0, 5)
    return items.filter((it) => it.Type === typeFilter).slice(0, 5)
  }, [items, typeFilter])

  return (
    <Card>
      <CardHeader
        title={t('dashboard.topContent')}
        action={
          <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
            {timeRangeSelector}
            <ToggleButtonGroup
              size="small"
              exclusive
              value={typeFilter}
              onChange={(_, v) => { if (v) setTypeFilter(v) }}
              sx={{ height: 32, '& .MuiToggleButton-root': { px: 1.5, fontSize: 13, textTransform: 'none' } }}
            >
              <ToggleButton value="all">All</ToggleButton>
              <ToggleButton value="Movie">Movies</ToggleButton>
              <ToggleButton value="Series">Series</ToggleButton>
              <ToggleButton value="Audio">Music</ToggleButton>
            </ToggleButtonGroup>
          </Box>
        }
        slotProps={{ title: { variant: 'subtitle1', sx: { fontWeight: 600 } } }}
      />
      <CardContent sx={{ pt: 0 }}>
        {loading ? (
          Array.from({ length: 5 }).map((_, i) => (
            <Box key={i} sx={{ display: 'flex', gap: 1.5, mb: 1.5, alignItems: 'center' }}>
              <Skeleton variant="rectangular" width={36} height={52} sx={{ borderRadius: 1, flexShrink: 0 }} />
              <Box sx={{ flex: 1 }}>
                <Skeleton variant="text" width="70%" />
                <Skeleton variant="text" width="35%" />
              </Box>
            </Box>
          ))
        ) : filtered.length === 0 ? (
          <Typography variant="body2" color="text.secondary" sx={{ py: 2, textAlign: 'center' }}>
            {t('common.noData', 'No data')}
          </Typography>
        ) : (
          filtered.map((item, i) => (
            <Box
              key={`${typeFilter}-${item.Id}`}
              onClick={() => navigate(`/items/${item.Id}`)}
              sx={{
                display: 'flex',
                alignItems: 'center',
                gap: 1.5,
                py: 0.75,
                borderBottom: '1px solid',
                borderColor: 'divider',
                '&:last-child': { borderBottom: 0 },
                cursor: 'pointer',
                borderRadius: 1,
                mx: -0.5,
                px: 0.5,
                transition: 'background 150ms',
                '&:hover': { bgcolor: 'action.hover' },
              }}
            >
              {/* Rank */}
              <Typography
                variant="caption"
                color="text.secondary"
                sx={{ minWidth: 16, textAlign: 'right', flexShrink: 0, fontWeight: 600, fontSize: 11 }}
              >
                {i + 1}
              </Typography>

              {/* Poster */}
              <Box
                sx={{
                  position: 'relative',
                  width: 36,
                  height: 52,
                  borderRadius: 1,
                  overflow: 'hidden',
                  flexShrink: 0,
                  bgcolor: 'rgba(255,255,255,0.05)',
                }}
              >
                {/* Fallback icon underneath */}
                <Box sx={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                  <TypeFallback type={item.Type} />
                </Box>
                {/* Poster image on top — hides itself on error, revealing fallback */}
                <img
                  src={`/proxy/Items/Images/Primary/?id=${item.Id}&fillWidth=72&quality=85`}
                  alt={item.Name}
                  style={{ position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover' }}
                  onError={(e) => { e.currentTarget.style.display = 'none' }}
                />
              </Box>

              {/* Title + type */}
              <Box sx={{ flex: 1, minWidth: 0 }}>
                <Typography variant="body2" sx={{ fontSize: 13, fontWeight: 500 }} noWrap>
                  {item.Name}
                </Typography>
                <Typography variant="caption" color="text.secondary" sx={{ fontSize: 11 }}>
                  {item.Type}
                </Typography>
              </Box>

              {/* Play count */}
              <Chip
                label={`${item.PlayCount} ${t('common.plays')}`}
                size="small"
                sx={{ fontSize: 11, height: 20, flexShrink: 0 }}
              />
            </Box>
          ))
        )}
      </CardContent>
    </Card>
  )
}
