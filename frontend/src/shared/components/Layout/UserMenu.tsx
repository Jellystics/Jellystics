import { useState } from 'react'
import {
  IconButton, Popover, Box, Typography, Divider, MenuList, MenuItem,
  ListItemIcon, ListItemText, Avatar, alpha, Tooltip,
} from '@mui/material'
import { useTheme } from '@mui/material/styles'
import { Settings24Regular, SignOut24Regular } from '@fluentui/react-icons'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useCurrentUser } from './useCurrentUser'
import { usePalette } from '@/lib/PaletteContext'
import { getAllPalettes } from '@/lib/palette'

export default function UserMenu() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const theme = useTheme()
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null)
  const { username, avatarUrl } = useCurrentUser()
  const { paletteId, setPaletteId } = usePalette()
  const palettes = getAllPalettes()
  const isDark = theme.palette.mode === 'dark'

  const handleLogout = () => {
    setAnchorEl(null)
    localStorage.removeItem('jellystics-token')
    localStorage.removeItem('jellystics-username')
    window.location.href = '/login'
  }

  const initial = username ? username.charAt(0).toUpperCase() : '?'

  return (
    <>
      <IconButton size="large" onClick={(e) => setAnchorEl(e.currentTarget)}>
        <Avatar
          src={avatarUrl}
          sx={{
            width: 30,
            height: 30,
            fontSize: 13,
            fontWeight: 700,
            bgcolor: alpha(theme.palette.primary.main, 0.15),
            color: 'primary.main',
          }}
        >
          {initial}
        </Avatar>
      </IconButton>

      <Popover
        open={Boolean(anchorEl)}
        anchorEl={anchorEl}
        onClose={() => setAnchorEl(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'right' }}
      >
        {/* User info header */}
        <Box sx={{ px: 1.5, py: 1.25, minWidth: 220 }}>
          <Typography variant="body2" sx={{ fontWeight: 600, lineHeight: 1 }}>
            {username || t('common.user')}
          </Typography>
          <Typography variant="caption" color="text.secondary" sx={{ mt: 0.5, display: 'block' }}>
            {t('common.administrator')}
          </Typography>
        </Box>

        <Divider />

        {/* Palette picker */}
        <Box sx={{ px: 1.5, py: 1, display: 'flex', gap: 1, flexWrap: 'wrap' }}>
          {palettes.map((p) => {
            const color = isDark ? p.dark : p.light
            const selected = paletteId === p.id
            return (
              <Tooltip key={p.id} title={p.label} placement="top">
                <Box
                  onClick={() => setPaletteId(p.id)}
                  sx={{
                    width: 20,
                    height: 20,
                    borderRadius: '50%',
                    background: `linear-gradient(135deg, ${p.light} 50%, ${p.dark} 50%)`,
                    cursor: 'pointer',
                    boxShadow: selected
                      ? `0 0 0 2px ${theme.palette.background.paper}, 0 0 0 3.5px ${color}`
                      : 'none',
                    transition: 'transform 0.15s, box-shadow 0.15s',
                    '&:hover': { transform: 'scale(1.2)' },
                  }}
                />
              </Tooltip>
            )
          })}
        </Box>

        <Divider />

        <MenuList dense sx={{ mx: 0.5, py: 0.5 }}>
          <MenuItem onClick={() => { setAnchorEl(null); navigate('/settings') }}>
            <ListItemIcon>
              <Settings24Regular style={{ fontSize: 18 }} />
            </ListItemIcon>
            <ListItemText>{t('nav.settings')}</ListItemText>
          </MenuItem>
          <MenuItem
            onClick={handleLogout}
            sx={{ color: 'error.main', '& .MuiListItemIcon-root': { color: 'error.main' } }}
          >
            <ListItemIcon>
              <SignOut24Regular style={{ fontSize: 18 }} />
            </ListItemIcon>
            <ListItemText>{t('nav.logout')}</ListItemText>
          </MenuItem>
        </MenuList>
      </Popover>
    </>
  )
}
