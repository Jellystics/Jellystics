import { useState, useEffect, useRef } from 'react'

export function useFetchWithDays<T>(
  fetcher: (days: number) => Promise<T[]>,
  defaultDays = 30,
) {
  const [days, setDays] = useState(defaultDays)
  const [data, setData] = useState<T[]>([])
  const [loading, setLoading] = useState(true)

  const fetcherRef = useRef(fetcher)
  fetcherRef.current = fetcher

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    fetcherRef.current(days)
      .then((res) => { if (!cancelled) setData(res ?? []) })
      .catch(() => { if (!cancelled) setData([]) })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [days])

  return { data, loading, days, setDays }
}
