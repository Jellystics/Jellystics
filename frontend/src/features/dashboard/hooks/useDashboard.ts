import { useState, useEffect, useCallback } from 'react'
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

interface ActivityPoint {
  date: string
  plays: number
  duration: number
}

interface DashboardData {
  globalStats: GlobalStats | null
  sessions: Session[]
  topItems: TopItem[]
  topUsers: TopUser[]
  activityOverTime: ActivityPoint[]
  loading: boolean
  error: string | null
  refetch: () => void
}

export function useDashboard(): DashboardData {
  const [globalStats, setGlobalStats] = useState<GlobalStats | null>(null)
  const [sessions, setSessions] = useState<Session[]>([])
  const [topItems, setTopItems] = useState<TopItem[]>([])
  const [topUsers, setTopUsers] = useState<TopUser[]>([])
  const [activityOverTime, setActivityOverTime] = useState<ActivityPoint[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetch = useCallback(async (showLoading = true) => {
    if (showLoading) setLoading(true)
    setError(null)
    try {
      const [statsRes, sessionsRes, topItemsRes, topUsersRes, activityRes] = await Promise.all([
        api.get('/stats/getGlobalStats'),
        api.get('/sessions/current'),
        api.get('/stats/getMostPlayedItems?type=all&limit=5'),
        api.get('/stats/getMostActiveUsers?limit=5'),
        api.get('/stats/getWatchStatisticsOverTime?days=7'),
      ])
      setGlobalStats(statsRes.data)
      setSessions(sessionsRes.data ?? [])
      setTopItems(topItemsRes.data ?? [])
      setTopUsers(topUsersRes.data ?? [])
      setActivityOverTime(activityRes.data ?? [])
    } catch {
      setError('Failed to load dashboard data')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetch()
    const interval = window.setInterval(() => fetch(false), 15000)
    return () => window.clearInterval(interval)
  }, [fetch])

  return { globalStats, sessions, topItems, topUsers, activityOverTime, loading, error, refetch: fetch }
}
