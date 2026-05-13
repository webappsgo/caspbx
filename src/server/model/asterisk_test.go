package model

import "testing"

func TestDefaultAsteriskStateAndHelpers(t *testing.T) {
	state := DefaultAsteriskState()
	if state.MinimumSupportedVersion != "12" || state.DetectionStatus != "pending" || state.HealthStatus != AsteriskHealthUnknown {
		t.Fatalf("unexpected default asterisk state %+v", state)
	}

	state.Capabilities = []AsteriskCapability{
		{Key: "fax", Label: "Fax", Available: true},
		{Key: "browser_calling", Label: "Browser Calling", Available: true},
		{Key: "prompts", Label: "Prompts", Available: true},
		{Key: "presence", Label: "Presence", Available: true},
		{Key: "conferences", Label: "Conferences", Available: true},
		{Key: "queues", Label: "Queues", Available: true},
		{Key: "dahdi", Label: "DAHDI", Available: true},
	}
	state.ChannelDrivers = []string{"pjsip"}
	state.Subsystems = []AsteriskManagedSubsystem{{Key: "fax_backend", Provider: "hylafax+", Healthy: true}}

	if capability, found := state.Capability("fax"); !found || capability.Label != "Fax" {
		t.Fatalf("expected fax capability lookup, got %t / %+v", found, capability)
	}
	if _, found := state.Capability("missing"); found {
		t.Fatalf("expected missing capability lookup to fail")
	}
	available := state.AvailableCapabilities()
	if len(available) != 7 || available[0].Key != "browser_calling" {
		t.Fatalf("unexpected available capability list %+v", available)
	}
	surfaces := state.VisibleSurfaces()
	if len(surfaces) < 12 || surfaces[0].Key != "overview" || surfaces[len(surfaces)-1].Key != "browser" {
		t.Fatalf("unexpected visible surfaces %+v", surfaces)
	}
	foundOperator := false
	foundCallCenter := false
	for _, surface := range surfaces {
		if surface.Key == "operator" {
			foundOperator = true
		}
		if surface.Key == "callcenter" {
			foundCallCenter = true
		}
	}
	if !foundOperator || !foundCallCenter {
		t.Fatalf("expected operator and callcenter surfaces, got %+v", surfaces)
	}
	if !state.hasAnyCapability("prompts") || !state.hasSubsystem("fax_backend") {
		t.Fatalf("expected helper lookups to succeed")
	}
	if state.hasAnyCapability("mail_delivery") || state.hasSubsystem("messaging_backend") {
		t.Fatalf("expected unavailable helper lookups to fail")
	}
}
