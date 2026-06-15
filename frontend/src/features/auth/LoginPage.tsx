import { useEffect, useState } from 'react'
import {
  Box, Button, TextField, Typography,
  CircularProgress, Alert, Paper, Divider,
} from '@mui/material'
import { ArrowLeft24Regular } from '@fluentui/react-icons'
import AuthLogo from './AuthLogo'
import { useTranslation } from 'react-i18next'
import publicApi from '@/lib/publicApi'

type Step = 'server' | 'credentials'

interface ServerInfo {
  serverName: string
  url: string
}

export default function LoginPage() {
  const { t } = useTranslation()
  const [step, setStep] = useState<Step>('server')
  const [serverInfo, setServerInfo] = useState<ServerInfo | null>(null)
  const [jellyfinUrl, setJellyfinUrl] = useState('')
  const [apiKey, setApiKey] = useState('')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [initializing, setInitializing] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // If already configured (state 1), skip to credentials step
  useEffect(() => {
    publicApi.get<{ state: number; jellyfinUrl?: string }>('/auth/isConfigured')
      .then(async (res) => {
        if (res.data.state === 1 && res.data.jellyfinUrl) {
          try {
            const info = await publicApi.post<ServerInfo>('/auth/check-server', { url: res.data.jellyfinUrl })
            setServerInfo(info.data)
          } catch {
            setServerInfo({ serverName: res.data.jellyfinUrl, url: res.data.jellyfinUrl })
          }
          setStep('credentials')
        }
      })
      .catch(() => {})
      .finally(() => setInitializing(false))
  }, [])

  const handleConnect = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!jellyfinUrl.trim() || !apiKey.trim()) return
    setError(null)
    setLoading(true)
    try {
      await publicApi.post('/auth/configSetup', {
        JF_HOST: jellyfinUrl.trim(),
        JF_API_KEY: apiKey.trim(),
      })
      const info = await publicApi.post<ServerInfo>('/auth/check-server', { url: jellyfinUrl.trim() })
      setServerInfo(info.data)
      setStep('credentials')
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { errorMessage?: string } } })?.response?.data?.errorMessage
      setError(msg ?? t('setup.configError'))
    } finally {
      setLoading(false)
    }
  }

  const handleSignIn = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!username.trim() || !password) return
    setError(null)
    setLoading(true)
    try {
      const res = await publicApi.post<{ token: string }>('/auth/jellyfin-login', { username, password })
      localStorage.setItem('jellystics-token', res.data.token)
      localStorage.setItem('jellystics-username', username)
      window.location.href = '/'
    } catch (err: unknown) {
      const status = (err as { response?: { status?: number } })?.response?.status
      if (status === 403) setError(t('auth.adminRequired'))
      else if (status === 503) setError(t('auth.jellyfinUnreachable'))
      else setError(t('auth.invalidCredentials'))
    } finally {
      setLoading(false)
    }
  }

  const goBack = () => {
    setStep('server')
    setError(null)
    setPassword('')
  }

  if (initializing) {
    return (
      <Box sx={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', bgcolor: 'background.default' }}>
        <CircularProgress />
      </Box>
    )
  }

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default', display: 'flex', alignItems: 'center', justifyContent: 'center', p: 2 }}>
      <Box sx={{ width: '100%', maxWidth: 420 }}>
        <Paper
          sx={{
            width: '100%',
            pt: 2, px: 3, pb: 3,
            boxShadow: 'none',
            border: '1px solid',
            borderColor: 'divider',
            borderRadius: 1,
          }}
        >
          <AuthLogo />
          <Divider sx={{ mb: 3 }} />

          {/* Step 1 — server + API key */}
          {step === 'server' && (
            <Box>
              <Typography variant="h6" sx={{ fontWeight: 700 }} gutterBottom>
                {t('setup.title')}
              </Typography>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 2.5 }}>
                {t('setup.description')}
              </Typography>

              {error && (
                <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
                  {error}
                </Alert>
              )}

              <Box component="form" onSubmit={handleConnect} sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                <TextField
                  label={t('setup.jellyfinUrl')}
                  value={jellyfinUrl}
                  onChange={(e) => setJellyfinUrl(e.target.value)}
                  placeholder={t('placeholder.jellyfinUrl')}
                  fullWidth
                  autoFocus
                  disabled={loading}
                  helperText={t('setup.jellyfinUrlHint')}
                />
                <TextField
                  label={t('setup.apiKey')}
                  value={apiKey}
                  onChange={(e) => setApiKey(e.target.value)}
                  fullWidth
                  disabled={loading}
                  helperText={t('setup.apiKeyHint')}
                />
                <Button
                  type="submit"
                  variant="contained"
                  size="large"
                  fullWidth
                  disableElevation
                  disabled={loading || !jellyfinUrl.trim() || !apiKey.trim()}
                  sx={{ mt: 0.5 }}
                >
                  {loading ? <CircularProgress size={22} color="inherit" /> : t('setup.connect')}
                </Button>
              </Box>
            </Box>
          )}

          {/* Step 2 — credentials */}
          {step === 'credentials' && (
            <Box>
              <Typography variant="h6" sx={{ fontWeight: 700 }} gutterBottom>
                {t('auth.signIn')}
              </Typography>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 2.5 }}>
                {t('auth.connectedTo', { name: serverInfo?.serverName ?? '' })}
              </Typography>

              {error && (
                <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
                  {error}
                </Alert>
              )}

              <Box component="form" onSubmit={handleSignIn} sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                <TextField
                  label={t('auth.username')}
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  fullWidth
                  autoFocus
                  autoComplete="username"
                  disabled={loading}
                />
                <TextField
                  label={t('auth.password')}
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  fullWidth
                  autoComplete="current-password"
                  disabled={loading}
                />
                <Button
                  type="submit"
                  variant="contained"
                  size="large"
                  fullWidth
                  disableElevation
                  disabled={loading || !username.trim() || !password}
                  sx={{ mt: 0.5 }}
                >
                  {loading ? <CircularProgress size={22} color="inherit" /> : t('auth.signIn')}
                </Button>
                <Button
                  type="button"
                  fullWidth
                  disabled={loading}
                  startIcon={<ArrowLeft24Regular style={{ fontSize: 18 }} />}
                  onClick={goBack}
                >
                  {t('auth.changeServer')}
                </Button>
              </Box>
            </Box>
          )}
        </Paper>

        <Box sx={{ mt: 2, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 0.75 }}>
          <Typography variant="caption" color="text.disabled" sx={{ fontWeight: 500 }}>
            {t('frame.poweredBy')}
          </Typography>
          <Box
            component="a"
            href="https://github.com/CyferShepard/Jellystat"
            target="_blank"
            rel="noopener noreferrer"
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: 0.5,
              opacity: 0.4,
              transition: 'opacity 0.2s',
              textDecoration: 'none',
              '&:hover': { opacity: 1 },
              '& img': { filter: 'grayscale(100%) brightness(2)', transition: 'filter 0.2s' },
              '&:hover img': { filter: 'grayscale(100%) brightness(2)' },
            }}
          >
            <Box component="img" src="/logo.svg" alt="Jellystics" sx={{ width: 14, height: 14 }} />
            <Typography variant="caption" sx={{ fontWeight: 700, color: 'text.primary' }}>
              Jellystics
            </Typography>
          </Box>
        </Box>
      </Box>
    </Box>
  )
}
