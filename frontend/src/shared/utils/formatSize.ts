import i18next from 'i18next'

export function formatSize(bytes?: number): string | null {
  if (!bytes) return null
  const gb = bytes / 1024 / 1024 / 1024
  if (gb >= 1024) return `${(gb / 1024).toFixed(gb / 1024 >= 10 ? 1 : 2)} ${i18next.t('units.terabytes')}`
  if (gb >= 1) return `${gb.toFixed(gb >= 10 ? 1 : 2)} ${i18next.t('units.gigabytes')}`
  return `${Math.round(bytes / 1024 / 1024)} ${i18next.t('units.megabytes')}`
}
