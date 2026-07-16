import { Box } from '@mui/material'
import type { SxProps, Theme } from '@mui/material'
import { VideoClip24Regular, MusicNote224Regular, Library24Regular } from '@fluentui/react-icons'

interface MediaPosterProps {
  /** Image URL — null/undefined renders only the fallback icon */
  src: string | null | undefined
  alt?: string
  /** Jellyfin item type — used to pick the fallback icon */
  type?: string
  width?: number
  height?: number
  sx?: SxProps<Theme>
}

function FallbackIcon({ type, size }: { type?: string; size: number }) {
  const style = { fontSize: size, opacity: 0.4 } as const
  if (type === 'Audio') return <MusicNote224Regular style={style} />
  if (type === 'Movie' || type === 'Episode' || type === 'Series') return <VideoClip24Regular style={style} />
  return <Library24Regular style={style} />
}

export default function MediaPoster({ src, alt, type, width = 36, height = 52, sx }: MediaPosterProps) {
  const iconSize = Math.max(12, Math.round(Math.min(width, height) * 0.45))
  return (
    <Box
      sx={{
        position: 'relative',
        width,
        height,
        borderRadius: 1,
        overflow: 'hidden',
        flexShrink: 0,
        bgcolor: 'rgba(255,255,255,0.06)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        ...sx,
      }}
    >
      <FallbackIcon type={type} size={iconSize} />
      {src && (
        <Box
          component="img"
          src={src}
          alt={alt}
          onError={(e: React.SyntheticEvent<HTMLImageElement>) => { e.currentTarget.style.display = 'none' }}
          sx={{ position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover' }}
        />
      )}
    </Box>
  )
}
