import { createContext, useContext, useState, useCallback } from 'react'
import type { ReactNode } from 'react'
import { subDays } from 'date-fns'

export interface DateRange {
  from: Date
  to: Date
}

interface DateRangeContextValue extends DateRange {
  setRange: (from: Date, to: Date) => void
  setPreset: (days: number) => void // 0 = all time
}

const DateRangeContext = createContext<DateRangeContextValue | null>(null)

export function DateRangeProvider({ children }: { children: ReactNode }) {
  const [range, setRangeState] = useState<DateRange>({
    from: subDays(new Date(), 30),
    to: new Date(),
  })

  const setRange = useCallback((from: Date, to: Date) => {
    setRangeState({ from, to })
  }, [])

  const setPreset = useCallback((days: number) => {
    if (days === 0) {
      setRangeState({ from: new Date('2000-01-01'), to: new Date() })
    } else {
      setRangeState({ from: subDays(new Date(), days), to: new Date() })
    }
  }, [])

  return (
    <DateRangeContext.Provider value={{ ...range, setRange, setPreset }}>
      {children}
    </DateRangeContext.Provider>
  )
}

export function useDateRange(): DateRangeContextValue {
  const ctx = useContext(DateRangeContext)
  if (!ctx) throw new Error('useDateRange must be used within DateRangeProvider')
  return ctx
}

/** Format a Date as YYYY-MM-DD */
export function fmtDate(d: Date): string {
  return d.toISOString().split('T')[0]
}
