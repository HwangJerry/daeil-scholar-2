// sentry_user_events.go — Count Sentry events that reference a member, for erasure verification.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// CountUserEvents returns how many events in the last 90 days, across all
// projects, match the Sentry search query.
func (c *SentryClient) CountUserEvents(ctx context.Context, query string) (int, error) {
	if err := c.checkConfigured(); err != nil {
		return 0, err
	}
	values := url.Values{}
	values.Set("field", "count()")
	values.Set("query", query)
	values.Set("statsPeriod", "90d")
	values.Set("project", "-1")
	var payload struct {
		Data []map[string]json.Number `json:"data"`
	}
	if err := c.getJSON(ctx, c.organizationPath("events/"), values, &payload); err != nil {
		return 0, err
	}
	if len(payload.Data) == 0 {
		return 0, nil
	}
	count, err := payload.Data[0]["count()"].Int64()
	if err != nil {
		return 0, fmt.Errorf("unexpected sentry count payload")
	}
	return int(count), nil
}
