import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import api from '@/lib/axios'
import type { GlobalStats } from '@/shared/types/stats'
import type { Session } from '@/shared/types/activity'

interface TopItem {
  Id: string
  Name: string
  PlayCount: number
  Type: string
}

interface TopUser {
  UserId: string
  UserName: string
  TotalPlays: number
  TotalWatchTime: number
}

interface ViewsByLibraryType {
  Audio: number
  Movie: number
  Series: number
  Other: number
}

interface LibraryViewCount {
  Name: string
  Count: number
}

export interface HourStat {
  hour: number
  plays: number
  duration: number
}

export interface DayStat {
  day: number
  plays: number
  duration: number
}

export interface PlaybackMethod {
  method: string
  count: number
  duration: number
}

export interface ClientStat {
  client: string
  count: number
  duration: number
}

interface DashboardData {
  globalStats: GlobalStats | null
  sessions: Session[]
  topItems: TopItem[]
  topUsers: TopUser[]
  viewsByLibraryType: ViewsByLibraryType | null
  mostViewedLibraries: LibraryViewCount[]
  hourlyStats: HourStat[]
  weeklyStats: DayStat[]
  playbackMethods: PlaybackMethod[]
  topClients: ClientStat[]
  loading: boolean
  error: string | null
  refetch: () => void
}

export function useDashboard(days = 30): DashboardData {
  const { t } = useTranslation()
  const [globalStats, setGlobalStats] = useState<GlobalStats | null>(null)
  const [sessions, setSessions] = useState<Session[]>([])
  const [topItems, setTopItems] = useState<TopItem[]>([])
  const [topUsers, setTopUsers] = useState<TopUser[]>([])
  const [viewsByLibraryType, setViewsByLibraryType] = useState<ViewsByLibraryType | null>(null)
  const [mostViewedLibraries, setMostViewedLibraries] = useState<LibraryViewCount[]>([])
  const [hourlyStats, setHourlyStats] = useState<HourStat[]>([])
  const [weeklyStats, setWeeklyStats] = useState<DayStat[]>([])
  const [playbackMethods, setPlaybackMethods] = useState<PlaybackMethod[]>([])
  const [topClients, setTopClients] = useState<ClientStat[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetch = useCallback(async (showLoading = true) => {
    if (showLoading) setLoading(true)
    setError(null)

    const results = await Promise.allSettled([
      api.get('/stats/getGlobalStats'),
      api.get('/sessions/current'),
      api.get(`/stats/getMostPlayedItems?type=all&limit=5&days=${days}`),
      api.get(`/stats/getMostActiveUsers?limit=5&days=${days}`),
      api.get('/stats/getViewsByLibraryType'),
      api.post('/stats/getMostViewedLibraries', { days }),
      api.get(`/stats/getPopularHourOfDay?days=${days}`),
      api.get(`/stats/getPopularDayOfWeek?days=${days}`),
      api.get(`/stats/getMostUsedPlaybackMethod?days=${days}`),
      api.get(`/stats/getMostUsedClients?days=${days}`),
    ])

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const val = (r: PromiseSettledResult<any>) =>
      r.status === 'fulfilled' ? r.value.data : null
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const arr = (r: PromiseSettledResult<any>) => {
      const data = val(r)
      return Array.isArray(data) ? data : []
    }

    const [
      statsRes, sessionsRes, topItemsRes, topUsersRes,
      libTypeRes, mostLibRes, hourRes, dayRes, methodRes, clientRes,
    ] = results

    if (statsRes.status === 'rejected') {
      setError(t('error.loadDashboard'))
    }

    setGlobalStats(val(statsRes))
    setSessions(arr(sessionsRes))
    setTopItems(arr(topItemsRes))
    setTopUsers(arr(topUsersRes))
    setViewsByLibraryType(val(libTypeRes))
    setMostViewedLibraries(arr(mostLibRes).slice(0, 8))
    setHourlyStats(arr(hourRes))
    setWeeklyStats(arr(dayRes))
    setPlaybackMethods(arr(methodRes))
    setTopClients(arr(clientRes))

    setLoading(false)
  }, [t, days])

  useEffect(() => {
    fetch()
    const interval = window.setInterval(() => fetch(false), 15000)
    return () => window.clearInterval(interval)
  }, [fetch])

  return {
    globalStats, sessions, topItems, topUsers,
    viewsByLibraryType, mostViewedLibraries, hourlyStats, weeklyStats,
    playbackMethods, topClients, loading, error, refetch: fetch,
  }
}
