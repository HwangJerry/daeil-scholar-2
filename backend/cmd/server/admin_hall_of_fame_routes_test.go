// admin_hall_of_fame_routes_test.go — Hall of Fame admin routes stay registered.
package main

import (
	"net/http"
	"testing"

	"github.com/dflh-saf/backend/internal/handler"
	"github.com/go-chi/chi/v5"
)

func TestRegisterAdminHallOfFameRoutes(t *testing.T) {
	outer := chi.NewRouter()
	outer.Route("/hall-of-fame", func(router chi.Router) {
		registerAdminHallOfFameRoutes(router, handler.NewAdminHallOfFameHandler(nil), handler.NewAdminHallOfFameImportHandler(nil, 10))
	})
	registerAdminHallOfFameImportPreviewRoute(outer, handler.NewAdminHallOfFameImportHandler(nil, 10))
	for _, test := range []struct{ method, path string }{
		{http.MethodGet, "/hall-of-fame"},
		{http.MethodPost, "/hall-of-fame"},
		{http.MethodPatch, "/hall-of-fame/1"},
		{http.MethodDelete, "/hall-of-fame/1"},
		{http.MethodPost, "/hall-of-fame/import/commit"},
		{http.MethodPost, "/api/admin/hall-of-fame/import/preview"},
	} {
		if !outer.Match(chi.NewRouteContext(), test.method, test.path) {
			t.Fatalf("route %s %s is not registered", test.method, test.path)
		}
	}
}
