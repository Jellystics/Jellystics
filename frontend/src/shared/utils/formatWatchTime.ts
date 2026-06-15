export function formatWatchTime(minutes: number | null | undefined): string {
  if (!minutes || minutes <= 0) return '0m'

  const totalMinutes = Math.floor(minutes)
  const hours = Math.floor(totalMinutes / 60)
  const mins = totalMinutes % 60

  if (hours === 0) return `${mins}m`
  return mins > 0 ? `${hours}h ${mins}m` : `${hours}h`
}
