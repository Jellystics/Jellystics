import {
  Card, CardContent, CardHeader, List, ListItem, ListItemText,
  ListItemAvatar, Avatar, Typography, Chip, Skeleton, Box,
} from '@mui/material'
import { VideoClip24Regular, MusicNote224Regular, Library24Regular } from '@fluentui/react-icons'
import { useTranslation } from 'react-i18next'

interface TopItem { Id: string; Name: string; PlayCount: number; Type: string }
interface TopContentProps { items: TopItem[]; loading: boolean }

function typeIcon(type: string) {
  if (type === 'Audio') return <MusicNote224Regular />
  if (type === 'Episode' || type === 'Series') return <VideoClip24Regular />
  return <Library24Regular />
}

export default function TopContent({ items, loading }: TopContentProps) {
  const { t } = useTranslation()
  return (
    <Card>
      <CardHeader title={t('dashboard.topContent')} slotProps={{ title: { variant: 'subtitle1', sx: { fontWeight: 600 } } }} />
      <CardContent sx={{ pt: 0 }}>
        {loading ? (
          Array.from({ length: 5 }).map((_, i) => (
            <Box key={i} sx={{ display: 'flex', gap: 1, mb: 1 }}>
              <Skeleton variant="circular" width={36} height={36} />
              <Box sx={{ flex: 1 }}><Skeleton variant="text" width="70%" /><Skeleton variant="text" width="40%" /></Box>
            </Box>
          ))
        ) : (
          <List dense disablePadding>
            {items.map((item, i) => (
              <ListItem key={item.Id} disablePadding sx={{ py: 0.5 }}>
                <Typography variant="caption" color="text.secondary" sx={{ minWidth: 20, mr: 1 }}>{i + 1}</Typography>
                <ListItemAvatar sx={{ minWidth: 44 }}>
                  <Avatar sx={{ width: 32, height: 32, bgcolor: 'rgba(255,255,255,0.05)', fontSize: 16 }}>{typeIcon(item.Type)}</Avatar>
                </ListItemAvatar>
                <ListItemText
                  primary={<Typography variant="body2" sx={{ fontSize: 13, fontWeight: 500 }} noWrap>{item.Name}</Typography>}
                  secondary={<Typography variant="caption" sx={{ fontSize: 11 }}>{item.Type}</Typography>}
                />
                <Chip label={`${item.PlayCount} ${t('common.plays')}`} size="small" sx={{ fontSize: 11, height: 20 }} />
              </ListItem>
            ))}
          </List>
        )}
      </CardContent>
    </Card>
  )
}
