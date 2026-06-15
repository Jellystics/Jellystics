import { useState, useEffect } from 'react'
import socket from '@/lib/socket'
import {
  Box, AppBar, Toolbar, IconButton, Drawer, Container,
  useTheme, useMediaQuery, Collapse, Tooltip,
} from '@mui/material'
import { Navigation24Regular, WeatherMoon24Regular, WeatherSunny24Regular } from '@fluentui/react-icons'
import { Outlet } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import SidebarContent from './Sidebar'
import UserMenu from './UserMenu'
import { useThemeMode } from '@/lib/ThemeModeContext'

const DRAWER_WIDTH = 240

export default function AppShell() {
  const { t } = useTranslation()
  const theme = useTheme()
  const isMobile = useMediaQuery(theme.breakpoints.down('md'))
  const [open, setOpen] = useState(true)
  const [mobileOpen, setMobileOpen] = useState(false)
  const { mode, toggleMode } = useThemeMode()

  useEffect(() => {
    socket.connect()
    return () => { socket.disconnect() }
  }, [])

  const appBarBg = theme.palette.background.default

  const drawerContent = (
    <SidebarContent onClose={() => (isMobile ? setMobileOpen(false) : setOpen(false))} />
  )

  return (
    <Box sx={{ display: 'flex', bgcolor: 'background.default', minHeight: '100vh' }}>
      {/* Fixed TopAppBar */}
      <AppBar
        position="fixed"
        elevation={0}
        sx={(t) => ({
          bgcolor: appBarBg,
          color: 'text.primary',
          transition: t.transitions.create(['margin', 'width'], {
            easing: t.transitions.easing.sharp,
            duration: t.transitions.duration.leavingScreen,
          }),
          ...(open && !isMobile && {
            width: `calc(100% - ${DRAWER_WIDTH}px)`,
            marginLeft: `${DRAWER_WIDTH}px`,
            transition: t.transitions.create(['margin', 'width'], {
              easing: t.transitions.easing.easeOut,
              duration: t.transitions.duration.enteringScreen,
            }),
          }),
        })}
      >
        <Toolbar>
          {/* Hamburger — shows only when drawer is closed */}
          <Collapse orientation="horizontal" in={!open || isMobile}>
            <IconButton
              color="inherit"
              onClick={() => isMobile ? setMobileOpen(true) : setOpen(true)}
              edge="start"
              sx={{ mr: 1 }}
            >
              <Navigation24Regular style={{ fontSize: 22 }} />
            </IconButton>
          </Collapse>
          <Box sx={{ flexGrow: 1 }} />
          <Tooltip title={mode === 'dark' ? t('theme.lightMode') : t('theme.darkMode')} enterDelay={500}>
            <IconButton color="inherit" onClick={toggleMode} sx={{ mr: 0.5 }}>
              {mode === 'dark'
                ? <WeatherSunny24Regular style={{ fontSize: 20 }} />
                : <WeatherMoon24Regular style={{ fontSize: 20 }} />}
            </IconButton>
          </Tooltip>
          <UserMenu />
        </Toolbar>
      </AppBar>

      {/* Mobile temporary drawer */}
      <Drawer
        variant="temporary"
        open={mobileOpen}
        onClose={() => setMobileOpen(false)}
        ModalProps={{ keepMounted: true }}
        sx={{
          display: { xs: 'block', md: 'none' },
          '& .MuiDrawer-paper': {
            width: DRAWER_WIDTH,
            bgcolor: 'background.default',
            border: 'none',
            boxSizing: 'border-box',
          },
        }}
      >
        {drawerContent}
      </Drawer>

      {/* Desktop persistent drawer */}
      <Drawer
        variant="persistent"
        open={open}
        sx={{
          display: { xs: 'none', md: 'block' },
          width: DRAWER_WIDTH,
          flexShrink: 0,
          '& .MuiDrawer-paper': {
            width: DRAWER_WIDTH,
            bgcolor: 'background.default',
            border: 'none',
            boxSizing: 'border-box',
          },
        }}
      >
        {drawerContent}
      </Drawer>

      {/* Main content */}
      <Box
        component="main"
        sx={(t) => ({
          flexGrow: 1,
          display: 'flex',
          flexDirection: 'column',
          height: '100vh',
          overflow: 'hidden',
          marginRight: { md: 2 },
          transition: t.transitions.create('margin', {
            easing: t.transitions.easing.sharp,
            duration: t.transitions.duration.leavingScreen,
          }),
          marginLeft: isMobile ? 0 : open ? 0 : `${-DRAWER_WIDTH + 16}px`,
          ...(open && !isMobile && {
            transition: t.transitions.create('margin', {
              easing: t.transitions.easing.easeOut,
              duration: t.transitions.duration.enteringScreen,
            }),
          }),
        })}
      >
        {/* AppBar spacer */}
        <Box sx={{ ...theme.mixins.toolbar }} />

        {/* Page container */}
        <Box sx={{ flexGrow: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden', mb: { md: 1 } }}>
          <Box
            sx={{
              flexGrow: 1,
              overflow: 'auto',
              bgcolor: 'background.paper',
              borderRadius: { md: 3 },
              border: { md: '1px solid' },
              borderColor: { md: 'divider' },
              py: 4,
            }}
          >
            <Container maxWidth="xl">
              <Outlet />
            </Container>
          </Box>
        </Box>
      </Box>
    </Box>
  )
}
