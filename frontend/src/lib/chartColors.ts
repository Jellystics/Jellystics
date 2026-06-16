import { useTheme } from '@mui/material/styles'

/**
 * Vivid palette for dark backgrounds — Tailwind *-400 range.
 * Deeper palette for light backgrounds — Tailwind *-600 range (better contrast on white).
 * Order is intentional: blue first, then emerald, orange, pink, violet, yellow, sky, green.
 */
const DARK: string[] = [
  '#60a5fa', // blue-400
  '#34d399', // emerald-400
  '#fb923c', // orange-400
  '#f472b6', // pink-400
  '#a78bfa', // violet-400
  '#facc15', // yellow-400
  '#38bdf8', // sky-400
  '#4ade80', // green-400
]

const LIGHT: string[] = [
  '#2563eb', // blue-600
  '#059669', // emerald-600
  '#ea580c', // orange-600
  '#db2777', // pink-600
  '#7c3aed', // violet-600
  '#ca8a04', // yellow-600
  '#0284c7', // sky-600
  '#16a34a', // green-600
]

/**
 * Returns an ordered list of chart series colors adapted to the current theme mode.
 * Use index 0 for single-series charts, and iterate for multi-series.
 */
export function useChartColors(): string[] {
  const { palette } = useTheme()
  return palette.mode === 'dark' ? DARK : LIGHT
}
