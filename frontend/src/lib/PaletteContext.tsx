import { createContext, useContext, useState, type ReactNode } from 'react'
import {
  type ColorPalette,
  getPalette,
  getSelectedPaletteId,
  setSelectedPaletteId,
} from './palette'

interface PaletteContextValue {
  palette: ColorPalette
  paletteId: string
  setPaletteId: (id: string) => void
}

const PaletteContext = createContext<PaletteContextValue>({
  palette: getPalette('slate'),
  paletteId: 'slate',
  setPaletteId: () => {},
})

export function PaletteProvider({ children }: { children: ReactNode }) {
  const [paletteId, setPaletteIdState] = useState<string>(getSelectedPaletteId)

  const setPaletteId = (id: string) => {
    setSelectedPaletteId(id)
    setPaletteIdState(id)
  }

  return (
    <PaletteContext.Provider value={{ palette: getPalette(paletteId), paletteId, setPaletteId }}>
      {children}
    </PaletteContext.Provider>
  )
}

export function usePalette() {
  return useContext(PaletteContext)
}
