import { enUS, fr, es } from 'date-fns/locale'
import i18next from 'i18next'

const locales = { en: enUS, fr, es }

export function getDateLocale() {
  return locales[i18next.language as keyof typeof locales] ?? enUS
}
