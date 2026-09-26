// allowed_origins.go — CORS/CSRF origin list from the user-site and admin-site settings
package main

import (
	"strings"

	"github.com/dflh-saf/backend/internal/config"
)

// configuredOrigins returns the user-site origins (comma-separated ALLOWED_ORIGIN)
// plus the admin SPA origin, which is served from its own host.
func configuredOrigins(server config.ServerConfig) []string {
	origins := []string{}
	for _, origin := range strings.Split(server.AllowedOrigin, ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			origins = append(origins, origin)
		}
	}
	if admin := strings.TrimSpace(server.AdminOrigin); admin != "" {
		origins = append(origins, admin)
	}
	return origins
}
