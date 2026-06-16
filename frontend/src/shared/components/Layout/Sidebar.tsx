import { useState } from 'react'
import { useLogo } from '@/lib/FaviconContext'
import {
  Box, Stack, Typography, IconButton, Tooltip,
  ButtonBase, Fade, darken,
} from '@mui/material'
import { useTheme } from '@mui/material/styles'
import { useNavigate, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  Home20Regular, Home20Filled,
  CalendarClock20Regular, CalendarClock20Filled,
  VideoClip20Regular, VideoClip20Filled,
  Library20Regular, Library20Filled,
  People20Regular, People20Filled,
  DataPie20Regular, DataPie20Filled,
  Settings20Regular, Settings20Filled,
  ChevronLeft24Regular,
} from '@fluentui/react-icons'

interface NavItem {
  key: string
  path: string
  icon: React.ReactNode
  iconActive: React.ReactNode
  labelKey: string
}

const navItems: NavItem[] = [
  { key: 'dashboard', path: '/', icon: <Home20Regular />, iconActive: <Home20Filled />, labelKey: 'nav.dashboard' },
  { key: 'activity', path: '/activity', icon: <CalendarClock20Regular />, iconActive: <CalendarClock20Filled />, labelKey: 'nav.activity' },
  { key: 'timeline', path: '/activity/timeline', icon: <VideoClip20Regular />, iconActive: <VideoClip20Filled />, labelKey: 'nav.timeline' },
  { key: 'libraries', path: '/libraries', icon: <Library20Regular />, iconActive: <Library20Filled />, labelKey: 'nav.libraries' },
  { key: 'users', path: '/users', icon: <People20Regular />, iconActive: <People20Filled />, labelKey: 'nav.users' },
  { key: 'statistics', path: '/statistics', icon: <DataPie20Regular />, iconActive: <DataPie20Filled />, labelKey: 'nav.statistics' },
  { key: 'settings', path: '/settings', icon: <Settings20Regular />, iconActive: <Settings20Filled />, labelKey: 'nav.settings' },
]

interface SidebarContentProps {
  onClose: () => void
}

export default function SidebarContent({ onClose }: SidebarContentProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const location = useLocation()
  const theme = useTheme()
  const [showCollapse, setShowCollapse] = useState(false)
  const { logoUrl } = useLogo()

  const activeColor = darken(theme.palette.primary.main, 0.45)

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* Header — same height as AppBar (theme.mixins.toolbar) */}
      <Box
        onMouseEnter={() => setShowCollapse(true)}
        onMouseLeave={() => setShowCollapse(false)}
        sx={{
          ...theme.mixins.toolbar,
          display: 'flex',
          alignItems: 'center',
          px: 2,
          justifyContent: 'space-between',
          flexShrink: 0,

        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <Box component="img" src={logoUrl} alt="logo" sx={{ width: 40, height: 40, objectFit: 'contain' }} />
          <Typography variant="h6" sx={{ fontWeight: 700, letterSpacing: '-0.01em' }}>
            Jellystics
          </Typography>
        </Box>
        <Fade in={showCollapse}>
          <IconButton size="small" onClick={onClose} sx={{ color: 'text.disabled' }}>
            <ChevronLeft24Regular style={{ fontSize: 18 }} />
          </IconButton>
        </Fade>
      </Box>

      {/* Nav items */}
      <Stack
        direction="column"
        spacing={0}
        sx={{ px: 1, pt: 1, pb: 1, flexGrow: 1, overflow: 'auto' }}
      >
        {(() => {
          const activeItem = navItems
            .filter((item) =>
              item.path === '/'
                ? location.pathname === '/'
                : location.pathname === item.path || location.pathname.startsWith(item.path + '/')
            )
            .sort((a, b) => b.path.length - a.path.length)[0]

          return navItems.map((item) => {
          const isActive = activeItem?.key === item.key

          return (
            <Tooltip key={item.key} title="" placement="right">
              <ButtonBase
                onClick={() => navigate(item.path)}
                sx={{
                  borderRadius: '90px',
                  display: 'flex',
                  justifyContent: 'flex-start',
                  alignItems: 'center',
                  width: '100%',
                  height: 32,
                  px: '4px',
                  pl: '14px',
                  mb: '2px',
                  color: isActive ? '#ffffff' : 'text.secondary',
                  bgcolor: isActive ? activeColor : 'transparent',
                  transition: 'background-color 200ms cubic-bezier(0.4,0,0.2,1)',
                  '&:hover': {
                    bgcolor: isActive ? activeColor : 'action.hover',
                    color: isActive ? '#ffffff' : 'text.primary',
                  },
                }}
              >
                <Box sx={{ width: 20, height: 20, mr: '14px', display: 'flex', alignItems: 'center', flexShrink: 0 }}>
                  {isActive ? item.iconActive : item.icon}
                </Box>
                <Typography
                  variant="body2"
                  noWrap
                  sx={{ fontWeight: isActive ? 600 : 400, color: 'inherit' }}
                >
                  {t(item.labelKey)}
                </Typography>
              </ButtonBase>
            </Tooltip>
          )
        })
        })()}
      </Stack>
    </Box>
  )
}
