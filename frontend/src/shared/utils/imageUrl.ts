export function getItemImageUrl(id: string, width = 420, quality = 90): string {
  return `/proxy/Items/Images/Primary/?id=${encodeURIComponent(id)}&fillWidth=${width}&quality=${quality}`
}

export function getBackdropImageUrl(id: string, width = 900, quality = 70): string {
  return `/proxy/Items/Images/Backdrop/?id=${encodeURIComponent(id)}&fillWidth=${width}&quality=${quality}`
}

export function getUserImageUrl(id: string, width = 64, quality = 90): string {
  return `/proxy/Users/Images/Primary/?id=${encodeURIComponent(id)}&fillWidth=${width}&quality=${quality}`
}
