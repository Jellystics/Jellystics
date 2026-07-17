import { useState, useEffect, useCallback, useRef } from 'react'
import { useDateRange, fmtDate } from '@/lib/dateRange'

export function useFetchWithRange<T>(
  fetcher: (from: string, to: string) => Promise<T[]>,
  defaultValue: T[] = [],
) {
  const { from, to } = useDateRange()
  const [data, setData] = useState<T[]>(defaultValue)
  const [loading, setLoading] = useState(true)

  const fetcherRef = useRef(fetcher)
  fetcherRef.current = fetcher

  const fetch = useCallback(async () => {
    setLoading(true)
    try {
      const result = await fetcherRef.current(fmtDate(from), fmtDate(to))
      setData(result ?? defaultValue)
    } catch {
      setData(defaultValue)
    } finally {
      setLoading(false)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [from, to])

  useEffect(() => { fetch() }, [fetch])

  return { data, loading, refetch: fetch }
}
