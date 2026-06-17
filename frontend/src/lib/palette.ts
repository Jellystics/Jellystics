export interface ColorPalette {
  id: string
  label: string
  light: string  // primary color for light mode
  dark: string   // primary color for dark mode
  builtIn?: boolean
}

const SELECTED_KEY = 'jellystics-palette'
const CUSTOM_KEY = 'jellystics-custom-palettes'

export const DEFAULT_PALETTES: ColorPalette[] = [
  { id: 'slate',   label: 'Slate',   light: '#64748b', dark: '#94a3b8', builtIn: true },
  { id: 'blue',    label: 'Blue',    light: '#2563eb', dark: '#60a5fa', builtIn: true },
  { id: 'emerald', label: 'Emerald', light: '#059669', dark: '#34d399', builtIn: true },
  { id: 'orange',  label: 'Orange',  light: '#ea580c', dark: '#fb923c', builtIn: true },
  { id: 'pink',    label: 'Pink',    light: '#db2777', dark: '#f472b6', builtIn: true },
  { id: 'violet',  label: 'Violet',  light: '#7c3aed', dark: '#a78bfa', builtIn: true },
  { id: 'sky',     label: 'Sky',     light: '#0284c7', dark: '#38bdf8', builtIn: true },
  { id: 'green',   label: 'Green',   light: '#16a34a', dark: '#4ade80', builtIn: true },
]

export function getSelectedPaletteId(): string {
  return localStorage.getItem(SELECTED_KEY) ?? 'slate'
}

export function setSelectedPaletteId(id: string): void {
  localStorage.setItem(SELECTED_KEY, id)
}

export function getCustomPalettes(): ColorPalette[] {
  try {
    return JSON.parse(localStorage.getItem(CUSTOM_KEY) ?? '[]')
  } catch { return [] }
}

export function saveCustomPalettes(palettes: ColorPalette[]): void {
  localStorage.setItem(CUSTOM_KEY, JSON.stringify(palettes))
}

export function getAllPalettes(): ColorPalette[] {
  return [...DEFAULT_PALETTES, ...getCustomPalettes()]
}

export function getPalette(id: string): ColorPalette {
  return getAllPalettes().find((p) => p.id === id) ?? DEFAULT_PALETTES[0]
}
