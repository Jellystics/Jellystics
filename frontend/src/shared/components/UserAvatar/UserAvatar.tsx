import { Avatar } from '@mui/material'
import type { SxProps, Theme } from '@mui/material'
import { getUserImageUrl } from '@/shared/utils/imageUrl'

interface UserAvatarProps {
  userId: string
  userName: string
  /** Diameter in px — fontSize scales automatically */
  size?: number
  sx?: SxProps<Theme>
}

export default function UserAvatar({ userId, userName, size = 32, sx }: UserAvatarProps) {
  return (
    <Avatar
      src={getUserImageUrl(userId, size * 2)}
      sx={{
        width: size,
        height: size,
        bgcolor: 'primary.main',
        fontSize: Math.round(size * 0.41),
        fontWeight: 700,
        flexShrink: 0,
        ...sx,
      }}
    >
      {userName.charAt(0).toUpperCase()}
    </Avatar>
  )
}
