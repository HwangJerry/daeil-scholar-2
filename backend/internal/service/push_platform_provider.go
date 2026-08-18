package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog"
)

type MissingPushProviderError struct {
	Platform string
}

func (e *MissingPushProviderError) Error() string {
	return fmt.Sprintf("push provider is not configured for platform %q", e.Platform)
}

type platformPushProvider struct {
	providers map[string]MobilePushProvider
	logger    zerolog.Logger
}

func NewPlatformPushProvider(logger zerolog.Logger, providers map[string]MobilePushProvider) MobilePushProvider {
	filtered := make(map[string]MobilePushProvider, len(providers))
	for platform, provider := range providers {
		platform = strings.ToLower(strings.TrimSpace(platform))
		if platform != "" && provider != nil {
			filtered[platform] = provider
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return &platformPushProvider{
		providers: filtered,
		logger:    logger,
	}
}

func (p *platformPushProvider) SendPush(ctx context.Context, notification PushNotification) error {
	platform := strings.ToLower(strings.TrimSpace(notification.Platform))
	provider := p.providers[platform]
	if provider == nil {
		return &MissingPushProviderError{Platform: platform}
	}
	return provider.SendPush(ctx, notification)
}
