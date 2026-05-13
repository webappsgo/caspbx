package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/casapps/caspbx/src/server/model"
	"github.com/casapps/caspbx/src/server/store"
)

type failingAsteriskStore struct {
	state model.AsteriskState
	err   error
}

func (asteriskStore failingAsteriskStore) SaveAsteriskState(context.Context, model.AsteriskState) (model.AsteriskState, error) {
	return asteriskStore.state, asteriskStore.err
}

func (asteriskStore failingAsteriskStore) GetAsteriskState(context.Context) (model.AsteriskState, error) {
	if asteriskStore.err != nil {
		return model.AsteriskState{}, asteriskStore.err
	}
	return asteriskStore.state, nil
}

func TestAsteriskServiceOverviewAndSurfaces(t *testing.T) {
	memoryStore := store.NewMemoryStore()
	if _, saveError := memoryStore.SaveAsteriskState(context.Background(), model.AsteriskState{
		MinimumSupportedVersion: "12",
		DetectedVersion:         "20.5.1",
		DetectionStatus:         "detected",
		HealthStatus:            model.AsteriskHealthReady,
		ChannelDrivers:          []string{"pjsip", "iax2"},
		EndpointStacks:          []string{"pjsip"},
		Codecs:                  []string{"ulaw", "opus"},
		Capabilities: []model.AsteriskCapability{
			{Key: "recordings", Label: "Recordings", Family: "media", Available: true},
			{Key: "voicemail", Label: "Voicemail", Family: "media", Available: true},
			{Key: "prompts", Label: "Prompts", Family: "media", Available: true},
			{Key: "music_on_hold", Label: "Music on Hold", Family: "media", Available: true},
			{Key: "fax", Label: "Fax", Family: "fax", Available: true},
			{Key: "queues", Label: "Queues", Family: "queue", Available: true},
			{Key: "conferences", Label: "Conferences", Family: "conference", Available: true},
			{Key: "browser_calling", Label: "Browser Calling", Family: "webphone", Available: true},
			{Key: "tls", Label: "TLS", Family: "security", Available: true},
			{Key: "dahdi", Label: "DAHDI", Family: "hardware", Available: true},
			{Key: "presence", Label: "Presence", Family: "messaging", Available: true},
		},
		Subsystems: []model.AsteriskManagedSubsystem{
			{Key: "fax_backend", Label: "Fax backend", Provider: "hylafax+", Healthy: true},
			{Key: "tts_engine", Label: "TTS engine", Provider: "flite", Healthy: true},
			{Key: "music_on_hold", Label: "Music on hold", Provider: "mixed", Healthy: true},
			{Key: "messaging_backend", Label: "Messaging backend", Provider: "xmpp", Healthy: false, Reason: "broker offline"},
		},
		UpdatedAt: time.Unix(1_700_400_000, 0).UTC(),
	}); saveError != nil {
		t.Fatalf("save asterisk state: %v", saveError)
	}

	asteriskService := NewAsteriskService(memoryStore)
	overview, overviewError := asteriskService.Overview(context.Background())
	if overviewError != nil || overview.Surface.Key != "overview" || len(overview.AvailableSurfaces) < 8 {
		t.Fatalf("unexpected asterisk overview %v / %+v", overviewError, overview)
	}
	modulesView, modulesError := asteriskService.Surface(context.Background(), "modules")
	if modulesError != nil || len(modulesView.Items) < 3 {
		t.Fatalf("unexpected modules view %v / %+v", modulesError, modulesView)
	}
	faxView, faxError := asteriskService.Surface(context.Background(), "fax")
	if faxError != nil || faxView.Surface.Key != "fax" {
		t.Fatalf("unexpected fax view %v / %+v", faxError, faxView)
	}
	if _, capabilitiesError := asteriskService.Surface(context.Background(), "capabilities"); capabilitiesError != nil {
		t.Fatalf("expected capabilities surface success, got %v", capabilitiesError)
	}
	if _, messagingError := asteriskService.Surface(context.Background(), "messaging"); messagingError != nil {
		t.Fatalf("expected messaging surface success, got %v", messagingError)
	}
	if _, conferencesError := asteriskService.Surface(context.Background(), "conferences"); conferencesError != nil {
		t.Fatalf("expected conferences surface success, got %v", conferencesError)
	}
	if _, queuesError := asteriskService.Surface(context.Background(), "queues"); queuesError != nil {
		t.Fatalf("expected queues surface success, got %v", queuesError)
	}
	if _, hardwareError := asteriskService.Surface(context.Background(), "hardware"); hardwareError != nil {
		t.Fatalf("expected hardware surface success, got %v", hardwareError)
	}
	browserView, browserError := asteriskService.Surface(context.Background(), "browser")
	if browserError != nil || browserView.Items[0].Key != "browser_calling" {
		t.Fatalf("unexpected browser view %v / %+v", browserError, browserView)
	}
	applyView, applyError := asteriskService.Surface(context.Background(), "apply")
	if applyError != nil || len(applyView.Items) != 4 {
		t.Fatalf("unexpected apply view %v / %+v", applyError, applyView)
	}
}

func TestAsteriskServiceErrorAndAvailabilityBranches(t *testing.T) {
	state := model.AsteriskState{
		MinimumSupportedVersion: "12",
		DetectionStatus:         "pending",
		HealthStatus:            model.AsteriskHealthDegraded,
		Capabilities: []model.AsteriskCapability{
			{Key: "prompts", Label: "Prompts", Family: "media", Available: true},
			{Key: "fax", Label: "Fax", Family: "fax", Available: false, Reason: "backend missing"},
		},
		Subsystems: []model.AsteriskManagedSubsystem{
			{Key: "tts_engine", Label: "TTS engine", Provider: "flite", Healthy: true},
		},
	}

	failingService := NewAsteriskService(failingAsteriskStore{err: errors.New("lookup failed")})
	if _, overviewError := failingService.Overview(context.Background()); overviewError == nil || overviewError.Error() != "lookup failed" {
		t.Fatalf("expected overview lookup failure, got %v", overviewError)
	}
	if _, surfaceError := failingService.Surface(context.Background(), "health"); surfaceError == nil || surfaceError.Error() != "lookup failed" {
		t.Fatalf("expected surface lookup failure, got %v", surfaceError)
	}

	asteriskService := NewAsteriskService(failingAsteriskStore{state: state})
	if domainError := (*AsteriskError)(nil).Error(); domainError != "" || (&AsteriskError{Code: "X", Message: "boom"}).Error() != "boom" {
		t.Fatalf("expected asterisk error string helpers to work")
	}
	if _, surfaceError := asteriskService.Surface(context.Background(), "missing"); surfaceError == nil || surfaceError.(*AsteriskError).Code != "ASTERISK_SURFACE_NOT_FOUND" {
		t.Fatalf("expected missing surface error, got %v", surfaceError)
	}
	if _, surfaceError := asteriskService.Surface(context.Background(), "fax"); surfaceError == nil || surfaceError.(*AsteriskError).Code != "ASTERISK_SURFACE_NOT_FOUND" {
		t.Fatalf("expected unavailable fax surface error, got %v", surfaceError)
	}
	for _, surfaceKey := range []string{"messaging", "conferences", "queues", "hardware", "browser"} {
		if _, surfaceError := asteriskService.Surface(context.Background(), surfaceKey); surfaceError == nil || surfaceError.(*AsteriskError).Code != "ASTERISK_SURFACE_NOT_FOUND" {
			t.Fatalf("expected unavailable %s surface error, got %v", surfaceKey, surfaceError)
		}
	}
	mediaView, mediaError := asteriskService.Surface(context.Background(), "media")
	if mediaError != nil || mediaView.Surface.Key != "media" {
		t.Fatalf("expected media surface success, got %v / %+v", mediaError, mediaView)
	}
	if _, mediaError := NewAsteriskService(failingAsteriskStore{state: model.AsteriskState{MinimumSupportedVersion: "12", DetectionStatus: "pending", HealthStatus: model.AsteriskHealthUnknown}}).Surface(context.Background(), "media"); mediaError == nil || mediaError.(*AsteriskError).Code != "ASTERISK_SURFACE_NOT_FOUND" {
		t.Fatalf("expected unavailable media surface error, got %v", mediaError)
	}
	if overviewView, overviewError := asteriskService.Surface(context.Background(), ""); overviewError != nil || overviewView.Surface.Key != "overview" {
		t.Fatalf("expected overview alias success, got %v / %+v", overviewError, overviewView)
	}
	healthView, healthError := asteriskService.Surface(context.Background(), "health")
	if healthError != nil || healthView.HealthStatus != model.AsteriskHealthDegraded {
		t.Fatalf("expected health surface success, got %v / %+v", healthError, healthView)
	}
	if normalizeAsteriskSurfaceKey(" /FAX/ ") != "fax" || valueOrUnknown("") != "unknown" {
		t.Fatalf("expected asterisk helper normalization")
	}
	if items := listItems("codec", "Codec", nil); len(items) != 1 || items[0].Status != "pending" {
		t.Fatalf("expected empty module list fallback, got %+v", items)
	}
	if items := subsystemItem(state.Subsystems, "tts_engine", "TTS"); len(items) != 1 || items[0].Value != "flite" {
		t.Fatalf("expected subsystem item lookup, got %+v", items)
	}
	if items := subsystemItems(state.Subsystems); len(items) != 1 || items[0].Status != "ready" {
		t.Fatalf("expected subsystem list conversion, got %+v", items)
	}
	if items := capabilityItems(state.Capabilities); len(items) != 2 || items[1].Status != "unavailable" {
		t.Fatalf("expected capability item conversion, got %+v", items)
	}
	if items := filterCapabilityItems(state.Capabilities, "fax"); len(items) != 1 || items[0].Key != "fax" {
		t.Fatalf("expected filtered capability items, got %+v", items)
	}
	if !hasAnyCapability(model.AsteriskState{Capabilities: []model.AsteriskCapability{{Key: "prompts", Available: true}}}, "prompts") {
		t.Fatalf("expected capability helper success")
	}
	if !hasSubsystem(model.AsteriskState{Subsystems: []model.AsteriskManagedSubsystem{{Key: "tts_engine", Provider: "flite"}}}, "tts_engine") {
		t.Fatalf("expected subsystem helper success")
	}
	if hasAnyCapability(model.AsteriskState{}, "prompts") || hasSubsystem(model.AsteriskState{}, "tts_engine") {
		t.Fatalf("expected helper misses")
	}
}
