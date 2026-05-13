package model

import (
	"slices"
	"strings"
	"time"
)

type AsteriskHealthStatus string

const (
	AsteriskHealthUnknown  AsteriskHealthStatus = "unknown"
	AsteriskHealthReady    AsteriskHealthStatus = "ready"
	AsteriskHealthDegraded AsteriskHealthStatus = "degraded"
	AsteriskHealthError    AsteriskHealthStatus = "error"
)

type AsteriskCapability struct {
	Key       string
	Label     string
	Family    string
	Available bool
	Reason    string
}

type AsteriskManagedSubsystem struct {
	Key      string
	Label    string
	Provider string
	Healthy  bool
	Reason   string
}

type AsteriskSurface struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Summary string `json:"summary"`
}

type AsteriskState struct {
	MinimumSupportedVersion string
	DetectedVersion         string
	DetectionStatus         string
	HealthStatus            AsteriskHealthStatus
	ChannelDrivers          []string
	EndpointStacks          []string
	Codecs                  []string
	Capabilities            []AsteriskCapability
	Subsystems              []AsteriskManagedSubsystem
	UpdatedAt               time.Time
}

func DefaultAsteriskState() AsteriskState {
	return AsteriskState{
		MinimumSupportedVersion: "12",
		DetectionStatus:         "pending",
		HealthStatus:            AsteriskHealthUnknown,
		ChannelDrivers:          []string{},
		EndpointStacks:          []string{},
		Codecs:                  []string{},
		Capabilities:            []AsteriskCapability{},
		Subsystems:              []AsteriskManagedSubsystem{},
	}
}

func (state AsteriskState) Capability(key string) (AsteriskCapability, bool) {
	normalizedKey := strings.TrimSpace(strings.ToLower(key))
	for _, capability := range state.Capabilities {
		if capability.Key == normalizedKey {
			return capability, true
		}
	}
	return AsteriskCapability{}, false
}

func (state AsteriskState) AvailableCapabilities() []AsteriskCapability {
	available := []AsteriskCapability{}
	for _, capability := range state.Capabilities {
		if capability.Available {
			available = append(available, capability)
		}
	}
	sortAsteriskCapabilities(available)
	return available
}

func (state AsteriskState) VisibleSurfaces() []AsteriskSurface {
	surfaces := []AsteriskSurface{
		{Key: "overview", Label: "Overview", Summary: "Deployment summary, compatibility floor, and visible operational surfaces."},
		{Key: "health", Label: "Health", Summary: "Platform health, detection state, and managed subsystem readiness."},
		{Key: "capabilities", Label: "Capabilities", Summary: "Detected feature families and why unavailable capabilities stay hidden."},
		{Key: "modules", Label: "Modules", Summary: "Detected channel drivers, endpoint stacks, and codec inventory."},
		{Key: "apply", Label: "Apply", Summary: "Preview, validation, diff, and post-apply control-loop readiness."},
	}

	if state.hasAnyCapability("recordings", "voicemail", "prompts", "music_on_hold") {
		surfaces = append(surfaces, AsteriskSurface{
			Key:     "media",
			Label:   "Media",
			Summary: "Prompt, recording, music-on-hold, and media lifecycle readiness.",
		})
	}
	if state.hasAnyCapability("fax") || state.hasSubsystem("fax_backend") {
		surfaces = append(surfaces, AsteriskSurface{
			Key:     "fax",
			Label:   "Fax",
			Summary: "Fax/document capability state and backend abstraction health.",
		})
	}
	if state.hasAnyCapability("xmpp", "presence", "mail_delivery") || state.hasSubsystem("messaging_backend") {
		surfaces = append(surfaces, AsteriskSurface{
			Key:     "messaging",
			Label:   "Messaging",
			Summary: "Presence, messaging, and mail-delivery readiness for messaging-capable deployments.",
		})
	}
	if state.hasAnyCapability("conferences") {
		surfaces = append(surfaces, AsteriskSurface{
			Key:     "conferences",
			Label:   "Conferences",
			Summary: "Conference feature-family readiness and moderation support visibility.",
		})
	}
	if state.hasAnyCapability("queues") {
		surfaces = append(surfaces, AsteriskSurface{
			Key:     "callcenter",
			Label:   "Call Center",
			Summary: "Queue, agent, and supervisor operational visibility for supported deployments.",
		})
		surfaces = append(surfaces, AsteriskSurface{
			Key:     "queues",
			Label:   "Queues",
			Summary: "Queue and call-center operational capability visibility.",
		})
	}
	if len(state.ChannelDrivers) > 0 || len(state.EndpointStacks) > 0 {
		surfaces = append(surfaces, AsteriskSurface{
			Key:     "operator",
			Label:   "Operator",
			Summary: "Switchboard-style live telephony visibility and fast operator actions.",
		})
	}
	if state.hasAnyCapability("dahdi") {
		surfaces = append(surfaces, AsteriskSurface{
			Key:     "hardware",
			Label:   "Hardware",
			Summary: "DAHDI and hardware-backed telephony capability state.",
		})
	}
	if state.hasAnyCapability("browser_calling") {
		surfaces = append(surfaces, AsteriskSurface{
			Key:     "browser",
			Label:   "Browser Calling",
			Summary: "Browser-calling prerequisites, transport readiness, and webphone exposure state.",
		})
	}

	sortAsteriskSurfaces(surfaces)
	return surfaces
}

func (state AsteriskState) hasAnyCapability(keys ...string) bool {
	for _, key := range keys {
		if capability, found := state.Capability(key); found && capability.Available {
			return true
		}
	}
	return false
}

func (state AsteriskState) hasSubsystem(key string) bool {
	normalizedKey := strings.TrimSpace(strings.ToLower(key))
	for _, subsystem := range state.Subsystems {
		if subsystem.Key == normalizedKey && strings.TrimSpace(subsystem.Provider) != "" {
			return true
		}
	}
	return false
}

func sortAsteriskCapabilities(capabilities []AsteriskCapability) {
	slices.SortFunc(capabilities, func(left AsteriskCapability, right AsteriskCapability) int {
		return strings.Compare(left.Key, right.Key)
	})
}

func sortAsteriskSurfaces(surfaces []AsteriskSurface) {
	order := map[string]int{
		"overview":     0,
		"health":       1,
		"capabilities": 2,
		"modules":      3,
		"apply":        4,
		"media":        5,
		"fax":          6,
		"messaging":    7,
		"callcenter":   8,
		"conferences":  9,
		"queues":       10,
		"operator":     11,
		"hardware":     12,
		"browser":      13,
	}
	slices.SortFunc(surfaces, func(left AsteriskSurface, right AsteriskSurface) int {
		return order[left.Key] - order[right.Key]
	})
}
