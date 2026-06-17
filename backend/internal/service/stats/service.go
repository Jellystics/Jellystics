package stats

import (
	"context"

	"github.com/Jellystics/Jellystics/internal/repository"
)

type Service struct {
	repos *repository.Container
}

func New(repos *repository.Container) *Service {
	return &Service{repos: repos}
}

func (s *Service) GlobalStats(ctx context.Context) (*repository.GlobalStats, error) {
	return s.repos.Stats.GetGlobalStats(ctx)
}

func (s *Service) MostViewedLibraries(ctx context.Context, limit int) ([]repository.LibraryPlayStat, error) {
	if limit <= 0 {
		limit = 10
	}
	return s.repos.Stats.GetMostViewedLibraries(ctx, limit)
}

func (s *Service) LibraryStats(ctx context.Context, libraryId string) (*repository.LibraryStats, error) {
	return s.repos.Stats.GetLibraryStats(ctx, libraryId)
}

func (s *Service) ActivityOverTime(ctx context.Context, days int) ([]repository.DailyActivity, error) {
	if days <= 0 {
		days = 90
	}
	return s.repos.Stats.GetActivityOverTime(ctx, days)
}

func (s *Service) TopUsers(ctx context.Context, limit int) ([]repository.UserStat, error) {
	if limit <= 0 {
		limit = 10
	}
	return s.repos.Stats.GetTopUsers(ctx, limit)
}

func (s *Service) MostPlayedItems(ctx context.Context, libraryId string, limit int) ([]repository.ItemPlayStat, error) {
	if limit <= 0 {
		limit = 10
	}
	return s.repos.Stats.GetMostPlayedItems(ctx, libraryId, limit)
}

func (s *Service) MostPlayedArtists(ctx context.Context, libraryId string, limit int) ([]repository.ArtistPlayStat, error) {
	if limit <= 0 {
		limit = 10
	}
	return s.repos.Stats.GetMostPlayedArtists(ctx, libraryId, limit)
}

func (s *Service) MostPlayedAlbums(ctx context.Context, libraryId, artistId string, limit int) ([]repository.AlbumPlayStat, error) {
	if limit <= 0 {
		limit = 10
	}
	return s.repos.Stats.GetMostPlayedAlbums(ctx, libraryId, artistId, limit)
}

func (s *Service) MostPlayedTracks(ctx context.Context, libraryId, albumId string, limit int) ([]repository.TrackPlayStat, error) {
	if limit <= 0 {
		limit = 10
	}
	return s.repos.Stats.GetMostPlayedTracks(ctx, libraryId, albumId, limit)
}

func (s *Service) UserHistory(ctx context.Context, userId string, page, pageSize int) ([]repository.ActivityEntry, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	return s.repos.Stats.GetUserHistory(ctx, userId, page, pageSize)
}
