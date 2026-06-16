package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/repository"
)

type Service struct {
	repos *repository.Container
}

func New(repos *repository.Container) *Service {
	return &Service{repos: repos}
}

func (s *Service) List(ctx context.Context) ([]models.Webhook, error) {
	return s.repos.Webhook.List(ctx)
}

func (s *Service) GetByID(ctx context.Context, id int) (*models.Webhook, error) {
	return s.repos.Webhook.GetByID(ctx, id)
}

func (s *Service) Create(ctx context.Context, wh *models.Webhook) error {
	return s.repos.Webhook.Create(ctx, wh)
}

func (s *Service) Update(ctx context.Context, wh *models.Webhook) error {
	return s.repos.Webhook.Update(ctx, wh)
}

func (s *Service) Delete(ctx context.Context, id int) error {
	return s.repos.Webhook.Delete(ctx, id)
}

// Fire sends all enabled webhooks matching the given trigger/event.
func (s *Service) Fire(ctx context.Context, triggerType, eventType string, payload any) []error {
	hooks, err := s.repos.Webhook.List(ctx)
	if err != nil {
		return []error{err}
	}

	var errs []error
	client := &http.Client{Timeout: 10 * time.Second}

	for _, hook := range hooks {
		if !hook.Enabled {
			continue
		}
		if hook.TriggerType != triggerType {
			continue
		}
		if hook.EventType != nil && eventType != "" && *hook.EventType != eventType {
			continue
		}

		body, _ := json.Marshal(payload)
		req, err := http.NewRequestWithContext(ctx, hook.Method, hook.Url, bytes.NewReader(body))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		// Apply custom headers
		var headers map[string]string
		if err := json.Unmarshal(hook.Headers, &headers); err == nil {
			for k, v := range headers {
				req.Header.Set(k, v)
			}
		}

		resp, err := client.Do(req)
		if err != nil {
			errs = append(errs, fmt.Errorf("webhook %d fire: %w", hook.Id, err))
			continue
		}
		resp.Body.Close()

		now := time.Now()
		hook.LastTriggered = &now
		_ = s.repos.Webhook.Update(ctx, &hook)
	}
	return errs
}
