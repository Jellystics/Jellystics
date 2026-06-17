export interface Activity {
  Id: string
  UserId: string
  UserName: string
  ItemId: string
  NowPlayingItemName: string
  SeriesName?: string
  SeasonId?: string
  EpisodeId?: string
  Client: string
  DeviceName: string
  DeviceId: string
  ApplicationVersion: string
  PlayMethod: string
  PlaybackItemId: string
  IsPaused: boolean
  IsActive: boolean
  PlayDuration: number
  ActivityDateInserted: string
  RemoteEndPoint?: string
  NowPlayingItemType?: string
  ParentId?: string
}

export interface Session {
  Id: string
  UserId: string
  UserName: string
  Client: string
  DeviceName: string
  DeviceId: string
  ApplicationVersion: string
  NowPlayingItem?: {
    Id: string
    Name: string
    Type: string
    SeriesName?: string
    SeriesId?: string
    AlbumId?: string
    IndexNumber?: number
    ParentIndexNumber?: number
    RunTimeTicks?: number
    ProductionYear?: number
    Album?: string
    AlbumArtist?: string
  }
  PlayState?: {
    IsPaused: boolean
    PositionTicks?: number
    PlayMethod?: string
  }
  TranscodingInfo?: {
    IsVideoDirect: boolean
    IsAudioDirect: boolean
    Bitrate?: number
    VideoCodec?: string
    AudioCodec?: string
    Width?: number
    Height?: number
    VideoDecoderIsHardware?: boolean
    VideoEncoderIsHardware?: boolean
  }
  RemoteEndPoint?: string
  LastActivityDate: string
}

export interface TimelineEntry {
  Id: string
  UserId: string
  UserName: string
  ItemId: string
  ItemName: string
  StartTime: string
  EndTime: string
  Duration: number
  Client: string
  PlayMethod: string
}
