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

export interface ItemWatchUser {
  UserId: string
  UserName: string
  PlayCount: number
  TotalWatchTime: number
  LastWatched?: string | null
  IsActive: boolean
}

export interface ItemWatchHistory {
  Id: string
  UserId: string
  UserName: string
  Client?: string | null
  DeviceName?: string | null
  PlayMethod?: string | null
  PlaybackDuration: number
  ActivityDateInserted: string
  RemoteEndPoint?: string | null
  IsActive: boolean
}

export interface ItemDetails {
  item: LibraryItem & {
    Genres?: string[]
    PremiereDate?: string
    DateCreated?: string
    RunTimeTicks?: number
    ParentId?: string
    Path?: string
    Bitrate?: number
  }
  stats: {
    TotalPlays: number
    TotalWatchTime: number
    UniqueUsers: number
    LastWatched?: string | null
    IsActive: boolean
  }
  users: ItemWatchUser[]
  history: ItemWatchHistory[]
}
