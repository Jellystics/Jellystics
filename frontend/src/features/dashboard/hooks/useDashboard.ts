import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import api from '@/lib/axios'
import type { GlobalStats } from '@/shared/types/stats'
import type { Session } from '@/shared/types/activity'

interface ViewsByLibraryType {
  Audio: number
  Movie: number
  Series: number
  Other: number
}

interface DashboardData {
  globalStats: GlobalStats | null
  sessions: Session[]
  viewsByLibraryType: ViewsByLibraryType | null
  loading: boolean
  error: string | null
  refetch: () => void
}

export function useDashboard(): DashboardData {
  const { t } = useTranslation()
  const [globalStats, setGlobalStats] = useState<GlobalStats | null>(null)
  const [sessions, setSessions] = useState<Session[]>([])
  const [viewsByLibraryType, setViewsByLibraryType] = useState<ViewsByLibraryType | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetch = useCallback(async (showLoading = true) => {
    if (showLoading) setLoading(true)
    setError(null)

    const results = await Promise.allSettled([
      api.get('/stats/getGlobalStats'),
      api.get('/sessions/current'),
      api.get('/stats/getViewsByLibraryType'),
    ])

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const val = (r: PromiseSettledResult<any>) =>
      r.status === 'fulfilled' ? r.value.data : null
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const arr = (r: PromiseSettledResult<any>) => {
      const data = val(r)
      return Array.isArray(data) ? data : []
    }

    const [statsRes, sessionsRes, libTypeRes] = results

    if (statsRes.status === 'rejected') {
      setError(t('error.loadDashboard'))
    }

    setGlobalStats(val(statsRes))
    setSessions(arr(sessionsRes))
    setViewsByLibraryType(val(libTypeRes))

    setLoading(false)
  }, [t])

  useEffect(() => {
    fetch()
    const interval = window.setInterval(() => fetch(false), 15000)
    return () => window.clearInterval(interval)
  }, [fetch])

  return { globalStats, sessions, viewsByLibraryType, loading, error, refetch: fetch }
}

// Re-export types used by DashboardPage for per-chart fetching
export interface HourStat { hour: number; plays: number; duration: number }
export interface DayStat { day: number; plays: number; duration: number }
export interface PlaybackMethod { method: string; count: number; duration: number }
export interface ClientStat { client: string; count: number; duration: number }
export interface TopItem { Id: string; Name: string; PlayCount: number; Type: string }
export interface TopUser { UserId: string; UserName: string; TotalPlays: number; TotalWatchTime: number }
export interface LibraryViewCount { Name: string; Count: number }
