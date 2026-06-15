export interface WatchStatOverTime {
  date: string
  plays: number
  duration: number
}

export interface HourStat {
  hour: number
  plays: number
  duration: number
}

export interface DayStat {
  day: string
  plays: number
  duration: number
}

export interface PlayMethodStat {
  method: string
  count: number
  duration: number
}

export interface ClientStat {
  client: string
  count: number
  duration: number
}

export interface GlobalStats {
  TotalPlays: number
  TotalWatchTime: number
  ActiveUsers: number
  TotalUsers: number
  TotalLibraries: number
  TotalItems: number
}
