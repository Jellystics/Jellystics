export function useCurrentUser() {
  const username = localStorage.getItem('jellystics-username') ?? ''
  return { username }
}
