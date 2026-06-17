import { createContext, useContext, useState, type ReactNode } from 'react'
import { getCustomFavicon } from './favicon'

interface FaviconContextValue {
  logoUrl: string
  refreshLogo: () => void
}

const FaviconContext = createContext<FaviconContextValue>({
  logoUrl: '/logo.svg',
  refreshLogo: () => {},
})

export function FaviconProvider({ children }: { children: ReactNode }) {
  const [logoUrl, setLogoUrl] = useState<string>(() => getCustomFavicon() ?? '/logo.svg')

  const refreshLogo = () => setLogoUrl(getCustomFavicon() ?? '/logo.svg')

  return (
    <FaviconContext.Provider value={{ logoUrl, refreshLogo }}>
      {children}
    </FaviconContext.Provider>
  )
}

export function useLogo() {
  return useContext(FaviconContext)
}
