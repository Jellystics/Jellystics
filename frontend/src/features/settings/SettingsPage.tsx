import { Box, Tab, Tabs } from '@mui/material'
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

const TABS = [
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
  const [searchParams, setSearchParams] = useSearchParams()

  const currentView = searchParams.get('view') ?? 'config'
  const tabIndex = TABS.findIndex((tab) => tab.view === currentView)
  const activeTab = tabIndex >= 0 ? tabIndex : 0

  return (
    <>
      <PageHeader title={t('nav.settings')} />

      <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 3 }}>
        <Tabs
          value={activeTab}
          onChange={(_, v) => setSearchParams({ view: TABS[v].view }, { replace: true })}
          variant="scrollable"
          scrollButtons="auto"
        >
          {TABS.map(({ key }) => (
            <Tab key={key} label={t(key)} />
          ))}
        </Tabs>
      </Box>

      <Box
        key={activeTab}
        sx={{
          animation: 'fadeIn 150ms ease-in-out',
          '@keyframes fadeIn': { from: { opacity: 0 }, to: { opacity: 1 } },
        }}
      >
        {activeTab === 0 && <ConfigTab />}
        {activeTab === 1 && <TasksTab />}
        {activeTab === 2 && <BackupTab />}
        {activeTab === 3 && <WebhooksTab />}
        {activeTab === 4 && <SecurityTab />}
        {activeTab === 5 && <LogsTab />}
        {activeTab === 6 && <ApiKeysTab />}
      </Box>
    </>
  )
}
