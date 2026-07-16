import type { Activity } from '@/shared/types/activity'
import { getItemImageUrl } from './imageUrl'

/**
 * Returns the Jellyfin item ID to use for the activity row's media poster.
 *
 * - Some endpoints alias NowPlayingItemId → ItemId (e.g. getAllUserActivity).
 * - Others return NowPlayingItemId directly (e.g. getLibraryHistory, getUserHistory).
 * - Audio tracks have no artwork on their own ID — ParentId is the album.
 */
export function getActivityImageId(row: Pick<Activity, 'ItemId' | 'NowPlayingItemId' | 'ParentId' | 'NowPlayingItemType'>): string | null {
  const itemId = row.NowPlayingItemId ?? row.ItemId
  if (row.NowPlayingItemType === 'Audio') return itemId ?? null
  return itemId ?? null
}

/**
 * Returns the proxy image URL for an activity row's media poster, or null if no ID is available.
 */
export function getActivityImageUrl(
  row: Pick<Activity, 'ItemId' | 'NowPlayingItemId' | 'ParentId' | 'NowPlayingItemType'>,
  fillWidth = 80,
): string | null {
  const id = getActivityImageId(row)
  if (!id) return null
  return getItemImageUrl(id, fillWidth, 80)
}
