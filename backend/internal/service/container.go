package service

import (
	"github.com/Jellystics/Jellystics/internal/service/auth"
	"github.com/Jellystics/Jellystics/internal/service/stats"
	syncsvc "github.com/Jellystics/Jellystics/internal/service/sync"
	"github.com/Jellystics/Jellystics/internal/service/task"
	"github.com/Jellystics/Jellystics/internal/service/webhook"
)

// Container holds all service instances.
type Container struct {
	Auth    *auth.Service
	Sync    *syncsvc.Service
	Stats   *stats.Service
	Task    *task.Service
	Webhook *webhook.Service
}
