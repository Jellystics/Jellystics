const FAVICON_KEY = 'jellystics-favicon'
const DEFAULT_FAVICON = '/logo.svg'

/** Resize an image file to a square PNG data URL (max 128px). */
function resizeImage(file: File, size = 128): Promise<string> {
  return new Promise((resolve, reject) => {
    const img = new Image()
    const url = URL.createObjectURL(file)
    img.onload = () => {
      const canvas = document.createElement('canvas')
      canvas.width = size
      canvas.height = size
      const ctx = canvas.getContext('2d')
      if (!ctx) { reject(new Error('canvas')); return }
      ctx.drawImage(img, 0, 0, size, size)
      URL.revokeObjectURL(url)
      resolve(canvas.toDataURL('image/png'))
    }
    img.onerror = () => { URL.revokeObjectURL(url); reject(new Error('load')) }
    img.src = url
  })
}

function applyFaviconHref(href: string): void {
  let link = document.querySelector<HTMLLinkElement>('link[rel="icon"]')
  if (!link) {
    link = document.createElement('link')
    link.rel = 'icon'
    document.head.appendChild(link)
  }
  link.type = href.startsWith('data:') ? 'image/png' : 'image/svg+xml'
  link.href = href
}

export function getCustomFavicon(): string | null {
  return localStorage.getItem(FAVICON_KEY)
}

export async function uploadFavicon(file: File): Promise<void> {
  const dataUrl = await resizeImage(file)
  localStorage.setItem(FAVICON_KEY, dataUrl)
  applyFaviconHref(dataUrl)
}

export function resetFavicon(): void {
  localStorage.removeItem(FAVICON_KEY)
  applyFaviconHref(DEFAULT_FAVICON)
}

/** Call once on app init to restore any saved custom favicon. */
export function applyStoredFavicon(): void {
  const stored = getCustomFavicon()
  if (stored) applyFaviconHref(stored)
}
