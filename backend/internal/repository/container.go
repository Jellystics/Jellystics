package repository

import "gorm.io/gorm"

// Container holds all repository implementations.
type Container struct {
	Config      ConfigRepository
	User        UserRepository
	Library     LibraryRepository
	Item        ItemRepository
	Season      SeasonRepository
	Episode     EpisodeRepository
	MusicArtist MusicArtistRepository
	MusicTrack  MusicTrackRepository
	ItemInfo    ItemInfoRepository
	Playback    PlaybackRepository
	Watchdog    WatchdogRepository
	PluginData  PluginDataRepository
	Log         LogRepository
	Webhook     WebhookRepository
	Stats       StatsRepository
}

// New wires all GORM-backed repository implementations.
func New(db *gorm.DB) *Container {
	return &Container{
		Config:      &configRepo{db},
		User:        &userRepo{db},
		Library:     &libraryRepo{db},
		Item:        &itemRepo{db},
		Season:      &seasonRepo{db},
		Episode:     &episodeRepo{db},
		MusicArtist: &musicArtistRepo{db},
		MusicTrack:  &musicTrackRepo{db},
		ItemInfo:    &itemInfoRepo{db},
		Playback:    &playbackRepo{db},
		Watchdog:    &watchdogRepo{db},
		PluginData:  &pluginDataRepo{db},
		Log:         &logRepo{db},
		Webhook:     &webhookRepo{db},
		Stats:       &statsRepo{db},
	}
}
