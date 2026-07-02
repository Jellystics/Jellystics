/**
 * Date formatting helpers using the browser's Intl.DateTimeFormat.
 * Automatically adapts to user locale: 12h/24h, DD/MM vs MM/DD, etc.
 */

const locale = () => navigator.language || 'en'

export function formatDateTime(value: string | null | undefined): string {
  if (!value) return '—'
  try {
    return new Intl.DateTimeFormat(locale(), {
      day: '2-digit', month: '2-digit', year: 'numeric',
      hour: '2-digit', minute: '2-digit',
    }).format(new Date(value))
  } catch { return value }
}

export function formatDateTimeSeconds(value: string | null | undefined): string {
  if (!value) return '—'
  try {
    return new Intl.DateTimeFormat(locale(), {
      day: '2-digit', month: '2-digit', year: 'numeric',
      hour: '2-digit', minute: '2-digit', second: '2-digit',
    }).format(new Date(value))
  } catch { return value }
}

export function formatDateOnly(value: string | null | undefined): string {
  if (!value) return '—'
  try {
    return new Intl.DateTimeFormat(locale(), {
      day: '2-digit', month: '2-digit', year: 'numeric',
    }).format(new Date(value))
  } catch { return value }
}
