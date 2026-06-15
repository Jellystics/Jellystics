import { Box, Paper, Typography, Divider, Skeleton } from '@mui/material'

interface ChartCardProps {
  title: string
  children: React.ReactNode
  loading?: boolean
  height?: number
  action?: React.ReactNode
}

export default function ChartCard({ title, children, loading, height = 280, action }: ChartCardProps) {
  return (
    <Paper sx={{ boxShadow: 'none', border: '1px solid', borderColor: 'divider', borderRadius: 3 }}>
      <Box
        sx={{
          px: 3,
          pt: 2.5,
          pb: 0,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
          {title}
        </Typography>
        {action && action}
      </Box>
      <Divider sx={{ mt: 1.5, mb: 0 }} />
      <Box sx={{ px: 3, py: 2.5 }}>
        {loading ? (
          <Skeleton variant="rectangular" width="100%" height={height} sx={{ borderRadius: 2 }} />
        ) : (
          <Box sx={{ height, minWidth: 0, minHeight: height, width: '100%' }}>{children}</Box>
        )}
      </Box>
    </Paper>
  )
}
