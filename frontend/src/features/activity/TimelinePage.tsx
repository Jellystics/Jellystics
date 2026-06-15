import { useState, useEffect } from 'react'
import {
  Alert, Box, Card, CardContent, Typography, Chip, Avatar,
  Skeleton, TextField, InputAdornment,
} from '@mui/material'
import { format, parseISO } from 'date-fns'
import { Search24Regular } from '@fluentui/react-icons'
import { useTranslation } from 'react-i18next'
import PageHeader from '@/shared/components/PageHeader/PageHeader'
import api from '@/lib/axios'
import type { TimelineEntry } from '@/shared/types/activity'
import { useDebounce } from '@/shared/hooks/useDebounce'
import { getDateLocale } from '@/lib/dateLocale'

function formatDuration(seconds: number, t: (k: string) => string): string {
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (h > 0) return `${h}${t('time.hourShort')} ${m}${t('time.minuteShort')}`
  return `${m}${t('time.minuteShort')}`
}

export default function TimelinePage() {
  const { t } = useTranslation()
  const [entries, setEntries] = useState<TimelineEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [search, setSearch] = useState('')
  const debouncedSearch = useDebounce(search)

  useEffect(() => {
    const load = (showLoading = true) => {
      if (showLoading) setLoading(true)
      api
        .get('/stats/getActivityTimeline')
        .then((r) => setEntries(r.data ?? []))
        .catch(() => setError(t('common.loadError')))
        .finally(() => setLoading(false))
    }

    load()
    const interval = window.setInterval(() => load(false), 15000)
    return () => window.clearInterval(interval)
  }, [t])

  const filtered = entries.filter((e) =>
    debouncedSearch === '' ||
    e.UserName.toLowerCase().includes(debouncedSearch.toLowerCase()) ||
    e.ItemName.toLowerCase().includes(debouncedSearch.toLowerCase())
  )

  const grouped = filtered.reduce<Record<string, TimelineEntry[]>>((acc, entry) => {
    const date = format(parseISO(entry.StartTime), 'yyyy-MM-dd')
    if (!acc[date]) acc[date] = []
    acc[date].push(entry)
    return acc
  }, {})

  return (
    <>
      <PageHeader title={t('nav.timeline')} />
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      <TextField
        size="small"
        placeholder={t('common.search')}
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        slotProps={{
          input: {
            startAdornment: (
              <InputAdornment position="start">
                <Search24Regular style={{ fontSize: 18 }} />
              </InputAdornment>
            ),
          },
        }}
        sx={{ mb: 3, width: 280 }}
      />

      {loading ? (
        Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={i} variant="rectangular" height={80} sx={{ mb: 2, borderRadius: 1 }} />
        ))
      ) : (
        Object.entries(grouped)
          .sort(([a], [b]) => b.localeCompare(a))
          .map(([date, dayEntries]) => (
            <Box key={date} sx={{ mb: 3 }}>
              <Typography variant="overline" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
                {format(parseISO(date), 'EEEE dd MMMM yyyy', { locale: getDateLocale() })}
              </Typography>
              <Box sx={{ pl: 2, borderLeft: '2px solid', borderColor: 'primary.main' }}>
                {dayEntries.map((entry) => (
                  <Card key={entry.Id} sx={{ mb: 1 }}>
                    <CardContent sx={{ py: 1.5, px: 2, '&:last-child': { pb: 1.5 } }}>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
                        <Avatar sx={{ width: 32, height: 32, bgcolor: 'primary.main', fontSize: 12, fontWeight: 700 }}>
                          {entry.UserName.charAt(0).toUpperCase()}
                        </Avatar>
                        <Box sx={{ flex: 1, minWidth: 0 }}>
                          <Typography variant="body2" sx={{ fontWeight: 600 }} noWrap>{entry.ItemName}</Typography>
                          <Typography variant="caption" color="text.secondary">
                            {entry.UserName} · {format(parseISO(entry.StartTime), 'HH:mm')} · {formatDuration(entry.Duration, t)}
                          </Typography>
                        </Box>
                        <Chip label={entry.PlayMethod} size="small" sx={{ fontSize: 11, height: 20 }} />
                      </Box>
                    </CardContent>
                  </Card>
                ))}
              </Box>
            </Box>
          ))
      )}
    </>
  )
}
