import { Box, Typography, ButtonBase } from '@mui/material'
import { useTheme } from '@mui/material/styles'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'
import PageHeader from '@/shared/components/PageHeader/PageHeader'
import ConfigTab from './components/ConfigTab'
import TasksTab from './components/TasksTab'
import BackupTab from './components/BackupTab'
import WebhooksTab from './components/WebhooksTab'
import SecurityTab from './components/SecurityTab'
import LogsTab from './components/LogsTab'
import ApiKeysTab from './components/ApiKeysTab'

const TABS: { key: string; view: string }[] = [
  { key: 'settings.config',   view: 'config' },
  { key: 'settings.tasks',    view: 'tasks' },
  { key: 'settings.backup',   view: 'backup' },
  { key: 'settings.webhooks', view: 'webhooks' },
  { key: 'settings.security', view: 'security' },
  { key: 'settings.logs',     view: 'logs' },
  { key: 'settings.apiKeys',  view: 'apikeys' },
]

export default function SettingsPage() {
  const { t } = useTranslation()
  const theme = useTheme()
  const [searchParams, setSearchParams] = useSearchParams()

  const currentView = searchParams.get('view') ?? 'config'
  const tab = TABS.findIndex((t) => t.view === currentView).valueOf()
  const activeTab = tab >= 0 ? tab : 0
  const setTab = (index: number) => setSearchParams({ view: TABS[index].view }, { replace: true })

  const activeColor = theme.palette.mode === 'dark'
    ? theme.palette.primary.main
    : theme.palette.primary.main

  return (
    <>
      <PageHeader title={t('nav.settings')} />

      <Box
        sx={{
          display: 'flex',
          gap: 1,
          flexWrap: 'wrap',
          mb: 3,
          pb: 2,
          borderBottom: '1px solid',
          borderColor: 'divider',
        }}
      >
        {TABS.map(({ key }, index) => {
          const isActive = activeTab === index
          return (
            <ButtonBase
              key={key}
              onClick={() => setTab(index)}
              sx={{
                px: 2,
                py: 0.75,
                borderRadius: '90px',
                fontSize: 13,
                fontWeight: isActive ? 600 : 500,
                color: isActive ? '#ffffff' : 'text.secondary',
                bgcolor: isActive ? activeColor : 'transparent',
                transition: 'all 200ms cubic-bezier(0.4,0,0.2,1)',
                '&:hover': {
                  bgcolor: isActive ? activeColor : theme.palette.mode === 'dark' ? 'rgba(255,255,255,0.08)' : 'rgba(0,0,0,0.06)',
                  color: isActive ? '#ffffff' : 'text.primary',
                },
              }}
            >
              <Typography variant="body2" sx={{ fontWeight: 'inherit', fontSize: 'inherit', color: 'inherit' }}>
                {t(key)}
              </Typography>
            </ButtonBase>
          )
        })}
      </Box>

      {activeTab === 0 && <ConfigTab />}
      {activeTab === 1 && <TasksTab />}
      {activeTab === 2 && <BackupTab />}
      {activeTab === 3 && <WebhooksTab />}
      {activeTab === 4 && <SecurityTab />}
      {activeTab === 5 && <LogsTab />}
      {activeTab === 6 && <ApiKeysTab />}
    </>
  )
}
