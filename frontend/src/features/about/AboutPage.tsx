import { useState, useEffect } from 'react'
import {
  Box, Card, CardContent, Typography, Chip, Button, Divider,
  CircularProgress, Skeleton,
} from '@mui/material'
import {
  BugFilled,
  ArrowCircleUp24Regular,
  Checkmark24Regular,
} from '@fluentui/react-icons'

function GitHubIcon({ size = 16 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path d="M12 0C5.37 0 0 5.37 0 12c0 5.3 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61-.546-1.387-1.333-1.756-1.333-1.756-1.09-.745.083-.729.083-.729 1.205.084 1.84 1.237 1.84 1.237 1.07 1.834 2.807 1.304 3.492.997.108-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.31.468-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.3 1.23a11.5 11.5 0 0 1 3.003-.404c1.02.005 2.047.138 3.006.404 2.29-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222 0 1.606-.015 2.896-.015 3.286 0 .322.216.694.825.576C20.565 21.795 24 17.295 24 12c0-6.63-5.37-12-12-12z" />
    </svg>
  )
}
import { useTranslation } from 'react-i18next'
import PageHeader from '@/shared/components/PageHeader/PageHeader'
import api from '@/lib/axios'
import { useLogo } from '@/lib/FaviconContext'

const REPO_URL = 'https://github.com/Jellystics/Jellystics'

interface VersionInfo {
  current_version: string
  latest_version: string
  message: string
  update_available: boolean
}

function ExternalLink({ href, icon, label }: { href: string; icon: React.ReactNode; label: string }) {
  return (
    <Button
      variant="outlined"
      size="small"
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      startIcon={icon}
      sx={{ borderRadius: 2, textTransform: 'none' }}
    >
      {label}
    </Button>
  )
}

export default function AboutPage() {
  const { t } = useTranslation()
  const { logoUrl } = useLogo()
  const [info, setInfo] = useState<VersionInfo | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.get('/api/CheckForUpdates')
      .then((r) => setInfo(r.data))
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  const isUpToDate = info && !info.update_available && info.current_version !== '0.0.0'
  const hasUpdate = info?.update_available

  return (
    <>
      <PageHeader title={t('nav.about')} />

      <Box sx={{ maxWidth: 560, display: 'flex', flexDirection: 'column', gap: 2 }}>

        {/* Identity */}
        <Card>
          <CardContent sx={{ display: 'flex', alignItems: 'center', gap: 3, py: 3 }}>
            <Box
              component="img"
              src={logoUrl}
              alt="Jellystics"
              sx={{ width: 64, height: 64, objectFit: 'contain', flexShrink: 0 }}
            />
            <Box>
              <Typography variant="h5" sx={{ fontWeight: 700, letterSpacing: '-0.02em' }}>
                Jellystics
              </Typography>
              <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
                {t('about.description')}
              </Typography>
            </Box>
          </CardContent>
        </Card>

        {/* Version */}
        <Card>
          <CardContent>
            <Typography variant="subtitle2" color="text.secondary" sx={{ mb: 1.5, textTransform: 'uppercase', letterSpacing: '0.08em', fontSize: 11 }}>
              {t('about.version')}
            </Typography>

            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <Typography variant="body2" color="text.secondary">{t('about.installed')}</Typography>
                {loading ? (
                  <Skeleton width={60} height={24} />
                ) : (
                  <Chip
                    label={info?.current_version ? `v${info.current_version}` : '—'}
                    size="small"
                    sx={{ fontFamily: 'monospace', fontWeight: 400 }}
                  />
                )}
              </Box>

              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <Typography variant="body2" color="text.secondary">{t('about.latest')}</Typography>
                {loading ? (
                  <Skeleton width={60} height={24} />
                ) : (
                  <Chip
                    label={info?.latest_version ?? '—'}
                    size="small"
                    color={hasUpdate ? 'warning' : 'default'}
                    sx={{ fontFamily: 'monospace', fontWeight: 400 }}
                  />
                )}
              </Box>

              {!loading && info && (
                <>
                  <Divider />
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    {loading ? (
                      <CircularProgress size={14} />
                    ) : hasUpdate ? (
                      <ArrowCircleUp24Regular style={{ fontSize: 18, color: 'var(--mui-palette-warning-main)' }} />
                    ) : isUpToDate ? (
                      <Checkmark24Regular style={{ fontSize: 18, color: 'var(--mui-palette-success-main)' }} />
                    ) : null}
                    <Typography variant="body2" color={hasUpdate ? 'warning.main' : isUpToDate ? 'success.main' : 'text.secondary'}>
                      {info.message || '—'}
                    </Typography>
                    {hasUpdate && (
                      <Button
                        size="small"
                        variant="contained"
                        color="warning"
                        href={`${REPO_URL}/releases/latest`}
                        target="_blank"
                        rel="noopener noreferrer"
                        sx={{ ml: 'auto', borderRadius: 2, textTransform: 'none' }}
                      >
                        {t('about.update')}
                      </Button>
                    )}
                  </Box>
                </>
              )}
            </Box>
          </CardContent>
        </Card>

        {/* Links */}
        <Card>
          <CardContent>
            <Typography variant="subtitle2" color="text.secondary" sx={{ mb: 1.5, textTransform: 'uppercase', letterSpacing: '0.08em', fontSize: 11 }}>
              {t('about.links')}
            </Typography>
            <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
              <ExternalLink
                href={REPO_URL}
                icon={<GitHubIcon size={16} />}
                label={t('about.github')}
              />
              <ExternalLink
                href={`${REPO_URL}/issues/new`}
                icon={<BugFilled style={{ fontSize: 16 }} />}
                label={t('about.openIssue')}
              />
            </Box>
          </CardContent>
        </Card>

      </Box>
    </>
  )
}
