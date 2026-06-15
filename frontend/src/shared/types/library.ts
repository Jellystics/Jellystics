export interface Library {
  Id: string
  Name: string
  CollectionType: string
  ItemCount: number
  EpisodeCount?: number
  SeasonCount?: number
}

export interface LibraryItem {
  Id: string
  Name: string
  Type: string
  ProductionYear?: number
  CommunityRating?: number
  Size?: number
  PlayCount: number
  LastPlayed?: string
  SeriesName?: string
  IndexNumber?: number
  ParentIndexNumber?: number
}

export interface LibraryStats {
  Name?: string
  TotalItems: number
  TotalPlayCount: number
  TotalWatchTime: number
  MostPlayedItem?: LibraryItem
}

export interface GenreStat {
  Genre: string
  Count: number
  PlayCount: number
}
