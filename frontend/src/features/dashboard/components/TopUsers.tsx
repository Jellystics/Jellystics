import {
  Card, CardContent, CardHeader, List, ListItem, ListItemText,
  ListItemAvatar, Avatar, Chip, Skeleton, Box, Typography,
} from '@mui/material'
import { useTranslation } from 'react-i18next'

interface TopUser { UserId: string; UserName: string; TotalPlays: number; TotalWatchTime: number }
interface TopUsersProps { users: TopUser[]; loading: boolean }

function formatWatchTime(minutes: number): string {
  if (minutes < 60) return `${minutes}m`
  const h = Math.floor(minutes / 60)
  const m = minutes % 60
  return m > 0 ? `${h}h ${m}m` : `${h}h`
}

export default function TopUsers({ users, loading }: TopUsersProps) {
  const { t } = useTranslation()
  return (
    <Card>
      <CardHeader title={t('dashboard.topUsers')} slotProps={{ title: { variant: 'subtitle1', sx: { fontWeight: 600 } } }} />
      <CardContent sx={{ pt: 0 }}>
        {loading ? (
          Array.from({ length: 5 }).map((_, i) => (
            <Box key={i} sx={{ display: 'flex', gap: 1, mb: 1 }}>
              <Skeleton variant="circular" width={36} height={36} />
              <Box sx={{ flex: 1 }}><Skeleton variant="text" width="60%" /><Skeleton variant="text" width="35%" /></Box>
            </Box>
          ))
        ) : (
          <List dense disablePadding>
            {users.map((user, i) => (
              <ListItem key={user.UserId} disablePadding sx={{ py: 0.5 }}>
                <Typography variant="caption" color="text.secondary" sx={{ minWidth: 20, mr: 1 }}>{i + 1}</Typography>
                <ListItemAvatar sx={{ minWidth: 44 }}>
                  <Avatar sx={{ width: 32, height: 32, bgcolor: 'primary.main', fontSize: 13, fontWeight: 700 }}>
                    {user.UserName.charAt(0).toUpperCase()}
                  </Avatar>
                </ListItemAvatar>
                <ListItemText
                  primary={<Typography variant="body2" sx={{ fontSize: 13, fontWeight: 500 }}>{user.UserName}</Typography>}
                  secondary={<Typography variant="caption" sx={{ fontSize: 11 }}>{formatWatchTime(user.TotalWatchTime)}</Typography>}
                />
                <Chip label={`${user.TotalPlays} ${t('common.plays')}`} size="small" sx={{ fontSize: 11, height: 20 }} />
              </ListItem>
            ))}
          </List>
        )}
      </CardContent>
    </Card>
  )
}
