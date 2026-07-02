package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/jellyfin"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/ws"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Service struct {
	repos *repository.Container
	jf    *jellyfin.Client
	hub   *ws.Hub
}


func New(repos *repository.Container, jf *jellyfin.Client, hub *ws.Hub) *Service {
	return &Service{repos: repos, jf: jf, hub: hub}
}

func (s *Service) log(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	log.Printf("[sync] %s", msg)
	s.hub.Emit("TaskLog", msg)
}

// refreshClient reads JF host/apikey from the DB and updates s.jf.
// Must be called at the start of every top-level sync operation so that
// credentials stored via configSetup (DB) are always used, regardless of
// whether JF_HOST/JF_API_KEY env vars are set.
func (s *Service) refreshClient(ctx context.Context) error {
	cfg, err := s.repos.Config.Get(ctx)
	if err != nil {
		log.Printf("[sync] refreshClient: cannot read app config: %v", err)
		return fmt.Errorf("cannot read app config: %w", err)
	}
	if cfg.JFHost == nil || *cfg.JFHost == "" {
		log.Printf("[sync] refreshClient: Jellyfin host not configured in DB")
		return fmt.Errorf("Jellyfin host not configured")
	}
	apiKey := ""
	if cfg.JFApiKey != nil {
		apiKey = *cfg.JFApiKey
	}
	log.Printf("[sync] refreshClient: using host=%s", *cfg.JFHost)
	s.jf = jellyfin.NewClient(*cfg.JFHost, apiKey)
	return nil
}

const (
	ticksPerSecond     = 10_000_000 // Jellyfin PositionTicks are 100-nanosecond intervals
	minPlaybackSeconds = 30         // ignore sessions shorter than 30 s
	maxTickDelta       = 15         // cap per-tick increment to avoid jumps after missed ticks
)

// SessionTick is called every ~10 s. It:
//  1. Fetches live Jellyfin sessions and broadcasts them via WebSocket.
//  2. Upserts active sessions to jf_activity_watchdog, accumulating real watch time.
//  3. Detects sessions that have ended and promotes them to jf_playback_activity.
func (s *Service) SessionTick(ctx context.Context) {
	if err := s.refreshClient(ctx); err != nil {
		return
	}

	sessions, err := s.jf.GetSessions(ctx)
	if err != nil {
		return
	}

	// 1. Broadcast to frontend
	s.hub.Emit("sessions", sessions)

	// Load existing watchdog entries so we can accumulate WatchedSeconds.
	existingWatchdog, err := s.repos.Watchdog.List(ctx)
	if err != nil {
		return
	}
	wdMap := make(map[string]models.JFActivityWatchdog, len(existingWatchdog))
	for _, wd := range existingWatchdog {
		wdMap[wd.Id] = wd
	}

	// 2. Build watchdog entries for currently active sessions.
	// Detect media switches (same Jellyfin session, different item) and promote the old item.
	now := time.Now()
	liveIDs := make(map[string]struct{}, len(sessions))
	entries := make([]models.JFActivityWatchdog, 0)
	var switchPromotions []models.JFPlaybackActivity
	for _, sess := range sessions {
		if sess.NowPlayingItem == nil {
			continue
		}
		liveIDs[sess.Id] = struct{}{}

		nowPlayingItemId := sess.NowPlayingItem.Id
		var episodeId *string
		seriesName := sess.NowPlayingItem.SeriesName
		seasonId := sess.NowPlayingItem.SeasonId

		if sess.NowPlayingItem.SeriesId != nil && *sess.NowPlayingItem.SeriesId != "" {
			// TV show: parent = series, child = episode
			nowPlayingItemId = *sess.NowPlayingItem.SeriesId
			episodeId = &sess.NowPlayingItem.Id
		} else if sess.NowPlayingItem.AlbumId != nil && *sess.NowPlayingItem.AlbumId != "" {
			// Music track: parent = album, child = track
			nowPlayingItemId = *sess.NowPlayingItem.AlbumId
			episodeId = &sess.NowPlayingItem.Id
			seriesName = sess.NowPlayingItem.Album
			seasonId = nil
		}

		isPaused := false
		if sess.PlayState != nil {
			isPaused = sess.PlayState.IsPaused
		}

		// Calculate accumulated real watch time.
		var watchedSeconds int64
		var activityId string
		nowStr := now.Format("2006-01-02 15:04:05.000-07:00")

		if prev, exists := wdMap[sess.Id]; exists {
			// Detect media switch: same session, different item.
			prevItemKey := ""
			if prev.NowPlayingItemId != nil {
				prevItemKey = *prev.NowPlayingItemId
			}
			if prev.EpisodeId != nil {
				prevItemKey += "|" + *prev.EpisodeId
			}
			newItemKey := nowPlayingItemId
			if episodeId != nil {
				newItemKey += "|" + *episodeId
			}

			if prevItemKey != "" && prevItemKey != newItemKey {
				// Item changed — promote the old watchdog entry before starting fresh.
				oldDuration := prev.WatchedSeconds
				if prev.LastTickAt != nil && (prev.IsPaused == nil || !*prev.IsPaused) {
					delta := int64(now.Sub(*prev.LastTickAt).Seconds())
					if delta > maxTickDelta {
						delta = maxTickDelta
					}
					if delta > 0 {
						oldDuration += delta
					}
				}
				if oldDuration >= minPlaybackSeconds {
					oldActId := prev.Id
					if prev.ActivityId != nil {
						oldActId = *prev.ActivityId
					}
					switchPromotions = append(switchPromotions, models.JFPlaybackActivity{
						Id:                   oldActId,
						IsPaused:             prev.IsPaused,
						UserId:               prev.UserId,
						UserName:             prev.UserName,
						Client:               prev.Client,
						DeviceName:           prev.DeviceName,
						DeviceId:             prev.DeviceId,
						ApplicationVersion:   prev.ApplicationVersion,
						NowPlayingItemId:     prev.NowPlayingItemId,
						NowPlayingItemName:   prev.NowPlayingItemName,
						EpisodeId:            prev.EpisodeId,
						SeasonId:             prev.SeasonId,
						SeriesName:           prev.SeriesName,
						PlaybackDuration:     &oldDuration,
						PlayMethod:           prev.PlayMethod,
						ActivityDateInserted: prev.ActivityDateInserted,
						MediaStreams:          prev.MediaStreams,
						TranscodingInfo:      prev.TranscodingInfo,
						PlayState:            prev.PlayState,
						OriginalContainer:    prev.OriginalContainer,
						RemoteEndPoint:       prev.RemoteEndPoint,
						ServerId:             prev.ServerId,
						Source:               "watchdog",
					})
				}
				// Reset counters for the new item.
				watchedSeconds = 0
				activityId = uuid.New().String()
			} else {
				// Same item — keep accumulating.
				watchedSeconds = prev.WatchedSeconds
				if prev.LastTickAt != nil && !isPaused {
					delta := int64(now.Sub(*prev.LastTickAt).Seconds())
					if delta > maxTickDelta {
						delta = maxTickDelta
					}
					if delta > 0 {
						watchedSeconds += delta
					}
				}
				// Preserve the original ActivityId and start time.
				if prev.ActivityId != nil {
					activityId = *prev.ActivityId
				} else {
					activityId = uuid.New().String()
				}
				// Preserve the original start time.
				if prev.ActivityDateInserted != nil {
					nowStr = *prev.ActivityDateInserted
				}
			}
		} else {
			// Brand new session.
			activityId = uuid.New().String()
		}

		entry := models.JFActivityWatchdog{
			Id:                   sess.Id,
			ActivityId:           &activityId,
			UserId:               &sess.UserId,
			UserName:             &sess.UserName,
			Client:               &sess.Client,
			DeviceName:           &sess.DeviceName,
			DeviceId:             &sess.DeviceId,
			ApplicationVersion:   &sess.ApplicationVersion,
			NowPlayingItemId:     &nowPlayingItemId,
			NowPlayingItemName:   &sess.NowPlayingItem.Name,
			EpisodeId:            episodeId,
			SeasonId:             seasonId,
			SeriesName:           seriesName,
			ActivityDateInserted: &nowStr,
			RemoteEndPoint:       sess.RemoteEndPoint,
			ServerId:             sess.ServerId,
			WatchedSeconds:       watchedSeconds,
			LastTickAt:           &now,
		}
		if sess.PlayState != nil {
			entry.IsPaused = &isPaused
			entry.PlayMethod = sess.PlayState.PlayMethod
			entry.PlaybackDuration = sess.PlayState.PositionTicks
		}
		entries = append(entries, entry)
	}
	// Promote items from media switches.
	if len(switchPromotions) > 0 {
		if err := s.repos.Playback.Upsert(ctx, switchPromotions); err == nil {
			log.Printf("[monitor] promoted %d media-switch session(s) to jf_playback_activity", len(switchPromotions))
		}
	}
	if len(entries) > 0 {
		_ = s.repos.Watchdog.Upsert(ctx, entries)
	}

	// 3. Promote finished sessions (in watchdog but no longer live)
	type promotion struct {
		sessionID string
		activity  models.JFPlaybackActivity
	}
	var promotions []promotion
	for _, wd := range existingWatchdog {
		if _, alive := liveIDs[wd.Id]; alive {
			continue
		}
		// Use accumulated real watch time instead of PositionTicks.
		durationSecs := wd.WatchedSeconds
		if durationSecs < minPlaybackSeconds {
			_ = s.repos.Watchdog.Delete(ctx, wd.Id)
			continue
		}
		actId := wd.Id
		if wd.ActivityId != nil {
			actId = *wd.ActivityId
		}
		promotions = append(promotions, promotion{
			sessionID: wd.Id,
			activity: models.JFPlaybackActivity{
				Id:                   actId,
				IsPaused:             wd.IsPaused,
				UserId:               wd.UserId,
				UserName:             wd.UserName,
				Client:               wd.Client,
				DeviceName:           wd.DeviceName,
				DeviceId:             wd.DeviceId,
				ApplicationVersion:   wd.ApplicationVersion,
				NowPlayingItemId:     wd.NowPlayingItemId,
				NowPlayingItemName:   wd.NowPlayingItemName,
				EpisodeId:            wd.EpisodeId,
				SeasonId:             wd.SeasonId,
				SeriesName:           wd.SeriesName,
				PlaybackDuration:     &durationSecs,
				PlayMethod:           wd.PlayMethod,
				ActivityDateInserted: wd.ActivityDateInserted,
				MediaStreams:         wd.MediaStreams,
				TranscodingInfo:      wd.TranscodingInfo,
				PlayState:            wd.PlayState,
				OriginalContainer:    wd.OriginalContainer,
				RemoteEndPoint:       wd.RemoteEndPoint,
				ServerId:             wd.ServerId,
				Source:               "watchdog",
			},
		})
	}
	if len(promotions) > 0 {
		acts := make([]models.JFPlaybackActivity, len(promotions))
		for i, p := range promotions {
			acts[i] = p.activity
		}
		if err := s.repos.Playback.Upsert(ctx, acts); err == nil {
			for _, p := range promotions {
				_ = s.repos.Watchdog.Delete(ctx, p.sessionID)
			}
			log.Printf("[monitor] promoted %d finished session(s) to jf_playback_activity", len(promotions))
		}
	}
}

// BroadcastSessions fetches live Jellyfin sessions and emits them to all WS clients.
// Kept for backward-compatibility with callers; production uses SessionTick.
func (s *Service) BroadcastSessions(ctx context.Context) {
	if err := s.refreshClient(ctx); err != nil {
		return
	}
	sessions, err := s.jf.GetSessions(ctx)
	if err != nil {
		return
	}
	s.hub.Emit("sessions", sessions)
}

// FullSync runs a complete sync of all libraries, items, episodes, and music.
func (s *Service) FullSync(ctx context.Context) error {
	if err := s.refreshClient(ctx); err != nil {
		return err
	}
	s.log("[0%%] Starting full Jellyfin sync...")
	start := time.Now()

	s.log("[5%%] Syncing users...")
	if err := s.SyncUsers(ctx); err != nil {
		s.log("ERROR syncing users: %s", err.Error())
		return fmt.Errorf("sync users: %w", err)
	}
	s.log("[10%%] Users synced.")

	s.log("[12%%] Syncing libraries...")
	libs, err := s.SyncLibraries(ctx)
	if err != nil {
		s.log("ERROR syncing libraries: %s", err.Error())
		return fmt.Errorf("sync libraries: %w", err)
	}
	s.log("[15%%] Found %d libraries.", len(libs))

	n := len(libs)
	for i, lib := range libs {
		libId := lib.Id
		collType := ""
		if lib.CollectionType != nil {
			collType = *lib.CollectionType
		}

		pctStart := 15 + i*75/max(n, 1)
		pctEnd := 15 + (i+1)*75/max(n, 1)
		s.log("[%d%%] Syncing library: %s (%s)...", pctStart, deref(lib.Name), collType)

		var syncErr error
		switch collType {
		case "tvshows":
			syncErr = s.SyncTVLibrary(ctx, libId)
		case "movies":
			syncErr = s.SyncMovieLibrary(ctx, libId)
		case "music":
			syncErr = s.SyncMusicLibrary(ctx, libId)
		default:
			syncErr = s.SyncGenericLibrary(ctx, libId)
		}

		if syncErr != nil {
			s.log("[%d%%] ERROR syncing library %s: %s", pctEnd, deref(lib.Name), syncErr.Error())
			// Continue with other libraries
		} else {
			s.log("[%d%%] Library %s synced.", pctEnd, deref(lib.Name))
		}
	}

	// Refresh materialized views
	s.log("[92%%] Refreshing statistics views...")
	if err := s.repos.Stats.RefreshViews(ctx); err != nil {
		s.log("WARNING: view refresh failed: %s", err.Error())
	}

	s.log("[100%%] Full sync complete in %dms.", time.Since(start).Milliseconds())
	return nil
}


// SyncUsers syncs all Jellyfin users.
func (s *Service) SyncUsers(ctx context.Context) error {
	users, err := s.jf.GetUsers(ctx)
	if err != nil {
		return err
	}
	mapped := make([]models.JFUser, len(users))
	for i, u := range users {
		mapped[i] = models.JFUser{
			Id:               u.Id,
			Name:             &u.Name,
			PrimaryImageTag:  u.PrimaryImageTag,
			LastLoginDate:    u.LastLoginDate,
			LastActivityDate: u.LastActivityDate,
			IsAdministrator:  u.Policy.IsAdministrator,
		}
	}
	return s.repos.User.Upsert(ctx, mapped)
}

// SyncLibraries fetches libraries from Jellyfin, upserts them, archives removed ones.
// Returns the list of active libraries for further item sync.
func (s *Service) SyncLibraries(ctx context.Context) ([]models.JFLibrary, error) {
	cfg, err := s.repos.Config.Get(ctx)
	if err != nil || cfg.JFHost == nil {
		return nil, fmt.Errorf("jellyfin host not configured")
	}

	// Use the first admin user to fetch libraries
	users, err := s.jf.GetUsers(ctx)
	if err != nil {
		return nil, err
	}
	var adminId string
	for _, u := range users {
		if u.Policy.IsAdministrator {
			adminId = u.Id
			break
		}
	}
	if adminId == "" && len(users) > 0 {
		adminId = users[0].Id
	}

	libs, err := s.jf.GetLibraries(ctx, adminId)
	if err != nil {
		return nil, err
	}

	mapped := make([]models.JFLibrary, len(libs))
	ids := make([]string, len(libs))
	for i, l := range libs {
		id := l.ItemId
		if id == "" {
			id = l.Id
		}
		primary := l.ImageTags["Primary"]
		mapped[i] = models.JFLibrary{
			Id:               id,
			Name:             &l.Name,
			ServerId:         &l.ServerId,
			IsFolder:         &l.IsFolder,
			Type:             &l.Type,
			CollectionType:   &l.CollectionType,
			ImageTagsPrimary: strPtr(primary),
		}
		ids[i] = id
	}

	if err := s.repos.Library.Upsert(ctx, mapped); err != nil {
		return nil, err
	}
	if err := s.repos.Library.ArchiveNotIn(ctx, ids); err != nil {
		return nil, err
	}
	return mapped, nil
}

// SyncMovieLibrary syncs Movie items for a library.
func (s *Service) SyncMovieLibrary(ctx context.Context, libraryId string) error {
	items, err := s.jf.GetAllItems(ctx, libraryId, []string{"Movie"}, jellyfin.StandardFields)
	if err != nil {
		return err
	}

	mapped := make([]models.JFLibraryItem, 0, len(items))
	infos := make([]models.JFItemInfo, 0, len(items))
	ids := make([]string, 0, len(items))

	for _, item := range items {
		mapped = append(mapped, mapLibraryItem(item, libraryId))
		ids = append(ids, item.Id)
		if len(item.MediaSources) > 0 {
			infos = append(infos, mapItemInfo(item, item.MediaSources[0], "Movie"))
		}
	}

	if err := s.repos.Item.Upsert(ctx, mapped); err != nil {
		return err
	}
	if len(ids) > 0 {
		if err := s.repos.Item.ArchiveNotIn(ctx, libraryId, ids); err != nil {
			return err
		}
	}
	if err := s.repos.ItemInfo.Upsert(ctx, infos); err != nil {
		return err
	}
	return s.repos.ItemInfo.RemoveOrphaned(ctx)
}

// SyncTVLibrary syncs Series + Seasons + Episodes for a library.
func (s *Service) SyncTVLibrary(ctx context.Context, libraryId string) error {
	// Sync series
	series, err := s.jf.GetAllItems(ctx, libraryId, []string{"Series"}, jellyfin.StandardFields)
	if err != nil {
		return err
	}

	seriesMapped := make([]models.JFLibraryItem, 0, len(series))
	seriesIds := make([]string, 0, len(series))
	for _, item := range series {
		seriesMapped = append(seriesMapped, mapLibraryItem(item, libraryId))
		seriesIds = append(seriesIds, item.Id)
	}
	if err := s.repos.Item.Upsert(ctx, seriesMapped); err != nil {
		return err
	}
	if len(seriesIds) > 0 {
		if err := s.repos.Item.ArchiveNotIn(ctx, libraryId, seriesIds); err != nil {
			return err
		}
	}

	// Sync seasons
	seasons, err := s.jf.GetAllItems(ctx, libraryId, []string{"Season"}, jellyfin.StandardFields)
	if err != nil {
		return err
	}
	seasonMapped := make([]models.JFLibrarySeason, 0, len(seasons))
	for _, item := range seasons {
		backdrop := item.Backdrop()
		seasonMapped = append(seasonMapped, models.JFLibrarySeason{
			Id:                      item.Id,
			Name:                    &item.Name,
			ServerId:                &item.ServerId,
			IndexNumber:             item.IndexNumber,
			Type:                    &item.Type,
			ParentLogoItemId:        item.ParentLogoItemId,
			ParentBackdropItemId:    item.ParentBackdropItemId,
			ParentBackdropImageTags: backdrop,
			SeriesName:              item.SeriesName,
			SeriesId:                item.SeriesId,
			SeriesPrimaryImageTag:   item.SeriesPrimaryImageTag,
		})
	}
	if err := s.repos.Season.Upsert(ctx, seasonMapped); err != nil {
		return err
	}

	// Sync episodes
	episodes, err := s.jf.GetAllItems(ctx, libraryId, []string{"Episode"}, jellyfin.StandardFields)
	if err != nil {
		return err
	}
	epMapped := make([]models.JFLibraryEpisode, 0, len(episodes))
	epInfos := make([]models.JFItemInfo, 0)
	for _, item := range episodes {
		ep, info := mapEpisode(item)
		epMapped = append(epMapped, ep)
		if info != nil {
			epInfos = append(epInfos, *info)
		}
	}
	if err := s.repos.Episode.Upsert(ctx, epMapped); err != nil {
		return err
	}
	return s.repos.ItemInfo.Upsert(ctx, epInfos)
}

// SyncMusicLibrary syncs MusicArtist + MusicAlbum + Audio items.
func (s *Service) SyncMusicLibrary(ctx context.Context, libraryId string) error {
	// Artists
	artists, err := s.jf.GetAllItems(ctx, libraryId, []string{"MusicArtist"}, jellyfin.StandardFields)
	if err != nil {
		return err
	}
	artistMapped := make([]models.JFMusicArtist, 0, len(artists))
	artistIds := make([]string, 0, len(artists))
	for _, a := range artists {
		genres := genresJSON(a.Genres)
		primary := a.ImageTags["Primary"]
		artistMapped = append(artistMapped, models.JFMusicArtist{
			Id:               a.Id,
			LibraryId:        &libraryId,
			Name:             &a.Name,
			Overview:         a.Overview,
			ImageTagsPrimary: strPtr(primary),
			Genres:           genres,
		})
		artistIds = append(artistIds, a.Id)
	}
	if err := s.repos.MusicArtist.Upsert(ctx, artistMapped); err != nil {
		return err
	}
	if len(artistIds) > 0 {
		if err := s.repos.MusicArtist.ArchiveNotIn(ctx, libraryId, artistIds); err != nil {
			return err
		}
	}

	// Albums (MusicAlbum → jf_library_items)
	albums, err := s.jf.GetAllItems(ctx, libraryId, []string{"MusicAlbum"}, jellyfin.StandardFields)
	if err != nil {
		return err
	}
	albumMapped := make([]models.JFLibraryItem, 0, len(albums))
	albumIds := make([]string, 0, len(albums))
	for _, a := range albums {
		albumMapped = append(albumMapped, mapLibraryItem(a, libraryId))
		albumIds = append(albumIds, a.Id)
	}
	if err := s.repos.Item.Upsert(ctx, albumMapped); err != nil {
		return err
	}
	if len(albumIds) > 0 {
		if err := s.repos.Item.ArchiveNotIn(ctx, libraryId, albumIds); err != nil {
			return err
		}
	}

	// Tracks (Audio → jf_music_tracks + jf_item_info for file size)
	tracks, err := s.jf.GetAllItems(ctx, libraryId, []string{"Audio"}, jellyfin.StandardFields)
	if err != nil {
		return err
	}
	trackMapped := make([]models.JFMusicTrack, 0, len(tracks))
	trackInfos := make([]models.JFItemInfo, 0, len(tracks))
	trackIds := make([]string, 0, len(tracks))
	for _, t := range tracks {
		genres := genresJSON(t.Genres)
		primary := t.ImageTags["Primary"]
		artistId := t.FirstAlbumArtistId()
		track := models.JFMusicTrack{
			Id:               t.Id,
			LibraryId:        &libraryId,
			AlbumId:          t.AlbumId,
			ArtistId:         artistId,
			Name:             &t.Name,
			AlbumName:        t.Album,
			AlbumArtist:      t.AlbumArtist,
			IndexNumber:      t.IndexNumber,
			DiscNumber:       t.ParentIndexNumber, // ParentIndexNumber = disc number for tracks
			RunTimeTicks:     t.RunTimeTicks,
			DateCreated:      t.DateCreated,
			ProductionYear:   t.ProductionYear,
			ImageTagsPrimary: strPtr(primary),
			Genres:           genres,
		}
		trackMapped = append(trackMapped, track)
		trackIds = append(trackIds, t.Id)
		if len(t.MediaSources) > 0 {
			trackInfos = append(trackInfos, mapItemInfo(t, t.MediaSources[0], "Audio"))
		}
	}
	if err := s.repos.MusicTrack.Upsert(ctx, trackMapped); err != nil {
		return err
	}
	if len(trackIds) > 0 {
		if err := s.repos.MusicTrack.ArchiveNotIn(ctx, libraryId, trackIds); err != nil {
			return err
		}
	}
	if err := s.repos.ItemInfo.Upsert(ctx, trackInfos); err != nil {
		return err
	}
	return s.repos.ItemInfo.RemoveOrphaned(ctx)
}

// SyncGenericLibrary syncs a library with no special handling.
func (s *Service) SyncGenericLibrary(ctx context.Context, libraryId string) error {
	items, err := s.jf.GetAllItems(ctx, libraryId, nil, jellyfin.StandardFields)
	if err != nil {
		return err
	}
	mapped := make([]models.JFLibraryItem, 0, len(items))
	infos := make([]models.JFItemInfo, 0, len(items))
	ids := make([]string, 0, len(items))
	for _, item := range items {
		mapped = append(mapped, mapLibraryItem(item, libraryId))
		ids = append(ids, item.Id)
		if len(item.MediaSources) > 0 {
			infos = append(infos, mapItemInfo(item, item.MediaSources[0], item.Type))
		}
	}
	if err := s.repos.Item.Upsert(ctx, mapped); err != nil {
		return err
	}
	if len(ids) > 0 {
		if err := s.repos.Item.ArchiveNotIn(ctx, libraryId, ids); err != nil {
			return err
		}
	}
	return s.repos.ItemInfo.Upsert(ctx, infos)
}

// SyncSessions fetches active sessions and updates the watchdog table.
// This is called manually from tasks; it preserves existing WatchedSeconds
// and sets LastTickAt so the next SessionTick can accumulate properly.
func (s *Service) SyncSessions(ctx context.Context) error {
	if err := s.refreshClient(ctx); err != nil {
		return err
	}
	sessions, err := s.jf.GetSessions(ctx)
	if err != nil {
		return err
	}

	// Load existing watchdog to preserve WatchedSeconds.
	existingWatchdog, err := s.repos.Watchdog.List(ctx)
	if err != nil {
		return err
	}
	wdMap := make(map[string]models.JFActivityWatchdog, len(existingWatchdog))
	for _, wd := range existingWatchdog {
		wdMap[wd.Id] = wd
	}

	now := time.Now()
	entries := make([]models.JFActivityWatchdog, 0)
	for _, sess := range sessions {
		if sess.NowPlayingItem == nil {
			continue
		}

		nowStr := now.Format("2006-01-02 15:04:05.000-07:00")

		nowPlayingItemId := sess.NowPlayingItem.Id
		var episodeId *string
		seriesName := sess.NowPlayingItem.SeriesName
		seasonId := sess.NowPlayingItem.SeasonId

		if sess.NowPlayingItem.SeriesId != nil && *sess.NowPlayingItem.SeriesId != "" {
			nowPlayingItemId = *sess.NowPlayingItem.SeriesId
			episodeId = &sess.NowPlayingItem.Id
		} else if sess.NowPlayingItem.AlbumId != nil && *sess.NowPlayingItem.AlbumId != "" {
			nowPlayingItemId = *sess.NowPlayingItem.AlbumId
			episodeId = &sess.NowPlayingItem.Id
			seriesName = sess.NowPlayingItem.Album
			seasonId = nil
		}

		isPaused := false
		if sess.PlayState != nil {
			isPaused = sess.PlayState.IsPaused
		}

		// Preserve accumulated watch time and detect media switches.
		var watchedSeconds int64
		var activityId string

		if prev, exists := wdMap[sess.Id]; exists {
			// Detect media switch.
			prevItemKey := ""
			if prev.NowPlayingItemId != nil {
				prevItemKey = *prev.NowPlayingItemId
			}
			if prev.EpisodeId != nil {
				prevItemKey += "|" + *prev.EpisodeId
			}
			newItemKey := nowPlayingItemId
			if episodeId != nil {
				newItemKey += "|" + *episodeId
			}

			if prevItemKey != "" && prevItemKey != newItemKey {
				// Item changed — promote old entry, reset counters.
				oldDuration := prev.WatchedSeconds
				if prev.LastTickAt != nil && (prev.IsPaused == nil || !*prev.IsPaused) {
					delta := int64(now.Sub(*prev.LastTickAt).Seconds())
					if delta > maxTickDelta {
						delta = maxTickDelta
					}
					if delta > 0 {
						oldDuration += delta
					}
				}
				if oldDuration >= minPlaybackSeconds {
					oldActId := prev.Id
					if prev.ActivityId != nil {
						oldActId = *prev.ActivityId
					}
					_ = s.repos.Playback.Upsert(ctx, []models.JFPlaybackActivity{{
						Id:                   oldActId,
						IsPaused:             prev.IsPaused,
						UserId:               prev.UserId,
						UserName:             prev.UserName,
						Client:               prev.Client,
						DeviceName:           prev.DeviceName,
						DeviceId:             prev.DeviceId,
						ApplicationVersion:   prev.ApplicationVersion,
						NowPlayingItemId:     prev.NowPlayingItemId,
						NowPlayingItemName:   prev.NowPlayingItemName,
						EpisodeId:            prev.EpisodeId,
						SeasonId:             prev.SeasonId,
						SeriesName:           prev.SeriesName,
						PlaybackDuration:     &oldDuration,
						PlayMethod:           prev.PlayMethod,
						ActivityDateInserted: prev.ActivityDateInserted,
						MediaStreams:          prev.MediaStreams,
						TranscodingInfo:      prev.TranscodingInfo,
						PlayState:            prev.PlayState,
						OriginalContainer:    prev.OriginalContainer,
						RemoteEndPoint:       prev.RemoteEndPoint,
						ServerId:             prev.ServerId,
						Source:               "watchdog",
					}})
				}
				watchedSeconds = 0
				activityId = uuid.New().String()
			} else {
				// Same item — keep accumulating.
				watchedSeconds = prev.WatchedSeconds
				if prev.LastTickAt != nil && !isPaused {
					delta := int64(now.Sub(*prev.LastTickAt).Seconds())
					if delta > maxTickDelta {
						delta = maxTickDelta
					}
					if delta > 0 {
						watchedSeconds += delta
					}
				}
				if prev.ActivityId != nil {
					activityId = *prev.ActivityId
				} else {
					activityId = uuid.New().String()
				}
				// Preserve original start time.
				if prev.ActivityDateInserted != nil {
					nowStr = *prev.ActivityDateInserted
				}
			}
		} else {
			activityId = uuid.New().String()
		}

		entry := models.JFActivityWatchdog{
			Id:                   sess.Id,
			ActivityId:           &activityId,
			UserId:               &sess.UserId,
			UserName:             &sess.UserName,
			Client:               &sess.Client,
			DeviceName:           &sess.DeviceName,
			DeviceId:             &sess.DeviceId,
			ApplicationVersion:   &sess.ApplicationVersion,
			NowPlayingItemId:     &nowPlayingItemId,
			NowPlayingItemName:   &sess.NowPlayingItem.Name,
			EpisodeId:            episodeId,
			SeasonId:             seasonId,
			SeriesName:           seriesName,
			ActivityDateInserted: &nowStr,
			RemoteEndPoint:       sess.RemoteEndPoint,
			ServerId:             sess.ServerId,
			WatchedSeconds:       watchedSeconds,
			LastTickAt:           &now,
		}

		if sess.PlayState != nil {
			entry.IsPaused = &isPaused
			entry.PlayMethod = sess.PlayState.PlayMethod
		}

		if sess.PlayState != nil && sess.PlayState.PositionTicks != nil {
			entry.PlaybackDuration = sess.PlayState.PositionTicks
		}

		if len(sess.NowPlayingItem.MediaStreams) > 0 {
			if b, err := json.Marshal(sess.NowPlayingItem.MediaStreams); err == nil {
				entry.MediaStreams = datatypes.JSON(b)
			}
		}

		if sess.TranscodingInfo != nil {
			if b, err := json.Marshal(sess.TranscodingInfo); err == nil {
				entry.TranscodingInfo = datatypes.JSON(b)
			}
		}

		if sess.PlayState != nil {
			if b, err := json.Marshal(sess.PlayState); err == nil {
				entry.PlayState = datatypes.JSON(b)
			}
		}

		entry.OriginalContainer = sess.NowPlayingItem.Container

		entries = append(entries, entry)
	}
	return s.repos.Watchdog.Upsert(ctx, entries)
}

// SyncRecentlyAdded syncs only recently added items (partial sync equivalent).
// It fetches the latest items per library for an admin user and upserts them.
func (s *Service) SyncRecentlyAdded(ctx context.Context) error {
	if err := s.refreshClient(ctx); err != nil {
		return err
	}
	s.log("Starting recently-added sync...")

	if err := s.SyncUsers(ctx); err != nil {
		s.log("ERROR syncing users: %s", err.Error())
		return fmt.Errorf("sync users: %w", err)
	}

	libs, err := s.SyncLibraries(ctx)
	if err != nil {
		s.log("ERROR syncing libraries: %s", err.Error())
		return fmt.Errorf("sync libraries: %w", err)
	}

	// Find an admin user for the /Users/{id}/Items/Latest endpoint
	users, err := s.jf.GetUsers(ctx)
	if err != nil {
		return fmt.Errorf("get users: %w", err)
	}
	var adminId string
	for _, u := range users {
		if u.Policy.IsAdministrator {
			adminId = u.Id
			break
		}
	}
	if adminId == "" && len(users) > 0 {
		adminId = users[0].Id
	}
	if adminId == "" {
		return fmt.Errorf("no users found")
	}

	const recentLimit = 100

	for _, lib := range libs {
		libId := lib.Id
		collType := ""
		if lib.CollectionType != nil {
			collType = *lib.CollectionType
		}

		s.log("Syncing recently added for library: %s (%s)...", deref(lib.Name), collType)

		items, err := s.jf.GetRecentlyAdded(ctx, adminId, libId, recentLimit, jellyfin.StandardFields)
		if err != nil {
			s.log("WARNING: failed to get recently added for library %s: %s", deref(lib.Name), err.Error())
			continue
		}

		movies := make([]models.JFLibraryItem, 0)
		series := make([]models.JFLibraryItem, 0)
		albums := make([]models.JFLibraryItem, 0)
		episodes := make([]models.JFLibraryEpisode, 0)
		seasons := make([]models.JFLibrarySeason, 0)
		infos := make([]models.JFItemInfo, 0)

		for _, item := range items {
			switch item.Type {
			case "Movie":
				movies = append(movies, mapLibraryItem(item, libId))
				if len(item.MediaSources) > 0 {
					ms := item.MediaSources[0]
					infos = append(infos, mapItemInfo(item, ms, "Movie"))
				}
			case "Series":
				series = append(series, mapLibraryItem(item, libId))
			case "MusicAlbum":
				albums = append(albums, mapLibraryItem(item, libId))
			case "Episode":
				ep, info := mapEpisode(item)
				episodes = append(episodes, ep)
				if info != nil {
					infos = append(infos, *info)
				}
			case "Season":
				seasons = append(seasons, mapSeason(item))
			}
		}

		allItems := append(movies, append(series, albums...)...)
		if len(allItems) > 0 {
			if err := s.repos.Item.Upsert(ctx, allItems); err != nil {
				s.log("WARNING: upsert items for library %s: %s", deref(lib.Name), err.Error())
			}
		}
		if len(seasons) > 0 {
			if err := s.repos.Season.Upsert(ctx, seasons); err != nil {
				s.log("WARNING: upsert seasons for library %s: %s", deref(lib.Name), err.Error())
			}
		}
		if len(episodes) > 0 {
			if err := s.repos.Episode.Upsert(ctx, episodes); err != nil {
				s.log("WARNING: upsert episodes for library %s: %s", deref(lib.Name), err.Error())
			}
		}
		if len(infos) > 0 {
			if err := s.repos.ItemInfo.Upsert(ctx, infos); err != nil {
				s.log("WARNING: upsert item info for library %s: %s", deref(lib.Name), err.Error())
			}
		}
	}

	// Refresh materialized views
	if err := s.repos.Stats.RefreshViews(ctx); err != nil {
		s.log("WARNING: view refresh failed: %s", err.Error())
	}

	s.log("Recently-added sync complete.")
	return nil
}

// FetchItem fetches a single Jellyfin item by ID and upserts it into the appropriate table.
// It returns the number of records updated (always 1 on success).
func (s *Service) FetchItem(ctx context.Context, itemId string) (int, error) {
	if err := s.refreshClient(ctx); err != nil {
		return 0, err
	}
	item, err := s.jf.GetItem(ctx, itemId, jellyfin.StandardFields)
	if err != nil {
		return 0, fmt.Errorf("fetch item %s: %w", itemId, err)
	}

	switch item.Type {
	case "Episode":
		ep, info := mapEpisode(*item)
		if err := s.repos.Episode.Upsert(ctx, []models.JFLibraryEpisode{ep}); err != nil {
			return 0, err
		}
		if info != nil {
			if err := s.repos.ItemInfo.Upsert(ctx, []models.JFItemInfo{*info}); err != nil {
				return 0, err
			}
		}
	case "Season":
		season := mapSeason(*item)
		if err := s.repos.Season.Upsert(ctx, []models.JFLibrarySeason{season}); err != nil {
			return 0, err
		}
	default:
		// Determine parent ID: use ParentId from the item itself if available.
		parentId := ""
		if item.ParentId != nil {
			parentId = *item.ParentId
		}
		mapped := mapLibraryItem(*item, parentId)
		if err := s.repos.Item.Upsert(ctx, []models.JFLibraryItem{mapped}); err != nil {
			return 0, err
		}
		if len(item.MediaSources) > 0 {
			info := mapItemInfo(*item, item.MediaSources[0], item.Type)
			if err := s.repos.ItemInfo.Upsert(ctx, []models.JFItemInfo{info}); err != nil {
				return 0, err
			}
		}
	}
	return 1, nil
}

// SyncPlaybackPlugin fetches data from the PlaybackReporting plugin and imports it
// into jf_playback_reporting_plugin_data, then merges to jf_playback_activity.
func (s *Service) SyncPlaybackPlugin(ctx context.Context) error {
	if err := s.refreshClient(ctx); err != nil {
		return err
	}
	s.log("Starting PlaybackReporting plugin sync...")

	// Check if plugin is installed
	plugins, err := s.jf.GetInstalledPlugins(ctx)
	if err != nil {
		return fmt.Errorf("get installed plugins: %w", err)
	}

	hasPlugin := false
	for _, p := range plugins {
		if p.ConfigurationFileName != nil &&
			(*p.ConfigurationFileName == "playback_reporting.xml" ||
				*p.ConfigurationFileName == "Jellyfin.Plugin.PlaybackReporting.xml") {
			hasPlugin = true
			break
		}
	}

	if !hasPlugin {
		s.log("PlaybackReporting plugin not detected. Skipping.")
		return nil
	}

	s.log("PlaybackReporting plugin detected. Fetching data...")

	// Find the highest rowid already staged
	rows, err := s.repos.PluginData.GetMaxRowId(ctx)
	if err != nil {
		return fmt.Errorf("get max rowid: %w", err)
	}

	query := "SELECT rowid, * FROM PlaybackActivity"
	if rows > 0 {
		query += fmt.Sprintf(" WHERE rowid > %d", rows)
	}
	query += " ORDER BY rowid"

	s.log("Plugin query: %s", query)

	results, err := s.jf.SubmitCustomQuery(ctx, query)
	if err != nil {
		return fmt.Errorf("submit custom query: %w", err)
	}

	s.log("Plugin returned %d rows.", len(results))

	if len(results) > 0 {
		mapped := make([]models.JFPluginData, 0, len(results))
		for _, row := range results {
			pd := mapPluginRow(row)
			if pd != nil {
				mapped = append(mapped, *pd)
			}
		}
		if len(mapped) > 0 {
			if err := s.repos.PluginData.Upsert(ctx, mapped); err != nil {
				return fmt.Errorf("upsert plugin data: %w", err)
			}
			s.log("Inserted %d plugin rows.", len(mapped))
		}
	}

	// Merge plugin data into jf_playback_activity
	if err := s.repos.PluginData.MergeIntoPlaybackActivity(ctx); err != nil {
		return fmt.Errorf("merge into playback activity: %w", err)
	}

	// Refresh stats views
	if err := s.repos.Stats.RefreshViews(ctx); err != nil {
		s.log("WARNING: view refresh failed: %s", err.Error())
	}

	s.log("PlaybackReporting plugin sync complete.")
	return nil
}

// --- helpers ---

func mapLibraryItem(item jellyfin.Item, parentId string) models.JFLibraryItem {
	primary := item.ImageTags["Primary"]
	banner := item.ImageTags["Banner"]
	logo := item.ImageTags["Logo"]
	thumb := item.ImageTags["Thumb"]
	genres := genresJSON(item.Genres)

	return models.JFLibraryItem{
		Id:                item.Id,
		Name:              &item.Name,
		ServerId:          &item.ServerId,
		PremiereDate:      item.PremiereDate,
		DateCreated:       item.DateCreated,
		EndDate:           item.EndDate,
		CommunityRating:   item.CommunityRating,
		RunTimeTicks:      item.RunTimeTicks,
		ProductionYear:    item.ProductionYear,
		IsFolder:          &item.IsFolder,
		Type:              &item.Type,
		Status:            item.Status,
		ImageTagsPrimary:  strPtr(primary),
		ImageTagsBanner:   strPtr(banner),
		ImageTagsLogo:     strPtr(logo),
		ImageTagsThumb:    strPtr(thumb),
		BackdropImageTags: item.Backdrop(),
		ParentId:          &parentId,
		PrimaryImageHash:  item.PrimaryHash(),
		Genres:            genres,
		AlbumArtist:       item.AlbumArtist,
		ArtistId:          item.FirstAlbumArtistId(),
	}
}

func genresJSON(genres []string) datatypes.JSON {
	normalized := make([]string, len(genres))
	for i, g := range genres {
		normalized[i] = titleCase(g)
	}
	b, _ := json.Marshal(normalized)
	return datatypes.JSON(b)
}

func titleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return strings.Join(words, " ")
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// mapEpisode converts a jellyfin.Item of type Episode into a JFLibraryEpisode and optional JFItemInfo.
func mapEpisode(item jellyfin.Item) (models.JFLibraryEpisode, *models.JFItemInfo) {
	backdrop := item.Backdrop()
	primary := item.PrimaryHash()
	ep := models.JFLibraryEpisode{
		Id:                      item.Id,
		Name:                    &item.Name,
		ServerId:                &item.ServerId,
		PremiereDate:            item.PremiereDate,
		DateCreated:             item.DateCreated,
		OfficialRating:          item.OfficialRating,
		CommunityRating:         item.CommunityRating,
		RunTimeTicks:            item.RunTimeTicks,
		ProductionYear:          item.ProductionYear,
		IndexNumber:             item.IndexNumber,
		ParentIndexNumber:       item.ParentIndexNumber,
		Type:                    &item.Type,
		ParentLogoItemId:        item.ParentLogoItemId,
		ParentBackdropItemId:    item.ParentBackdropItemId,
		ParentBackdropImageTags: backdrop,
		SeriesId:                item.SeriesId,
		SeasonId:                item.SeasonId,
		SeasonName:              item.SeasonName,
		SeriesName:              item.SeriesName,
		PrimaryImageHash:        primary,
	}
	var info *models.JFItemInfo
	if len(item.MediaSources) > 0 {
		ms := item.MediaSources[0]
		i := mapItemInfo(item, ms, "Episode")
		info = &i
	}
	return ep, info
}

// mapSeason converts a jellyfin.Item of type Season into a JFLibrarySeason.
func mapSeason(item jellyfin.Item) models.JFLibrarySeason {
	backdrop := item.Backdrop()
	return models.JFLibrarySeason{
		Id:                      item.Id,
		Name:                    &item.Name,
		ServerId:                &item.ServerId,
		IndexNumber:             item.IndexNumber,
		Type:                    &item.Type,
		ParentLogoItemId:        item.ParentLogoItemId,
		ParentBackdropItemId:    item.ParentBackdropItemId,
		ParentBackdropImageTags: backdrop,
		SeriesName:              item.SeriesName,
		SeriesId:                item.SeriesId,
		SeriesPrimaryImageTag:   item.SeriesPrimaryImageTag,
	}
}

// mapItemInfo creates a JFItemInfo from a MediaSource plus the parent item.
func mapItemInfo(item jellyfin.Item, ms jellyfin.MediaSource, typeOverride string) models.JFItemInfo {
	var streamsJSON datatypes.JSON
	if len(ms.MediaStreams) > 0 {
		if b, err := json.Marshal(ms.MediaStreams); err == nil {
			streamsJSON = datatypes.JSON(b)
		}
	} else if len(item.MediaStreams) > 0 {
		if b, err := json.Marshal(item.MediaStreams); err == nil {
			streamsJSON = datatypes.JSON(b)
		}
	}
	t := typeOverride
	if t == "" {
		t = item.Type
	}
	return models.JFItemInfo{
		Id:          item.Id,
		Path:        strPtr(ms.Path),
		Name:        &item.Name,
		Size:        ms.Size,
		Bitrate:     ms.Bitrate,
		MediaStreams: streamsJSON,
		Type:        &t,
	}
}

// mapPluginRow converts a raw PlaybackReporting plugin result row into a JFPluginData.
// Row order: [0]=rowid [1]=DateCreated [2]=UserId [3]=ItemId [4]=ItemType
//            [5]=ItemName [6]=PlaybackMethod [7]=ClientName [8]=DeviceName [9]=PlayDuration
func mapPluginRow(row jellyfin.PlaybackReportingRow) *models.JFPluginData {
	if len(row) < 10 {
		return nil
	}

	getString := func(v interface{}) *string {
		if v == nil {
			return nil
		}
		s := fmt.Sprintf("%v", v)
		if s == "<nil>" || s == "" {
			return nil
		}
		return &s
	}

	getInt64 := func(v interface{}) *int64 {
		if v == nil {
			return nil
		}
		switch val := v.(type) {
		case float64:
			i := int64(val)
			return &i
		case int64:
			return &val
		case int:
			i := int64(val)
			return &i
		}
		return nil
	}

	rowId := ""
	if row[0] != nil {
		rowId = fmt.Sprintf("%v", row[0])
	}
	if rowId == "" {
		return nil
	}

	var duration int64
	if d := getInt64(row[9]); d != nil && *d >= 0 {
		duration = *d
	}

	return &models.JFPluginData{
		RowId:          rowId,
		DateCreated:    getString(row[1]),
		UserId:         getString(row[2]),
		ItemId:         getString(row[3]),
		ItemType:       getString(row[4]),
		ItemName:       getString(row[5]),
		PlaybackMethod: getString(row[6]),
		ClientName:     getString(row[7]),
		DeviceName:     getString(row[8]),
		PlayDuration:   &duration,
	}
}
