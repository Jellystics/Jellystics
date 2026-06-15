import { useState } from 'react'
import {
  IconButton, Popover, Box, Typography, Divider, MenuList, MenuItem,
  ListItemIcon, ListItemText, Avatar, alpha,
} from '@mui/material'
import { useTheme } from '@mui/material/styles'
import { Settings24Regular, SignOut24Regular } from '@fluentui/react-icons'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useCurrentUser } from './useCurrentUser'

export default function UserMenu() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const theme = useTheme()
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null)
  const { username, avatarUrl } = useCurrentUser()

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
            {username || 'User'}
          </Typography>
          <Typography variant="caption" color="text.secondary" sx={{ mt: 0.5, display: 'block' }}>
            Administrator
          </Typography>
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
