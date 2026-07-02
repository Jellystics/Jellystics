import { useState, useEffect, useMemo } from 'react'
import {
  Alert, Box, Card, CardContent, Typography, IconButton, Chip, Avatar, Skeleton,
} from '@mui/material'
import { alpha } from '@mui/material/styles'
import {
  format, parseISO, startOfMonth, endOfMonth,
  startOfWeek, endOfWeek, eachDayOfInterval,
  isSameMonth, isToday, isSameDay, addMonths, subMonths,
  getYear, getMonth,
} from 'date-fns'
import { ChevronLeft24Regular, ChevronRight24Regular } from '@fluentui/react-icons'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import PageHeader from '@/shared/components/PageHeader/PageHeader'
import api from '@/lib/axios'
import type { TimelineEntry } from '@/shared/types/activity'
import { getDateLocale } from '@/lib/dateLocale'
import { formatSecondsToWatchTime } from '@/shared/utils/formatWatchTime'
import { useChartColors } from '@/lib/chartColors'

export default function TimelinePage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const CHART_COLORS = useChartColors()
  const [viewMonth, setViewMonth] = useState(() => startOfMonth(new Date()))
  const [entries, setEntries] = useState<TimelineEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedDay, setSelectedDay] = useState<Date | null>(null)

  const isCurrentMonth = isSameMonth(viewMonth, new Date())

  const load = () => {
    setLoading(true)
    setError(null)
    setSelectedDay(null)
    const year = getYear(viewMonth)
    const month = getMonth(viewMonth) + 1
    api.get(`/stats/getActivityTimeline?year=${year}&month=${month}`)
      .then(r => setEntries(r.data ?? []))
      .catch(() => setError(t('common.loadError')))
      .finally(() => setLoading(false))
  }

  useEffect(() => { load() }, [viewMonth, t])

  const byDay = useMemo(() => {
    const map = new Map<string, TimelineEntry[]>()
    for (const e of entries) {
      if (!e.StartTime) continue
      const day = format(parseISO(e.StartTime), 'yyyy-MM-dd')
      if (!map.has(day)) map.set(day, [])
      map.get(day)!.push(e)
    }
    return map
  }, [entries])

  const calDays = useMemo(() => {
    const start = startOfWeek(startOfMonth(viewMonth), { weekStartsOn: 1 })
    const end = endOfWeek(endOfMonth(viewMonth), { weekStartsOn: 1 })
    return eachDayOfInterval({ start, end })
  }, [viewMonth])

const selectedKey = selectedDay ? format(selectedDay, 'yyyy-MM-dd') : null
  const selectedEntries = selectedKey ? (byDay.get(selectedKey) ?? []) : []

  const weekDayLabels = [
    t('days.short.mon'), t('days.short.tue'), t('days.short.wed'), t('days.short.thu'),
    t('days.short.fri'), t('days.short.sat'), t('days.short.sun'),
  ]

  return (
    <>
      <PageHeader title={t('nav.timeline')} onRefresh={() => load()} loading={loading} />
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      <Card sx={{ mb: 3 }}>
        <CardContent>
          {/* Month navigation */}
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
            <IconButton size="small" onClick={() => setViewMonth(m => subMonths(m, 1))}>
              <ChevronLeft24Regular style={{ fontSize: 18 }} />
            </IconButton>
            <Typography variant="h6" sx={{ fontWeight: 600, textTransform: 'capitalize', userSelect: 'none' }}>
              {format(viewMonth, 'MMMM yyyy', { locale: getDateLocale() })}
            </Typography>
            <IconButton size="small" onClick={() => setViewMonth(m => addMonths(m, 1))} disabled={isCurrentMonth}>
              <ChevronRight24Regular style={{ fontSize: 18 }} />
            </IconButton>
          </Box>

          {/* Weekday headers */}
          <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(7, 1fr)', gap: 0.5, mb: 0.5 }}>
            {weekDayLabels.map(d => (
              <Box key={d} sx={{ textAlign: 'center', py: 0.5 }}>
                <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600, fontSize: 11 }}>
                  {d}
                </Typography>
              </Box>
            ))}
          </Box>

          {/* Calendar grid */}
          <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(7, 1fr)', gap: 0.5 }}>
            {loading
              ? Array.from({ length: 35 }).map((_, i) => (
                  <Skeleton key={i} variant="rectangular" sx={{ borderRadius: 1, aspectRatio: '1' }} />
                ))
              : calDays.map(day => {
                  const key = format(day, 'yyyy-MM-dd')
                  const dayEntries = byDay.get(key) ?? []
                  const inMonth = isSameMonth(day, viewMonth)
                  const isSelected = selectedDay ? isSameDay(day, selectedDay) : false
                  const today = isToday(day)
                  return (
                    <Box
                      key={key}
                      onClick={() => inMonth && setSelectedDay(isSelected ? null : day)}
                      sx={{
                        aspectRatio: '1',
                        overflow: 'hidden',
                        borderRadius: 1,
                        p: 0.75,
                        display: 'flex',
                        flexDirection: 'column',
                        border: '1px solid',
                        borderColor: isSelected ? 'primary.main' : today ? 'primary.light' : 'divider',
                        bgcolor: isSelected ? alpha('#888888', 0.10) : 'transparent',
                        opacity: inMonth ? 1 : 0.25,
                        cursor: inMonth ? 'pointer' : 'default',
                        transition: 'border-color 150ms, background-color 150ms',
                        '&:hover': inMonth ? { borderColor: 'primary.main' } : {},
                      }}
                    >
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 0.25 }}>
                        <Typography
                          variant="caption"
                          sx={{ fontSize: 12, fontWeight: today ? 700 : 400, color: today ? 'primary.main' : 'text.primary' }}
                        >
                          {format(day, 'd')}
                        </Typography>
                        {dayEntries.length > 0 && (
                          <Box
                            sx={{
                              minWidth: 18, height: 18, borderRadius: '9px', px: 0.5,
                              bgcolor: CHART_COLORS[0],
                              display: 'flex', alignItems: 'center', justifyContent: 'center',
                            }}
                          >
                            <Typography sx={{ fontSize: 9, fontWeight: 700, color: '#fff', lineHeight: 1 }}>
                              {dayEntries.length}
                            </Typography>
                          </Box>
                        )}
                      </Box>

                      {/* Media names */}
                      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.3, mt: 0.5 }}>
                        {dayEntries.slice(0, 3).map((e, i) => (
                          <Typography
                            key={i}
                            sx={{
                              fontSize: 11, lineHeight: 1.3, fontWeight: 500,
                              overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
                              color: 'text.primary',
                            }}
                          >
                            {e.ItemName}
                          </Typography>
                        ))}
                        {dayEntries.length > 3 && (
                          <Typography sx={{ fontSize: 11, color: 'text.secondary', lineHeight: 1 }}>
                            +{dayEntries.length - 3}
                          </Typography>
                        )}
                      </Box>

                      {/* Poster thumbnails at bottom */}
                      <Box sx={{ display: 'flex', gap: 0.5, mt: 'auto', pt: 0.5, flexWrap: 'wrap' }}>
                        {dayEntries.slice(0, 4).map((e, i) => (
                          <Box
                            key={i}
                            sx={{
                              position: 'relative', width: 24, height: 34,
                              borderRadius: 0.5, overflow: 'hidden', flexShrink: 0,
                              bgcolor: 'rgba(128,128,128,0.15)',
                            }}
                          >
                            <img
                              src={`/proxy/Items/Images/Primary/?id=${encodeURIComponent(e.ItemId)}&fillWidth=48&quality=70`}
                              alt=""
                              style={{ position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover' }}
                              onError={(ev) => { ev.currentTarget.style.display = 'none' }}
                            />
                          </Box>
                        ))}
                        {dayEntries.length > 4 && (
                          <Box
                            sx={{
                              width: 24, height: 34, borderRadius: 0.5, flexShrink: 0,
                              bgcolor: 'rgba(128,128,128,0.2)',
                              display: 'flex', alignItems: 'center', justifyContent: 'center',
                            }}
                          >
                            <Typography sx={{ fontSize: 10, fontWeight: 700, color: 'text.secondary', lineHeight: 1 }}>
                              +{dayEntries.length - 4}
                            </Typography>
                          </Box>
                        )}
                      </Box>
                    </Box>
                  )
                })}
          </Box>
        </CardContent>
      </Card>

      {/* Selected day detail */}
      {selectedDay && (
        <Card>
          <CardContent>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
              <Typography variant="subtitle1" sx={{ fontWeight: 600, textTransform: 'capitalize' }}>
                {format(selectedDay, 'EEEE d MMMM yyyy', { locale: getDateLocale() })}
              </Typography>
              <Chip label={selectedEntries.length} size="small" />
            </Box>

            {selectedEntries.length === 0 ? (
              <Typography variant="body2" color="text.secondary">{t('common.noData')}</Typography>
            ) : (
              selectedEntries.map((e, i) => {
                const dur = formatSecondsToWatchTime(e.Duration)
                const startLabel = e.StartTime
                  ? format(parseISO(e.StartTime), 'HH:mm')
                  : ''

                return (
                  <Box
                    key={i}
                    sx={{
                      display: 'flex', alignItems: 'center', gap: 1.5, py: 1,
                      borderBottom: i < selectedEntries.length - 1 ? '1px solid' : 'none',
                      borderColor: 'divider',
                    }}
                  >
                    {/* Media poster */}
                    <Box
                      onClick={() => navigate(`/items/${e.ItemId}`)}
                      sx={{
                        position: 'relative', width: 36, height: 52,
                        borderRadius: 1, overflow: 'hidden', flexShrink: 0,
                        bgcolor: 'rgba(128,128,128,0.1)',
                        display: 'flex', alignItems: 'center', justifyContent: 'center',
                        cursor: 'pointer',
                      }}
                    >
                      <img
                        src={`/proxy/Items/Images/Primary/?id=${encodeURIComponent(e.ItemId)}&fillWidth=72&quality=85`}
                        alt={e.ItemName}
                        style={{ position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover' }}
                        onError={(ev) => { ev.currentTarget.style.display = 'none' }}
                      />
                    </Box>

                    <Box sx={{ flex: 1, minWidth: 0 }}>
                      <Typography
                        variant="body2"
                        noWrap
                        onClick={() => navigate(`/items/${e.ItemId}`)}
                        sx={{ fontWeight: 500, fontSize: 13, cursor: 'pointer', '&:hover': { textDecoration: 'underline' } }}
                      >
                        {e.ItemName}
                      </Typography>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, mt: 0.25 }}>
                        <Avatar
                          src={`/proxy/Users/Images/Primary/?id=${e.UserId}&fillWidth=32&quality=90`}
                          sx={{ width: 16, height: 16, bgcolor: 'primary.main', fontSize: 8, fontWeight: 700, cursor: 'pointer' }}
                          onClick={() => navigate(`/users/${e.UserId}?view=history`)}
                        >
                          {e.UserName?.charAt(0)?.toUpperCase()}
                        </Avatar>
                        <Typography variant="caption" color="text.secondary" sx={{ fontSize: 11 }}>
                          <Box
                            component="span"
                            onClick={() => navigate(`/users/${e.UserId}?view=history`)}
                            sx={{ cursor: 'pointer', '&:hover': { textDecoration: 'underline' } }}
                          >
                            {e.UserName}
                          </Box>
                          {startLabel && ` · ${startLabel}`}
                          {` · ${dur}`}
                          {e.Client && ` · ${e.Client}`}
                        </Typography>
                      </Box>
                    </Box>

                    {e.PlayMethod && (
                      <Chip label={e.PlayMethod} size="small" variant="outlined" sx={{ fontSize: 10, height: 20, flexShrink: 0 }} />
                    )}
                  </Box>
                )
              })
            )}
          </CardContent>
        </Card>
      )}
    </>
  )
}
