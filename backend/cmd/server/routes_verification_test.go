package main

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/patrickmn/go-cache"
)

func TestAuthenticatedRoutesIncludeAlumniVerificationEndpoints(t *testing.T) {
	router := chi.NewRouter()
	registerAuthRoutes(router, handlers{}, nil)

	found := map[string]bool{}
	if err := chi.Walk(router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		found[method+" "+route] = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	for _, route := range []string{
		http.MethodGet + " /api/alumni/verification",
		http.MethodPut + " /api/alumni/verification",
	} {
		if !found[route] {
			t.Fatalf("missing route %s", route)
		}
	}
}

func TestAuthenticatedRoutesIncludeIdentityLinkEndpoint(t *testing.T) {
	router := chi.NewRouter()
	registerAuthRoutes(router, handlers{}, nil)

	found := routesForTest(t, router)
	route := http.MethodPost + " /api/auth/identities/link/{provider}"
	if !found[route] {
		t.Fatalf("missing route %s", route)
	}
}

func TestAlumniWidgetIsMountedOnlyInApprovedAuthenticatedRoutes(t *testing.T) {
	publicRouter := chi.NewRouter()
	registerPublicRoutes(publicRouter, handlers{}, nil, cache.New(0, 0))
	if routesForTest(t, publicRouter)[http.MethodGet+" /api/alumni/widget"] {
		t.Fatal("widget remains publicly mounted")
	}

	authenticatedRouter := chi.NewRouter()
	registerAuthRoutes(authenticatedRouter, handlers{}, nil)
	if !routesForTest(t, authenticatedRouter)[http.MethodGet+" /api/alumni/widget"] {
		t.Fatal("widget is not mounted in authenticated routes")
	}
}

func TestApprovedRoutesIncludeCanonicalAlumniSearchEndpoints(t *testing.T) {
	router := chi.NewRouter()
	registerAuthRoutes(router, handlers{}, nil)
	found := routesForTest(t, router)

	for _, route := range []string{
		http.MethodGet + " /api/alumni",
		http.MethodGet + " /api/alumni/filters",
		http.MethodGet + " /api/alumni/{userSeq}",
		http.MethodGet + " /api/alumni/widget",
	} {
		if !found[route] {
			t.Fatalf("missing route %s", route)
		}
	}
}

func TestApprovedRoutesIncludeCanonicalMemberBlockEndpoints(t *testing.T) {
	router := chi.NewRouter()
	registerAuthRoutes(router, handlers{}, nil)
	found := routesForTest(t, router)

	for _, route := range []string{
		http.MethodGet + " /api/blocks",
		http.MethodGet + " /api/blocks/{userSeq}",
		http.MethodPut + " /api/blocks/{userSeq}",
		http.MethodDelete + " /api/blocks/{userSeq}",
	} {
		if !found[route] {
			t.Fatalf("missing route %s", route)
		}
	}
}

func TestApprovedRoutesIncludeCanonicalPushEndpoints(t *testing.T) {
	router := chi.NewRouter()
	registerAuthRoutes(router, handlers{}, nil)
	found := routesForTest(t, router)

	for _, route := range []string{
		http.MethodPost + " /api/push/device/register",
		http.MethodPost + " /api/push/device/unregister",
		http.MethodGet + " /api/push/preferences",
		http.MethodPut + " /api/push/preferences",
	} {
		if !found[route] {
			t.Fatalf("missing route %s", route)
		}
	}
}

func TestMobileEventCollectionIsPublicAndAdminSummaryIsAdminOnly(t *testing.T) {
	publicRouter := chi.NewRouter()
	registerPublicRoutes(publicRouter, handlers{}, nil, cache.New(0, 0))
	if !routesForTest(t, publicRouter)[http.MethodPost+" /api/mobile/events"] {
		t.Fatal("mobile event collection route is not publicly mounted")
	}
	if routesForTest(t, publicRouter)[http.MethodGet+" /api/admin/mobile-events/summary"] {
		t.Fatal("admin mobile event summary is publicly mounted")
	}

	adminRouter := chi.NewRouter()
	registerAdminRoutes(adminRouter, handlers{}, nil, nil)
	if !routesForTest(t, adminRouter)[http.MethodGet+" /api/admin/mobile-events/summary"] {
		t.Fatal("admin mobile event summary route is missing")
	}
}

func TestSentryMonitoringRoutesAreAdminOnly(t *testing.T) {
	publicRouter := chi.NewRouter()
	registerPublicRoutes(publicRouter, handlers{}, nil, cache.New(0, 0))
	adminRouter := chi.NewRouter()
	registerAdminRoutes(adminRouter, handlers{}, nil, nil)

	publicRoutes := routesForTest(t, publicRouter)
	adminRoutes := routesForTest(t, adminRouter)
	for _, route := range []string{
		http.MethodGet + " /api/admin/monitoring/crash-summary",
		http.MethodGet + " /api/admin/monitoring/performance-summary",
	} {
		if publicRoutes[route] {
			t.Fatalf("admin monitoring route is public: %s", route)
		}
		if !adminRoutes[route] {
			t.Fatalf("admin monitoring route is missing: %s", route)
		}
	}
}

func TestAdminRoutesIncludeAlumniVerificationReviewEndpoints(t *testing.T) {
	router := chi.NewRouter()
	registerAdminRoutes(router, handlers{}, nil, nil)
	found := routesForTest(t, router)

	for _, route := range []string{
		http.MethodGet + " /api/admin/alumni-verifications",
		http.MethodGet + " /api/admin/alumni-verifications/{userSeq}",
		http.MethodPost + " /api/admin/alumni-verifications/{userSeq}/approve",
		http.MethodPost + " /api/admin/alumni-verifications/{userSeq}/reject",
	} {
		if !found[route] {
			t.Fatalf("missing route %s", route)
		}
	}
}

func routesForTest(t *testing.T, router chi.Router) map[string]bool {
	t.Helper()
	found := map[string]bool{}
	if err := chi.Walk(router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		found[method+" "+route] = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return found
}
