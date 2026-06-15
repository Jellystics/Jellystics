import { Box, Typography, IconButton, Tooltip } from '@mui/material'
import { ArrowSync24Regular } from '@fluentui/react-icons'
import { useTranslation } from 'react-i18next'

interface PageHeaderProps {
  title: string
  actions?: React.ReactNode
  onRefresh?: () => void
  loading?: boolean
}

export default function PageHeader({ title, actions, onRefresh, loading }: PageHeaderProps) {
  const { t } = useTranslation()
  return (
    <Box sx={{ mb: 4, display: 'flex', alignItems: 'center' }}>
      <Typography variant="h4" sx={{ fontWeight: 600 }}>
        {title}
      </Typography>
      {onRefresh && (
        <Tooltip title={t('common.refresh')}>
          <IconButton onClick={onRefresh} disabled={loading} sx={{ ml: 1 }}>
            <ArrowSync24Regular />
          </IconButton>
        </Tooltip>
      )}
      <Box sx={{ flexGrow: 1 }} />
      {actions && actions}
    </Box>
  )
}
