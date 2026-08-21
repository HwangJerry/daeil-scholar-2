package service

import "time"

const defaultPushProviderRequestTimeout = 5 * time.Second

func normalizePushProviderRequestTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return defaultPushProviderRequestTimeout
	}
	return timeout
}
