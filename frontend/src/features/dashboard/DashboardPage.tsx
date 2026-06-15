import { Grid, Alert } from '@mui/material'
import {
  Play24Regular, Clock24Regular, People24Regular,
  VideoClip24Regular, Library24Regular, Apps24Regular,
} from '@fluentui/react-icons'
import { useTranslation } from 'react-i18next'
import { useState } from 'react'
import PageHeader from '@/shared/components/PageHeader/PageHeader'
import StatCard from '@/shared/components/StatCard/StatCard'
import LiveSessions from './components/LiveSessions'
import ActivityChart from './components/ActivityChart'
import TopContent from './components/TopContent'
import TopUsers from './components/TopUsers'
import { useDashboard } from './hooks/useDashboard'
import { formatWatchTime } from '@/shared/utils/formatWatchTime'
import type { ActivityMetric } from '@/shared/components/MetricToggle/MetricToggle'

export default function DashboardPage() {
  const { t } = useTranslation()
  const { globalStats, sessions, topItems, topUsers, activityOverTime, loading, error } = useDashboard()
  const [activityMetric, setActivityMetric] = useState<ActivityMetric>('count')

  return (
    <>
      <PageHeader title={t('nav.dashboard')} />
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid size={{ xs: 6, sm: 4, lg: 2 }}>
          <StatCard label={t('stats.totalPlays')} value={globalStats?.TotalPlays ?? '—'} icon={<Play24Regular />} loading={loading} />
        </Grid>
        <Grid size={{ xs: 6, sm: 4, lg: 2 }}>
          <StatCard label={t('stats.watchTime')} value={globalStats ? formatWatchTime(globalStats.TotalWatchTime) : '—'} icon={<Clock24Regular />} loading={loading} />
        </Grid>
        <Grid size={{ xs: 6, sm: 4, lg: 2 }}>
          <StatCard label={t('stats.activeUsers')} value={globalStats?.ActiveUsers ?? '—'} icon={<People24Regular />} loading={loading} />
        </Grid>
        <Grid size={{ xs: 6, sm: 4, lg: 2 }}>
          <StatCard label={t('stats.totalUsers')} value={globalStats?.TotalUsers ?? '—'} icon={<VideoClip24Regular />} loading={loading} />
        </Grid>
        <Grid size={{ xs: 6, sm: 4, lg: 2 }}>
          <StatCard label={t('stats.libraries')} value={globalStats?.TotalLibraries ?? '—'} icon={<Library24Regular />} loading={loading} />
        </Grid>
        <Grid size={{ xs: 6, sm: 4, lg: 2 }}>
          <StatCard label={t('stats.totalItems')} value={globalStats?.TotalItems ?? '—'} icon={<Apps24Regular />} loading={loading} />
        </Grid>
      </Grid>

      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid size={{ xs: 12, md: 8 }}>
          <ActivityChart data={activityOverTime} loading={loading} metric={activityMetric} onMetricChange={setActivityMetric} />
        </Grid>
        <Grid size={{ xs: 12, md: 4 }}>
          <LiveSessions initialSessions={sessions} loading={loading} />
        </Grid>
      </Grid>

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, md: 6 }}>
          <TopContent items={topItems} loading={loading} />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <TopUsers users={topUsers} loading={loading} />
        </Grid>
      </Grid>
    </>
  )
}
