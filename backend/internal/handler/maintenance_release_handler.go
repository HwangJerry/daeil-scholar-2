// maintenance_release_handler.go — Loopback control plane for atomic maintenance release handoff.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/dflh-saf/backend/internal/maintenance"
)

const maintenanceReleaseRequestLimit = 4096

type MaintenanceReleaseHandler struct {
	gate         *maintenance.Gate
	drainTimeout time.Duration
}

type maintenanceReleaseRequest struct {
	Generation        string `json:"generation"`
	ApprovalAttemptID string `json:"approval_attempt_id"`
}

type maintenanceReleaseResponse struct {
	State string `json:"state"`
}

func NewMaintenanceReleaseHandler(gate *maintenance.Gate, drainTimeout time.Duration) *MaintenanceReleaseHandler {
	return &MaintenanceReleaseHandler{gate: gate, drainTimeout: drainTimeout}
}

func (h *MaintenanceReleaseHandler) Drain(w http.ResponseWriter, r *http.Request) {
	if !h.authorized(r, maintenance.ReleaseDrainPath) {
		respondError(w, http.StatusForbidden, "MAINTENANCE_RELEASE_FORBIDDEN", "Maintenance release control is unavailable")
		return
	}
	request, ok := decodeMaintenanceReleaseRequest(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), h.drainTimeout)
	defer cancel()
	if err := h.gate.CloseAndDrain(ctx, request.Generation, request.ApprovalAttemptID); err != nil {
		status := http.StatusConflict
		code := "MAINTENANCE_RELEASE_AUTHORITY_INVALID"
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			status = http.StatusServiceUnavailable
			code = "MAINTENANCE_RELEASE_DRAIN_INCOMPLETE"
		}
		respondError(w, status, code, "Maintenance release remains blocked")
		return
	}
	respondJSON(w, http.StatusOK, maintenanceReleaseResponse{State: "DRAINED"})
}

func (h *MaintenanceReleaseHandler) ArmOpen(w http.ResponseWriter, r *http.Request) {
	if !h.authorized(r, maintenance.ReleaseArmOpenPath) {
		respondError(w, http.StatusForbidden, "MAINTENANCE_RELEASE_FORBIDDEN", "Maintenance release control is unavailable")
		return
	}
	request, ok := decodeMaintenanceReleaseRequest(w, r)
	if !ok {
		return
	}
	if err := h.gate.ArmOpen(request.Generation, request.ApprovalAttemptID); err != nil {
		respondError(w, http.StatusConflict, "MAINTENANCE_RELEASE_AUTHORITY_INVALID", "Maintenance release remains blocked")
		return
	}
	respondJSON(w, http.StatusOK, maintenanceReleaseResponse{State: "ARMED"})
}

func (h *MaintenanceReleaseHandler) authorized(r *http.Request, path string) bool {
	if h == nil || h.gate == nil || r.URL.Path != path {
		return false
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback() && h.gate.AllowsRelease(path, r.Header.Get(maintenance.ReleaseProofHeader))
}

func decodeMaintenanceReleaseRequest(w http.ResponseWriter, r *http.Request) (maintenanceReleaseRequest, bool) {
	var request maintenanceReleaseRequest
	r.Body = http.MaxBytesReader(w, r.Body, maintenanceReleaseRequestLimit)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid maintenance release request")
		return maintenanceReleaseRequest{}, false
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid maintenance release request")
		return maintenanceReleaseRequest{}, false
	}
	return request, true
}
