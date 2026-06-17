import { useState, useEffect } from 'react'
import {
  Dialog, DialogTitle, DialogContent, DialogActions,
  Button, Typography, Box, LinearProgress,
} from '@mui/material'
import { ArrowSync24Regular } from '@fluentui/react-icons'
import { useTranslation } from 'react-i18next'
import { useSnackbar } from 'notistack'
import api from '@/lib/axios'

export default function FirstRunDialog() {
  const { t } = useTranslation()
  const { enqueueSnackbar } = useSnackbar()
  const [open, setOpen] = useState(false)
  const [launching, setLaunching] = useState(false)

  useEffect(() => {
    api.get('/api/isFirstRun').then((r) => {
      if (r.data?.firstRun) setOpen(true)
    }).catch(() => {})
  }, [])

  const handleSync = async () => {
    setLaunching(true)
    try {
      await api.post('/api/runTask/Full Jellyfin Sync')
      enqueueSnackbar(t('settings.taskStarted'), { variant: 'info' })
      setOpen(false)
    } catch {
      enqueueSnackbar(t('common.error'), { variant: 'error' })
    } finally {
      setLaunching(false)
    }
  }

  return (
    <Dialog open={open} maxWidth="sm" fullWidth>
      <DialogTitle sx={{ fontWeight: 700 }}>{t('firstRun.title')}</DialogTitle>
      <DialogContent>
        <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 2, mb: 1 }}>
          <ArrowSync24Regular style={{ fontSize: 36, marginTop: 2, flexShrink: 0, opacity: 0.7 }} />
          <Box>
            <Typography variant="body1" sx={{ mb: 1 }}>{t('firstRun.description')}</Typography>
            <Typography variant="body2" color="text.secondary">{t('firstRun.hint')}</Typography>
          </Box>
        </Box>
        {launching && <LinearProgress sx={{ mt: 2, borderRadius: 1 }} />}
      </DialogContent>
      <DialogActions sx={{ px: 3, pb: 2, gap: 1 }}>
        <Button
          variant="text"
          color="inherit"
          onClick={() => setOpen(false)}
          disabled={launching}
        >
          {t('firstRun.dismiss')}
        </Button>
        <Button
          variant="contained"
          startIcon={<ArrowSync24Regular style={{ fontSize: 16 }} />}
          onClick={handleSync}
          disabled={launching}
        >
          {t('firstRun.runSync')}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

import { useState, useEffect } from 'react'
import {
  Dialog, DialogTitle, DialogContent, DialogActions,
  Button, Typography, Box, LinearProgress,
} from '@mui/material'
import { ArrowSync24Regular } from '@fluentui/react-icons'
import { useTranslation } from 'react-i18next'
import { useSnackbar } from 'notistack'
import api from '@/lib/axios'

export default function FirstRunDialog() {
  const { t } = useTranslation()
  const { enqueueSnackbar } = useSnackbar()
  const [open, setOpen] = useState(false)
  const [launching, setLaunching] = useState(false)

  useEffect(() => {
    api.get('/api/isFirstRun').then((r) => {
      if (r.data?.firstRun) setOpen(true)
    }).catch(() => {})
  }, [])

  const handleSync = async () => {
    setLaunching(true)
    try {
      await api.post('/api/runTask/Full Jellyfin Sync')
      enqueueSnackbar(t('settings.taskStarted'), { variant: 'info' })
      setOpen(false)
    } catch {
      enqueueSnackbar(t('common.error'), { variant: 'error' })
    } finally {
      setLaunching(false)
    }
  }

  return (
    <Dialog open={open} maxWidth="sm" fullWidth disableEscapeKeyDown>
      <DialogTitle sx={{ fontWeight: 700 }}>{t('firstRun.title')}</DialogTitle>
      <DialogContent>
        <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 2, mb: 1 }}>
          <ArrowSync24Regular style={{ fontSize: 36, marginTop: 2, flexShrink: 0, opacity: 0.7 }} />
          <Box>
            <Typography variant="body1" sx={{ mb: 1 }}>{t('firstRun.description')}</Typography>
            <Typography variant="body2" color="text.secondary">{t('firstRun.hint')}</Typography>
          </Box>
        </Box>
        {launching && <LinearProgress sx={{ mt: 2, borderRadius: 1 }} />}
      </DialogContent>
      <DialogActions sx={{ px: 3, pb: 2, gap: 1 }}>
        <Button
          variant="text"
          color="inherit"
          onClick={() => setOpen(false)}
          disabled={launching}
        >
          {t('firstRun.dismiss')}
        </Button>
        <Button
          variant="contained"
          startIcon={<ArrowSync24Regular style={{ fontSize: 16 }} />}
          onClick={handleSync}
          disabled={launching}
        >
          {t('firstRun.runSync')}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
