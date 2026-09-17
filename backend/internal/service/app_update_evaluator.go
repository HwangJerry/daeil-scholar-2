// app_update_evaluator.go — Pure decision rules for app update enforcement.
package service

import (
	"strconv"
	"strings"

	"github.com/dflh-saf/backend/internal/model"
)

const maxOSVersionSegments = 3

// EvaluateAppUpdate decides what a client running build/osVersion should see.
//
// The rules are intentionally fail-open: an unknown build or an unparsable OS
// version never blocks a user. Clients implement the same order, so the app and
// the API agree on every outcome.
//
//  1. build <= 0                                    → none (unknown client)
//  2. nothing to update to                          → none
//  3. osVersion < policy.MinOSVersion               → unsupported_os (cannot update)
//  4. ForceEnabled && build < MinBuild              → force
//  5. otherwise                                     → recommend
//
// The OS check runs only once the build is actually behind: a user already on
// the newest build has nothing to be told about, whatever their OS version.
func EvaluateAppUpdate(policy model.AppUpdatePolicy, build int64, osVersion string) model.AppUpdateDecision {
	if build <= 0 {
		return model.AppUpdateDecisionNone
	}
	needsForce := policy.ForceEnabled && policy.MinBuild > 0 && build < policy.MinBuild
	needsRecommend := policy.RecommendEnabled && policy.RecommendedBuild > 0 && build < policy.RecommendedBuild
	if !needsForce && !needsRecommend {
		return model.AppUpdateDecisionNone
	}
	if isOSVersionBelow(osVersion, policy.MinOSVersion) {
		return model.AppUpdateDecisionUnsupportedOS
	}
	if needsForce {
		return model.AppUpdateDecisionForce
	}
	return model.AppUpdateDecisionRecommend
}

// isOSVersionBelow reports whether deviceVersion is lower than minimumVersion.
// Either value being absent or unparsable yields false, so a malformed header
// never takes an app away from its user.
func isOSVersionBelow(deviceVersion, minimumVersion string) bool {
	device, deviceOK := parseOSVersion(deviceVersion)
	minimum, minimumOK := parseOSVersion(minimumVersion)
	if !deviceOK || !minimumOK {
		return false
	}
	for index := 0; index < maxOSVersionSegments; index++ {
		if device[index] != minimum[index] {
			return device[index] < minimum[index]
		}
	}
	return false
}

// parseOSVersion reads "17", "17.4" or "17.4.1" (and Android API levels such as
// "26") into a fixed-length segment array.
func parseOSVersion(version string) ([maxOSVersionSegments]int, bool) {
	var parsed [maxOSVersionSegments]int
	trimmed := strings.TrimSpace(version)
	if trimmed == "" {
		return parsed, false
	}
	segments := strings.Split(trimmed, ".")
	if len(segments) > maxOSVersionSegments {
		return parsed, false
	}
	for index, segment := range segments {
		value, err := strconv.Atoi(segment)
		if err != nil || value < 0 {
			return parsed, false
		}
		parsed[index] = value
	}
	return parsed, true
}
