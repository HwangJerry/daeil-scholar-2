// app_update_evaluator_test.go — Decision rules for app update enforcement.
package service

import (
	"testing"

	"github.com/dflh-saf/backend/internal/model"
)

func iosPolicy() model.AppUpdatePolicy {
	return model.AppUpdatePolicy{
		ForceEnabled:     true,
		MinBuild:         200,
		RecommendEnabled: true,
		RecommendedBuild: 300,
		MinOSVersion:     "17.0",
		StoreURL:         "https://apps.apple.com/app/id1",
	}
}

func TestEvaluateAppUpdate(t *testing.T) {
	disabled := iosPolicy()
	disabled.ForceEnabled = false
	disabled.RecommendEnabled = false

	forceOnly := iosPolicy()
	forceOnly.RecommendEnabled = false

	tests := []struct {
		name      string
		policy    model.AppUpdatePolicy
		build     int64
		osVersion string
		want      model.AppUpdateDecision
	}{
		{"below minimum forces", iosPolicy(), 199, "18.1", model.AppUpdateDecisionForce},
		{"at minimum does not force", iosPolicy(), 200, "18.1", model.AppUpdateDecisionRecommend},
		{"below recommended recommends", iosPolicy(), 299, "18.1", model.AppUpdateDecisionRecommend},
		{"at recommended is clean", iosPolicy(), 300, "18.1", model.AppUpdateDecisionNone},
		{"toggles off means none", disabled, 1, "18.1", model.AppUpdateDecisionNone},
		{"recommend toggle off leaves force only", forceOnly, 250, "18.1", model.AppUpdateDecisionNone},
		{"unsupported os wins over force", iosPolicy(), 100, "16.7.2", model.AppUpdateDecisionUnsupportedOS},
		{"equal os is supported", iosPolicy(), 199, "17.0", model.AppUpdateDecisionForce},
		{"newer patch os is supported", iosPolicy(), 199, "17.0.1", model.AppUpdateDecisionForce},
		{"unknown build never blocks", iosPolicy(), 0, "18.1", model.AppUpdateDecisionNone},
		{"unparsable os never blocks on os", iosPolicy(), 400, "not-a-version", model.AppUpdateDecisionNone},
		{"missing os header still enforces build", iosPolicy(), 199, "", model.AppUpdateDecisionForce},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := EvaluateAppUpdate(test.policy, test.build, test.osVersion); got != test.want {
				t.Fatalf("EvaluateAppUpdate() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestEvaluateAppUpdateUsesAndroidAPILevels(t *testing.T) {
	policy := model.AppUpdatePolicy{
		ForceEnabled: true,
		MinBuild:     500,
		MinOSVersion: "26",
	}

	if got := EvaluateAppUpdate(policy, 400, "25"); got != model.AppUpdateDecisionUnsupportedOS {
		t.Fatalf("API 25 decision = %q", got)
	}
	if got := EvaluateAppUpdate(policy, 400, "26"); got != model.AppUpdateDecisionForce {
		t.Fatalf("API 26 decision = %q", got)
	}
}

// A threshold left at zero must never block, even if the toggle was switched on
// through a path that skipped validation.
func TestEvaluateAppUpdateIgnoresZeroThresholds(t *testing.T) {
	policy := model.AppUpdatePolicy{ForceEnabled: true, RecommendEnabled: true, MinOSVersion: "17.0"}

	if got := EvaluateAppUpdate(policy, 10, "18.0"); got != model.AppUpdateDecisionNone {
		t.Fatalf("decision = %q, want none", got)
	}
}
