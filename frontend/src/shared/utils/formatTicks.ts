export function formatTicks(ticks: number | null | undefined): string {
  if (!ticks) return '—'
  const totalSeconds = Math.floor(ticks / 10_000_000)
  const h = Math.floor(totalSeconds / 3600)
  const m = Math.floor((totalSeconds % 3600) / 60)
  const s = totalSeconds % 60
  if (h > 0) return `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
  return `${m}:${String(s).padStart(2, '0')}`
}

export function formatDuration(ticks: number | null | undefined): string {
  if (!ticks) return '0:00'
  const totalSeconds = Math.floor(ticks / 10_000_000)
  const h = Math.floor(totalSeconds / 3600)
  const m = Math.floor((totalSeconds % 3600) / 60)
  const s = totalSeconds % 60
  if (h > 0) return `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
  return `${m}:${String(s).padStart(2, '0')}`
}

export function ticksToMinutes(ticks: number | null | undefined): number {
  return Math.floor((ticks ?? 0) / 10_000_000 / 60)
}
