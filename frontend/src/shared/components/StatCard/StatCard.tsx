import { Box, Paper, Avatar, Typography, Skeleton } from '@mui/material'
import { alpha } from '@mui/material/styles'
import { useTheme } from '@mui/material/styles'

interface StatCardProps {
  label: string
  value: string | number
  icon?: React.ReactNode
  loading?: boolean
  subtitle?: string
  color?: string
}

export default function StatCard({ label, value, icon, loading, subtitle, color }: StatCardProps) {
  const theme = useTheme()
  const accentColor = color ?? theme.palette.primary.main

  return (
    <Paper
      sx={{
        p: 2.5,
        boxShadow: 'none',
        border: '1px solid',
        borderColor: 'divider',
        display: 'flex',
        alignItems: 'center',
        gap: 2,
      }}
    >
      {icon && (
        <Avatar
          sx={{
            bgcolor: alpha(accentColor, 0.12),
            color: accentColor,
            width: 44,
            height: 44,
            flexShrink: 0,
            fontSize: 22,
          }}
        >
          {icon}
        </Avatar>
      )}
      <Box sx={{ minWidth: 0 }}>
        <Typography variant="body2" color="text.secondary" noWrap>
          {label}
        </Typography>
        {loading ? (
          <Skeleton variant="text" width={64} height={32} />
        ) : (
          <Typography variant="h6" sx={{ fontWeight: 600, lineHeight: 1.2 }}>
            {value}
          </Typography>
        )}
        {subtitle && (
          <Typography variant="caption" color="text.secondary">
            {subtitle}
          </Typography>
        )}
      </Box>
    </Paper>
  )
}
