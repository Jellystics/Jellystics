import { createTheme } from '@mui/material/styles'
import type { ThemeMode } from './ThemeModeContext'

const ACCENT_KEY = 'jellystics-accent-color'
const MODE_KEY = 'jellystics-theme-mode'
const DEFAULT_ACCENT = '#a78bfa'

export function getAccentColor(): string {
  return localStorage.getItem(ACCENT_KEY) ?? DEFAULT_ACCENT
}

export function setAccentColor(color: string): void {
  localStorage.setItem(ACCENT_KEY, color)
}

export function getThemeMode(): ThemeMode {
  return (localStorage.getItem(MODE_KEY) as ThemeMode) ?? 'dark'
}

export function setThemeMode(mode: ThemeMode): void {
  localStorage.setItem(MODE_KEY, mode)
}

export function buildTheme(accent: string = getAccentColor(), mode: ThemeMode = getThemeMode()) {
  const isDark = mode === 'dark'

  return createTheme({
    palette: {
      mode,
      background: isDark
        ? { default: '#111114', paper: '#18181f' }
        : { default: '#f0f2f5', paper: '#ffffff' },
      primary: { main: accent, contrastText: '#ffffff' },
      secondary: { main: '#7c3aed' },
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
      MuiTab: {
        styleOverrides: {
          root: { textTransform: 'none' },
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

export const theme = buildTheme()
