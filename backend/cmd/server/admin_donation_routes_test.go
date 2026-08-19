package main

import (
	"net/http"
	"testing"

	"github.com/dflh-saf/backend/internal/handler"
	"github.com/go-chi/chi/v5"
)

func TestRegisterAdminDonationRoutes(t *testing.T) {
	router := chi.NewRouter()
	registerAdminDonationRoutes(
		router,
		handler.NewAdminDonationHandler(nil),
		handler.NewAdminDonationImportHandler(nil, 10),
	)

	tests := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/orders"},
		{http.MethodGet, "/orders/3001"},
		{http.MethodPost, "/orders"},
		{http.MethodPut, "/orders/3001"},
		{http.MethodPost, "/import/commit"},
	}
	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			context := chi.NewRouteContext()
			if !router.Match(context, test.method, test.path) {
				t.Fatalf("route %s %s is not registered", test.method, test.path)
			}
		})
	}
}

func TestRegisterAdminDonationImportPreviewRoute(t *testing.T) {
	router := chi.NewRouter()
	registerAdminDonationImportPreviewRoute(router, handler.NewAdminDonationImportHandler(nil, 10))

	context := chi.NewRouteContext()
	if !router.Match(context, http.MethodPost, "/api/admin/donation/import/preview") {
		t.Fatal("donation import preview route is not registered")
	}
}
