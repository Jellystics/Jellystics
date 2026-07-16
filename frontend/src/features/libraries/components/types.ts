export type HistoryPoint = { date: string; plays: number }
export type ItemWithStats = { Name: string; times_played: number; total_play_time: number }
export type PlayMethodStat = { Key: string; Transcodes: number; DirectPlays: number }
export type LastPlayedRow = { NowPlayingItemId?: string; NowPlayingItemName: string; ActivityDateInserted: string; UserName: string }

export type TimeToWatchData = {
  avgDaysToWatch: number
  medianDaysToWatch: number
  distribution: { bucket: string; count: number }[]
  slowestItems: { id: string; name: string; type: string; daysToWatch: number; dateAdded: string; firstWatched: string }[]
  fastestItems: { id: string; name: string; type: string; daysToWatch: number; dateAdded: string; firstWatched: string }[]
}

export type UnwatchedContentData = {
  summary: { totalItems: number; unwatchedItems: number; unwatchedPercent: number; byType: { type: string; count: number }[] }
  items: { current_page: number; pages: number; size: number; results: { id: string; name: string; type: string; dateAdded: string; genres: string[]; libraryName: string }[] }
}
