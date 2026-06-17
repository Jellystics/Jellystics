import { createContext, useContext } from 'react'

export type ThemeMode = 'dark' | 'light'

interface ThemeModeContextValue {
  mode: ThemeMode
  toggleMode: () => void
}

export const ThemeModeContext = createContext<ThemeModeContextValue>({
  mode: 'dark',
  toggleMode: () => {},
})

export const useThemeMode = () => useContext(ThemeModeContext)
