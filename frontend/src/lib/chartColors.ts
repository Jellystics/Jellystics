import { useTheme } from '@mui/material/styles'
import { usePalette } from './PaletteContext'

// ── HSL helpers ──────────────────────────────────────────────────────────────

function hexToHsl(hex: string): [number, number, number] {
  const r = parseInt(hex.slice(1, 3), 16) / 255
  const g = parseInt(hex.slice(3, 5), 16) / 255
  const b = parseInt(hex.slice(5, 7), 16) / 255

  const max = Math.max(r, g, b)
  const min = Math.min(r, g, b)
  const l = (max + min) / 2

  if (max === min) return [0, 0, Math.round(l * 100)]

  const d = max - min
  const s = l > 0.5 ? d / (2 - max - min) : d / (max + min)

  let h: number
  switch (max) {
    case r:  h = ((g - b) / d + (g < b ? 6 : 0)) / 6; break
    case g:  h = ((b - r) / d + 2) / 6; break
    default: h = ((r - g) / d + 4) / 6
  }

  return [Math.round(h * 360), Math.round(s * 100), Math.round(l * 100)]
}

function hslToHex(h: number, s: number, l: number): string {
  const hNorm = h / 360
  const sNorm = s / 100
  const lNorm = l / 100

  const hue2rgb = (p: number, q: number, t: number) => {
    let tt = t
    if (tt < 0) tt += 1
    if (tt > 1) tt -= 1
    if (tt < 1 / 6) return p + (q - p) * 6 * tt
    if (tt < 1 / 2) return q
    if (tt < 2 / 3) return p + (q - p) * (2 / 3 - tt) * 6
    return p
  }

  let r: number, g: number, b: number
  if (sNorm === 0) {
    r = g = b = lNorm
  } else {
    const q = lNorm < 0.5 ? lNorm * (1 + sNorm) : lNorm + sNorm - lNorm * sNorm
    const p = 2 * lNorm - q
    r = hue2rgb(p, q, hNorm + 1 / 3)
    g = hue2rgb(p, q, hNorm)
    b = hue2rgb(p, q, hNorm - 1 / 3)
  }

  const toHex = (x: number) => Math.round(x * 255).toString(16).padStart(2, '0')
  return `#${toHex(r)}${toHex(g)}${toHex(b)}`
}

// ── Color generation ─────────────────────────────────────────────────────────

/**
 * Generates `count` chart colors distributed evenly around the hue wheel,
 * starting from the palette's primary color. The first color always matches
 * the palette accent; others fan out harmoniously.
 */
function generateChartColors(baseHex: string, isDark: boolean, count = 8): string[] {
  const [h, s] = hexToHsl(baseHex)
  // Ensure enough saturation for vibrancy; keep existing saturation if already rich
  const saturation = Math.max(s, 55)
  // In dark mode we need lighter colors for contrast; in light mode, darker
  const lightness = isDark ? 62 : 44

  return Array.from({ length: count }, (_, i) => {
    const hue = (h + (i * 360) / count) % 360
    return hslToHex(hue, saturation, lightness)
  })
}

// ── Hook ─────────────────────────────────────────────────────────────────────

/**
 * Returns 8 chart colors derived from the active palette's primary hue,
 * adapted to the current theme mode.
 * Index 0 always matches the palette accent color.
 */
export function useChartColors(): string[] {
  const { palette: muiPalette } = useTheme()
  const { palette } = usePalette()
  const isDark = muiPalette.mode === 'dark'
  const baseColor = isDark ? palette.dark : palette.light
  return generateChartColors(baseColor, isDark)
}
