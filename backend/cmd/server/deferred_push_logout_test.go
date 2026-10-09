package main

import (
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"net/http"
	"testing"
)

func TestDeferredPushLogoutDesiredHTTPContracts(t *testing.T) {
	for _, scope := range []string{"device", "all"} {
		t.Run(scope, func(t *testing.T) {
			s := newGoldenServer(t)
			seedGoldenMember(t, s.db)
			user, err := repository.NewAuthRepository(s.db).GetMemberBySeq(goldenMemberID)
			if err != nil {
				t.Fatal(err)
			}
			session, err := service.NewMobileSessionIssuer(s.deps.authService).Issue(user)
			if err != nil {
				t.Fatal(err)
			}
			body := map[string]string{"refreshToken": session.RefreshToken}
			path := "/api/auth/logout/all/deferred"
			if scope == "device" {
				path = "/api/auth/logout/deferred"
				body["deviceToken"] = "synthetic-device"
			}
			s.request(t, http.MethodPost, path, body, "", "ios", "100", 204)
		})
	}
}
