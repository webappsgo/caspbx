package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/casapps/caspbx/src/server/model"
	"github.com/casapps/caspbx/src/server/store"
)

type AsteriskError struct {
	Code    string
	Message string
}

type AsteriskSurfaceItem struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Status string `json:"status"`
	Value  string `json:"value,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type AsteriskSurfaceView struct {
	Surface           model.AsteriskSurface      `json:"surface"`
	DetectionStatus   string                     `json:"detection_status"`
	HealthStatus      model.AsteriskHealthStatus `json:"health_status"`
	MinimumVersion    string                     `json:"minimum_supported_version"`
	DetectedVersion   string                     `json:"detected_version,omitempty"`
	Summary           string                     `json:"summary"`
	Items             []AsteriskSurfaceItem      `json:"items,omitempty"`
	AvailableSurfaces []model.AsteriskSurface    `json:"available_surfaces,omitempty"`
}

type AsteriskService struct {
	store store.AsteriskStore
}

func (errorValue *AsteriskError) Error() string {
	if errorValue == nil {
		return ""
	}
	return errorValue.Message
}

func NewAsteriskService(asteriskStore store.AsteriskStore) AsteriskService {
	return AsteriskService{store: asteriskStore}
}

func (service AsteriskService) Overview(ctx context.Context) (AsteriskSurfaceView, error) {
	state, lookupError := service.store.GetAsteriskState(ctx)
	if lookupError != nil {
		return AsteriskSurfaceView{}, lookupError
	}
	return AsteriskSurfaceView{
		Surface:         model.AsteriskSurface{Key: "overview", Label: "Overview", Summary: "Deployment summary, compatibility floor, and visible operational surfaces."},
		DetectionStatus: state.DetectionStatus,
		HealthStatus:    state.HealthStatus,
		MinimumVersion:  state.MinimumSupportedVersion,
		DetectedVersion: valueOrUnknown(state.DetectedVersion),
		Summary:         fmt.Sprintf("Asterisk admin foundation with %d visible operational surfaces.", len(state.VisibleSurfaces())),
		Items: []AsteriskSurfaceItem{
			{Key: "minimum_supported_version", Label: "Minimum supported version", Status: "supported", Value: state.MinimumSupportedVersion},
			{Key: "detected_version", Label: "Detected version", Status: state.DetectionStatus, Value: valueOrUnknown(state.DetectedVersion)},
			{Key: "health", Label: "Health", Status: string(state.HealthStatus), Value: string(state.HealthStatus)},
			{Key: "visible_surface_count", Label: "Visible surfaces", Status: "ready", Value: fmt.Sprintf("%d", len(state.VisibleSurfaces()))},
		},
		AvailableSurfaces: state.VisibleSurfaces(),
	}, nil
}

func (service AsteriskService) Surface(ctx context.Context, key string) (AsteriskSurfaceView, error) {
	state, lookupError := service.store.GetAsteriskState(ctx)
	if lookupError != nil {
		return AsteriskSurfaceView{}, lookupError
	}

	normalizedKey := normalizeAsteriskSurfaceKey(key)
	switch normalizedKey {
	case "", "overview":
		return service.Overview(ctx)
	case "health":
		return buildAsteriskSurfaceView(state, model.AsteriskSurface{Key: "health", Label: "Health", Summary: "Platform health, detection state, and managed subsystem readiness."}, "Platform and managed subsystem health.", append([]AsteriskSurfaceItem{
			{Key: "health", Label: "Health", Status: string(state.HealthStatus), Value: string(state.HealthStatus)},
			{Key: "detection_status", Label: "Detection status", Status: state.DetectionStatus, Value: state.DetectionStatus},
		}, subsystemItems(state.Subsystems)...)), nil
	case "capabilities":
		return buildAsteriskSurfaceView(state, model.AsteriskSurface{Key: "capabilities", Label: "Capabilities", Summary: "Detected feature families and why unavailable capabilities stay hidden."}, "Capability-gated feature exposure for the active deployment.", capabilityItems(state.Capabilities)), nil
	case "modules":
		items := listItems("channel_driver", "Channel driver", state.ChannelDrivers)
		items = append(items, listItems("endpoint_stack", "Endpoint stack", state.EndpointStacks)...)
		items = append(items, listItems("codec", "Codec", state.Codecs)...)
		return buildAsteriskSurfaceView(state, model.AsteriskSurface{Key: "modules", Label: "Modules", Summary: "Detected channel drivers, endpoint stacks, and codec inventory."}, "Backend module inventory used by capability-driven UI exposure.", items), nil
	case "apply":
		return buildAsteriskSurfaceView(state, model.AsteriskSurface{Key: "apply", Label: "Apply", Summary: "Preview, validation, diff, and post-apply control-loop readiness."}, "Apply pipeline readiness for preview, validation, activation, and post-apply checks.", []AsteriskSurfaceItem{
			{Key: "preview", Label: "Preview", Status: "enabled", Value: "true", Detail: "Operators receive intent-to-config previews before activation."},
			{Key: "validation", Label: "Validation", Status: "enabled", Value: "true", Detail: "Capability and compatibility checks run before apply."},
			{Key: "diff", Label: "Diff visibility", Status: "enabled", Value: "true", Detail: "Generated artifacts can be summarized before reload or restart."},
			{Key: "post_apply_checks", Label: "Post-apply checks", Status: "enabled", Value: "true", Detail: "Activation loops must confirm backend state after apply."},
		}), nil
	case "media":
		if !hasAnyCapability(state, "recordings", "voicemail", "prompts", "music_on_hold") {
			return AsteriskSurfaceView{}, &AsteriskError{Code: "ASTERISK_SURFACE_NOT_FOUND", Message: "surface not available"}
		}
		items := filterCapabilityItems(state.Capabilities, "recordings", "voicemail", "prompts", "music_on_hold")
		items = append(items, subsystemItem(state.Subsystems, "tts_engine", "TTS engine")...)
		items = append(items, subsystemItem(state.Subsystems, "music_on_hold", "Music on hold")...)
		return buildAsteriskSurfaceView(state, model.AsteriskSurface{Key: "media", Label: "Media", Summary: "Prompt, recording, music-on-hold, and media lifecycle readiness."}, "Media plane visibility for prompts, recordings, and music-on-hold workflows.", items), nil
	case "fax":
		if !hasAnyCapability(state, "fax") && !hasSubsystem(state, "fax_backend") {
			return AsteriskSurfaceView{}, &AsteriskError{Code: "ASTERISK_SURFACE_NOT_FOUND", Message: "surface not available"}
		}
		items := filterCapabilityItems(state.Capabilities, "fax")
		items = append(items, subsystemItem(state.Subsystems, "fax_backend", "Fax backend")...)
		return buildAsteriskSurfaceView(state, model.AsteriskSurface{Key: "fax", Label: "Fax", Summary: "Fax/document capability state and backend abstraction health."}, "Fax transport and backend abstraction visibility.", items), nil
	case "messaging":
		if !hasAnyCapability(state, "xmpp", "presence", "mail_delivery") && !hasSubsystem(state, "messaging_backend") {
			return AsteriskSurfaceView{}, &AsteriskError{Code: "ASTERISK_SURFACE_NOT_FOUND", Message: "surface not available"}
		}
		items := filterCapabilityItems(state.Capabilities, "xmpp", "presence", "mail_delivery")
		items = append(items, subsystemItem(state.Subsystems, "messaging_backend", "Messaging backend")...)
		return buildAsteriskSurfaceView(state, model.AsteriskSurface{Key: "messaging", Label: "Messaging", Summary: "Presence, messaging, and mail-delivery readiness for messaging-capable deployments."}, "Presence and messaging readiness for supported deployments.", items), nil
	case "conferences":
		if !hasAnyCapability(state, "conferences") {
			return AsteriskSurfaceView{}, &AsteriskError{Code: "ASTERISK_SURFACE_NOT_FOUND", Message: "surface not available"}
		}
		return buildAsteriskSurfaceView(state, model.AsteriskSurface{Key: "conferences", Label: "Conferences", Summary: "Conference feature-family readiness and moderation support visibility."}, "Conference family readiness.", filterCapabilityItems(state.Capabilities, "conferences")), nil
	case "queues":
		if !hasAnyCapability(state, "queues") {
			return AsteriskSurfaceView{}, &AsteriskError{Code: "ASTERISK_SURFACE_NOT_FOUND", Message: "surface not available"}
		}
		return buildAsteriskSurfaceView(state, model.AsteriskSurface{Key: "queues", Label: "Queues", Summary: "Queue and call-center operational capability visibility."}, "Queue-family readiness for call-center tooling.", filterCapabilityItems(state.Capabilities, "queues")), nil
	case "hardware":
		if !hasAnyCapability(state, "dahdi") {
			return AsteriskSurfaceView{}, &AsteriskError{Code: "ASTERISK_SURFACE_NOT_FOUND", Message: "surface not available"}
		}
		return buildAsteriskSurfaceView(state, model.AsteriskSurface{Key: "hardware", Label: "Hardware", Summary: "DAHDI and hardware-backed telephony capability state."}, "Hardware-backed telephony visibility.", filterCapabilityItems(state.Capabilities, "dahdi")), nil
	case "browser":
		if !hasAnyCapability(state, "browser_calling") {
			return AsteriskSurfaceView{}, &AsteriskError{Code: "ASTERISK_SURFACE_NOT_FOUND", Message: "surface not available"}
		}
		return buildAsteriskSurfaceView(state, model.AsteriskSurface{Key: "browser", Label: "Browser Calling", Summary: "Browser-calling prerequisites, transport readiness, and webphone exposure state."}, "Browser-calling readiness for the webphone surface.", filterCapabilityItems(state.Capabilities, "browser_calling", "tls")), nil
	default:
		return AsteriskSurfaceView{}, &AsteriskError{Code: "ASTERISK_SURFACE_NOT_FOUND", Message: "surface not found"}
	}
}

func buildAsteriskSurfaceView(state model.AsteriskState, surface model.AsteriskSurface, summary string, items []AsteriskSurfaceItem) AsteriskSurfaceView {
	return AsteriskSurfaceView{
		Surface:           surface,
		DetectionStatus:   state.DetectionStatus,
		HealthStatus:      state.HealthStatus,
		MinimumVersion:    state.MinimumSupportedVersion,
		DetectedVersion:   valueOrUnknown(state.DetectedVersion),
		Summary:           summary,
		Items:             items,
		AvailableSurfaces: state.VisibleSurfaces(),
	}
}

func normalizeAsteriskSurfaceKey(key string) string {
	return strings.Trim(strings.TrimSpace(strings.ToLower(key)), "/")
}

func valueOrUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return strings.TrimSpace(value)
}

func capabilityItems(capabilities []model.AsteriskCapability) []AsteriskSurfaceItem {
	items := []AsteriskSurfaceItem{}
	for _, capability := range capabilities {
		status := "unavailable"
		if capability.Available {
			status = "available"
		}
		items = append(items, AsteriskSurfaceItem{
			Key:    capability.Key,
			Label:  capability.Label,
			Status: status,
			Value:  capability.Family,
			Detail: capability.Reason,
		})
	}
	return items
}

func filterCapabilityItems(capabilities []model.AsteriskCapability, keys ...string) []AsteriskSurfaceItem {
	allowed := map[string]struct{}{}
	for _, key := range keys {
		allowed[key] = struct{}{}
	}
	items := []AsteriskSurfaceItem{}
	for _, capability := range capabilities {
		if _, found := allowed[capability.Key]; !found {
			continue
		}
		status := "unavailable"
		if capability.Available {
			status = "available"
		}
		items = append(items, AsteriskSurfaceItem{
			Key:    capability.Key,
			Label:  capability.Label,
			Status: status,
			Value:  capability.Family,
			Detail: capability.Reason,
		})
	}
	return items
}

func subsystemItems(subsystems []model.AsteriskManagedSubsystem) []AsteriskSurfaceItem {
	items := []AsteriskSurfaceItem{}
	for _, subsystem := range subsystems {
		status := "degraded"
		if subsystem.Healthy {
			status = "ready"
		}
		items = append(items, AsteriskSurfaceItem{
			Key:    subsystem.Key,
			Label:  subsystem.Label,
			Status: status,
			Value:  subsystem.Provider,
			Detail: subsystem.Reason,
		})
	}
	return items
}

func subsystemItem(subsystems []model.AsteriskManagedSubsystem, key string, label string) []AsteriskSurfaceItem {
	for _, subsystem := range subsystems {
		if subsystem.Key != key {
			continue
		}
		status := "degraded"
		if subsystem.Healthy {
			status = "ready"
		}
		return []AsteriskSurfaceItem{{
			Key:    subsystem.Key,
			Label:  label,
			Status: status,
			Value:  subsystem.Provider,
			Detail: subsystem.Reason,
		}}
	}
	return nil
}

func listItems(prefix string, label string, values []string) []AsteriskSurfaceItem {
	items := []AsteriskSurfaceItem{}
	for index, value := range values {
		items = append(items, AsteriskSurfaceItem{
			Key:    fmt.Sprintf("%s_%d", prefix, index+1),
			Label:  label,
			Status: "detected",
			Value:  value,
		})
	}
	if len(items) == 0 {
		items = append(items, AsteriskSurfaceItem{
			Key:    prefix + "_none",
			Label:  label,
			Status: "pending",
			Value:  "none detected",
			Detail: "Capability detection has not published any entries for this family yet.",
		})
	}
	return items
}

func hasAnyCapability(state model.AsteriskState, keys ...string) bool {
	for _, key := range keys {
		capability, found := state.Capability(key)
		if found && capability.Available {
			return true
		}
	}
	return false
}

func hasSubsystem(state model.AsteriskState, key string) bool {
	for _, subsystem := range state.Subsystems {
		if subsystem.Key == key && strings.TrimSpace(subsystem.Provider) != "" {
			return true
		}
	}
	return false
}

var _ error = (*AsteriskError)(nil)
