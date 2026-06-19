import { createTheme, alpha } from '@mui/material/styles'
import type { ThemeMode } from './ThemeModeContext'

const MODE_KEY = 'jellystics-theme-mode'
const DEFAULT_PRIMARY = '#64748b'

export function getThemeMode(): ThemeMode {
  const stored = localStorage.getItem(MODE_KEY) as ThemeMode | null
  if (stored) return stored
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

export function setThemeMode(mode: ThemeMode): void {
  localStorage.setItem(MODE_KEY, mode)
}

export function buildTheme(mode: ThemeMode = getThemeMode()) {
  const isDark = mode === 'dark'

  return createTheme({
    palette: {
      mode,
      background: isDark
        ? { default: '#111114', paper: '#18181f' }
        : { default: '#f0f2f5', paper: '#ffffff' },
      primary: { main: DEFAULT_PRIMARY },
      secondary: { main: '#475569' },
      divider: isDark ? 'rgba(255,255,255,0.08)' : 'rgba(0,0,0,0.08)',
      text: isDark
        ? { primary: '#e8e8f0', secondary: '#8b8b9e' }
        : { primary: '#111827', secondary: '#6b7280' },
    },
    typography: {
      fontFamily: '"Inter", "Roboto", "Helvetica", "Arial", sans-serif',
      h4: { fontWeight: 600 },
      h5: { fontWeight: 600 },
      h6: { fontWeight: 600 },
    },
    shape: { borderRadius: 12 },
    components: {
      MuiCssBaseline: {
        styleOverrides: {
          body: { overscrollBehavior: 'none' },
        },
      },
      MuiButton: {
        styleOverrides: {
          root: { textTransform: 'none', fontWeight: 500 },
        },
        defaultProps: {
          disableElevation: true,
        },
      },
      MuiListItemButton: {
        styleOverrides: {
          root: { borderRadius: 12 },
        },
      },
      MuiOutlinedInput: {
        styleOverrides: {
          root: { borderRadius: 12 },
        },
      },
      MuiMenu: {
        styleOverrides: {
          paper: { borderRadius: 12 },
          list: { padding: '4px 0' },
        },
        defaultProps: {
          slotProps: { paper: { elevation: 3 } },
        },
      },
      MuiMenuItem: {
        styleOverrides: {
          root: {
            borderRadius: 12,
            margin: '0px 4px',
            paddingLeft: '8px',
            paddingRight: '8px',
          },
        },
      },
      MuiTooltip: {
        defaultProps: { enterDelay: 500 },
      },
      MuiSkeleton: {
        defaultProps: { animation: 'wave' },
      },
      MuiDialogContent: {
        styleOverrides: {
          root: { paddingTop: 0 },
        },
      },
      MuiTableCell: {
        styleOverrides: {
          root: ({ theme }) => ({
            borderColor: theme.palette.divider,
          }),
        },
      },
      MuiPaper: {
        styleOverrides: {
          root: { backgroundImage: 'none' },
        },
      },
      MuiTabs: {
        styleOverrides: {
          root: {
            minHeight: 36,
            overflow: 'initial',
            '& .MuiTabs-scroller': { overflow: 'initial !important' },
          },
          indicator: { bottom: 'initial' },
        },
      },
      MuiTab: {
        styleOverrides: {
          root: ({ theme }) => ({
            textTransform: 'none',
            padding: '8px 0px',
            overflow: 'initial',
            minHeight: 36,
            minWidth: 0,
            marginRight: 32,
            '&:last-of-type': { marginRight: 0 },
            transition: theme.transitions.create(['background-color', 'color']),
            '&::after': {
              content: "''",
              borderRadius: 8,
              position: 'absolute',
              top: 4,
              bottom: 4,
              left: -8,
              right: -8,
              transition: theme.transitions.create(['background-color']),
            },
            '&.MuiButtonBase-root .MuiTouchRipple-root': {
              borderRadius: 8,
              top: 4,
              bottom: 4,
              left: -8,
              right: -8,
            },
            '&:hover': {
              '&:not(.Mui-selected)': { color: theme.palette.text.primary },
              '&::after': { backgroundColor: theme.palette.action.hover },
              '&.Mui-selected::after': { backgroundColor: alpha(theme.palette.primary.main, 0.06) },
            },
          }),
        },
      },
      MuiCard: {
        styleOverrides: {
          root: ({ theme }) => ({
            backgroundImage: 'none',
            boxShadow: 'none',
            border: `1px solid ${theme.palette.divider}`,
          }),
        },
      },
    },
  })
}

