function parseJwtPayload(token: string): Record<string, unknown> | null {
  try {
    const payload = token.split('.')[1]
    return JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/')))
  } catch {
    return null
  }
}

export function useCurrentUser() {
  const username = localStorage.getItem('jellystics-username') ?? ''
  const token = localStorage.getItem('jellystics-token') ?? ''

  const payload = token ? parseJwtPayload(token) : null
  const userId = (payload?.userId as string | undefined) ?? ''

  const avatarUrl = userId
    ? `/proxy/Users/Images/Primary/?id=${userId}&fillWidth=100&quality=90`
    : ''

  return { username, userId, avatarUrl }
}
