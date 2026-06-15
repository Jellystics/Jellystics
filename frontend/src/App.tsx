import { useEffect, useState } from 'react'
import { RouterProvider } from 'react-router-dom'
import { ThemeProvider } from '@mui/material/styles'
import CssBaseline from '@mui/material/CssBaseline'
import GlobalStyles from '@mui/material/GlobalStyles'
import { SnackbarProvider } from 'notistack'
import { useTheme, useMediaQuery } from '@mui/material'
import { grey } from '@mui/material/colors'
import { buildTheme, getAccentColor, getThemeMode, setThemeMode } from '@/lib/theme'
import { ThemeModeContext, type ThemeMode } from '@/lib/ThemeModeContext'
import { router } from '@/lib/router'
import '@/lib/i18n'
import socket from '@/lib/socket'
import { useSnackbar } from 'notistack'

const TASK_EVENTS = [
  'PlaybackSyncTask',
  'PartialSyncTask',
  'FullSyncTask',
  'BackupTask',
  'TaskError',
  'GeneralAlert',
]

function SocketNotifier() {
  const { enqueueSnackbar } = useSnackbar()

  useEffect(() => {
    const handlers: Array<(msg: unknown) => void> = []

    TASK_EVENTS.forEach((event) => {
      const handler = (msg: unknown) => {
        const m = msg as { type?: string; message?: string } | string
        const text = typeof m === 'string' ? m : m?.message ?? String(m)
        const type = typeof m === 'object' && m !== null ? (m as { type?: string }).type : undefined

        if (type === 'Success') enqueueSnackbar(text, { variant: 'success' })
        else if (type === 'Error') enqueueSnackbar(text, { variant: 'error' })
        else enqueueSnackbar(text, { variant: 'info' })
      }
      socket.on(event, handler)
      handlers.push(handler)
    })

    return () => {
      TASK_EVENTS.forEach((event, i) => socket.off(event, handlers[i]))
    }
  }, [enqueueSnackbar])

  return null
}

function ScrollbarStyles() {
  const theme = useTheme()
  const isMobile = useMediaQuery(theme.breakpoints.down('sm'))
  const isDark = theme.palette.mode === 'dark'

  return (
    <GlobalStyles
      styles={{
        html: { scrollbarWidth: isMobile ? 'initial' : 'thin' },
        body: { overflowY: isMobile ? 'initial' : 'hidden' },
        ...(isMobile ? {} : {
          '*::-webkit-scrollbar': { width: 8, height: 8 },
          '*::-webkit-scrollbar-button': { width: 0, height: 0 },
          '*::-webkit-scrollbar-corner': { background: '0 0' },
          '*::-webkit-scrollbar-thumb': { borderRadius: 4, backgroundColor: 'transparent' },
          '*::-webkit-scrollbar-track': { borderRadius: 4 },
          '*::-webkit-scrollbar-track:hover': {
            backgroundColor: isDark ? grey[800] : grey[200],
          },
          '*::-webkit-scrollbar-thumb:hover': {
            backgroundColor: `${theme.palette.primary.main}!important`,
          },
          '*:hover::-webkit-scrollbar-thumb': {
            backgroundColor: isDark ? grey[600] : grey[400],
          },
          '.notistack-MuiContent': { borderRadius: '12px' },
        }),
      }}
    />
  )
}

export default function App() {
  const [mode, setMode] = useState<ThemeMode>(getThemeMode)

  const toggleMode = () => {
    setMode((prev) => {
      const next: ThemeMode = prev === 'dark' ? 'light' : 'dark'
      setThemeMode(next)
      return next
    })
  }

  const theme = buildTheme(getAccentColor(), mode)

  return (
    <ThemeModeContext.Provider value={{ mode, toggleMode }}>
      <ThemeProvider theme={theme}>
        <CssBaseline />
        <ScrollbarStyles />
        <SnackbarProvider
          maxSnack={5}
          anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }}
          autoHideDuration={5000}
        >
          <SocketNotifier />
          <RouterProvider router={router} />
        </SnackbarProvider>
      </ThemeProvider>
    </ThemeModeContext.Provider>
  )
}
