import i18next from 'i18next'

export function formatWatchTime(minutes: number | null | undefined): string {
  if (!minutes || minutes <= 0) return `0${i18next.t('time.minuteShort')}`

  const totalMinutes = Math.floor(minutes)
  const hours = Math.floor(totalMinutes / 60)
  const mins = totalMinutes % 60

  const h = i18next.t('time.hourShort')
  const m = i18next.t('time.minuteShort')

  if (hours === 0) return `${mins}${m}`
  return mins > 0 ? `${hours}${h} ${mins}${m}` : `${hours}${h}`
}

export function formatSecondsToWatchTime(seconds: number | null | undefined): string {
  return formatWatchTime(seconds ? Math.floor(seconds / 60) : 0)
}
