export interface JellyfinUser {
  Id: string
  Name: string
  HasPassword: boolean
  LastLoginDate?: string
  LastActivityDate?: string
  PrimaryImageTag?: string
}

export interface UserStats {
  UserId: string
  UserName: string
  TotalPlays: number
  TotalWatchTime: number
  LastSeen?: string
  FavoriteGenre?: string
}

export interface UserActivity {
  date: string
  count: number
  duration: number
}
