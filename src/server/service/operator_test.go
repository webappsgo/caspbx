package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/casapps/caspbx/src/server/model"
	"github.com/casapps/caspbx/src/server/store"
)

func TestOperatorServiceSurfacesAndPreview(t *testing.T) {
	memoryStore := seededOperatorStore(t)
	serviceValue := NewOperatorService(memoryStore, memoryStore, memoryStore)
	if serviceValue.now().IsZero() {
		t.Fatalf("expected default now function")
	}

	dashboard, err := serviceValue.Dashboard(context.Background())
	if err != nil || dashboard.QueueCount != 1 || len(dashboard.VisibleSurfaces) < 4 {
		t.Fatalf("expected operator dashboard, got %v / %+v", err, dashboard)
	}

	queues, err := serviceValue.Queues(context.Background())
	if err != nil || len(queues.Queues) != 1 {
		t.Fatalf("expected queue wallboard, got %v / %+v", err, queues)
	}
	agents, err := serviceValue.Agents(context.Background())
	if err != nil || len(agents.Agents) != 1 {
		t.Fatalf("expected agent wallboard, got %v / %+v", err, agents)
	}
	trunks, err := serviceValue.Trunks(context.Background())
	if err != nil || len(trunks.Trunks) != 1 {
		t.Fatalf("expected trunk wallboard, got %v / %+v", err, trunks)
	}
	conferences, err := serviceValue.Conferences(context.Background())
	if err != nil || len(conferences.Conferences) != 1 {
		t.Fatalf("expected conference wallboard, got %v / %+v", err, conferences)
	}
	parking, err := serviceValue.Parking(context.Background())
	if err != nil || len(parking.ParkedCalls) != 1 {
		t.Fatalf("expected parking wallboard, got %v / %+v", err, parking)
	}
	actions, err := serviceValue.SupervisorActions(context.Background())
	if err != nil || len(actions) < 10 {
		t.Fatalf("expected supervisor actions, got %v / %+v", err, actions)
	}

	preview, err := serviceValue.PreviewSupervisorAction(context.Background(), SupervisorActionRequest{
		Action:     "spy",
		TargetKind: "queue",
		TargetRef:  "Support",
	})
	if err != nil || !preview.Available || preview.RequiredScope != "supervisor" {
		t.Fatalf("expected preview success, got %v / %+v", err, preview)
	}

	preview, err = serviceValue.PreviewSupervisorAction(context.Background(), SupervisorActionRequest{
		Action:     "pickup",
		TargetKind: "call",
		TargetRef:  "1",
	})
	if err != nil || !preview.Available || preview.RequiredScope != "operator" {
		t.Fatalf("expected telephony preview success, got %v / %+v", err, preview)
	}
}

func TestOperatorServiceErrorsAndHelpers(t *testing.T) {
	memoryStore := seededOperatorStore(t)
	serviceValue := NewOperatorService(memoryStore, memoryStore, memoryStore)

	if _, err := serviceValue.PreviewSupervisorAction(context.Background(), SupervisorActionRequest{}); err == nil {
		t.Fatalf("expected invalid preview request")
	}
	if _, err := serviceValue.PreviewSupervisorAction(context.Background(), SupervisorActionRequest{
		Action:     "unsupported",
		TargetKind: "queue",
		TargetRef:  "Support",
	}); err == nil {
		t.Fatalf("expected unsupported action error")
	}

	preview, err := serviceValue.PreviewSupervisorAction(context.Background(), SupervisorActionRequest{
		Action:     "spy",
		TargetKind: "queue",
		TargetRef:  "Missing",
	})
	if err != nil || preview.Available || len(preview.Validations) == 0 {
		t.Fatalf("expected unavailable queue preview, got %v / %+v", err, preview)
	}

	preview, err = serviceValue.PreviewSupervisorAction(context.Background(), SupervisorActionRequest{
		Action:     "queue_agent_pause",
		TargetKind: "agent",
		TargetRef:  "Missing",
	})
	if err != nil || preview.Available {
		t.Fatalf("expected unavailable agent preview, got %v / %+v", err, preview)
	}

	preview, err = serviceValue.PreviewSupervisorAction(context.Background(), SupervisorActionRequest{
		Action:     "pickup",
		TargetKind: "call",
		TargetRef:  "99",
	})
	if err != nil || preview.Available {
		t.Fatalf("expected unavailable call preview, got %v / %+v", err, preview)
	}

	preview, err = serviceValue.PreviewSupervisorAction(context.Background(), SupervisorActionRequest{
		Action:     "force_recording",
		TargetKind: "conference",
		TargetRef:  "Missing",
	})
	if err != nil || preview.Available {
		t.Fatalf("expected unavailable conference preview, got %v / %+v", err, preview)
	}

	noCapabilityStore := store.NewMemoryStore()
	if _, err := noCapabilityStore.SavePBXPlan(context.Background(), model.PBXPlan{}); err != nil {
		t.Fatalf("save plan: %v", err)
	}
	if _, err := noCapabilityStore.SaveOperatorRuntimeState(context.Background(), model.OperatorRuntimeState{}); err != nil {
		t.Fatalf("save operator state: %v", err)
	}
	if _, err := noCapabilityStore.SaveAsteriskState(context.Background(), model.AsteriskState{}); err != nil {
		t.Fatalf("save asterisk state: %v", err)
	}
	noCapabilityService := NewOperatorService(noCapabilityStore, noCapabilityStore, noCapabilityStore)
	if _, err := noCapabilityService.Queues(context.Background()); err == nil {
		t.Fatalf("expected unavailable queues")
	}
	if _, err := noCapabilityService.Agents(context.Background()); err == nil {
		t.Fatalf("expected unavailable agents")
	}
	if _, err := noCapabilityService.Conferences(context.Background()); err == nil {
		t.Fatalf("expected unavailable conferences")
	}
	if _, err := noCapabilityService.Trunks(context.Background()); err == nil {
		t.Fatalf("expected unavailable trunks")
	}
	if _, err := noCapabilityService.Parking(context.Background()); err == nil {
		t.Fatalf("expected unavailable parking")
	}
	if _, err := noCapabilityService.SupervisorActions(context.Background()); err == nil {
		t.Fatalf("expected unavailable supervisor actions")
	}
	if _, err := noCapabilityService.PreviewSupervisorAction(context.Background(), SupervisorActionRequest{
		Action:     "spy",
		TargetKind: "queue",
		TargetRef:  "Support",
	}); err == nil {
		t.Fatalf("expected unavailable preview surface")
	}
	if _, err := noCapabilityService.Dashboard(context.Background()); err == nil {
		t.Fatalf("expected unavailable dashboard")
	}

	if !supportsOperatorSurface(model.AsteriskState{ChannelDrivers: []string{"pjsip"}}, model.OperatorRuntimeState{}, model.PBXPlan{}) {
		t.Fatalf("expected operator support from telephony")
	}
	if !supportsCallCenterSurface(model.AsteriskState{Capabilities: []model.AsteriskCapability{{Key: "queues", Available: true}}}, model.PBXPlan{}) {
		t.Fatalf("expected callcenter support from queue capability")
	}
	if len(visibleOperatorSurfaces(model.AsteriskState{Capabilities: []model.AsteriskCapability{{Key: "queues", Available: true}, {Key: "conferences", Available: true}}}, model.OperatorRuntimeState{}, model.PBXPlan{})) < 5 {
		t.Fatalf("expected visible operator surfaces")
	}
	if operatorRequiredScope("spy") != "supervisor" || operatorRequiredScope("pickup") != "operator" {
		t.Fatalf("expected required scopes")
	}
	if !operatorActionExists("pickup") || operatorActionExists("missing") {
		t.Fatalf("expected action existence helper behavior")
	}
	if !operatorRequiresTelephony("pickup") || operatorRequiresTelephony("spy") {
		t.Fatalf("expected telephony helper behavior")
	}
	if !operatorRequiresQueueCapability("spy") || operatorRequiresQueueCapability("pickup") {
		t.Fatalf("expected queue helper behavior")
	}
	if !queueExists(model.PBXPlan{Queues: []model.Queue{{Name: "Support"}}}, model.OperatorRuntimeState{}, "Support") {
		t.Fatalf("expected queue existence from plan")
	}
	if !queueExists(model.PBXPlan{}, model.OperatorRuntimeState{Queues: []model.OperatorQueueState{{Name: "Support"}}}, "Support") {
		t.Fatalf("expected queue existence from runtime state")
	}
	if !agentExists(model.OperatorRuntimeState{Agents: []model.OperatorAgentState{{DisplayName: "Alice", ExtensionNumber: "1000"}}}, "1000") {
		t.Fatalf("expected agent existence")
	}
	if !agentExists(model.OperatorRuntimeState{Agents: []model.OperatorAgentState{{DisplayName: "Alice"}}}, "Alice") {
		t.Fatalf("expected agent existence by display name")
	}
	if !callExists(model.OperatorRuntimeState{ActiveCalls: []model.OperatorActiveCall{{ID: 7}}}, "7") {
		t.Fatalf("expected call existence")
	}
	if !conferenceExists(model.OperatorRuntimeState{Conferences: []model.OperatorConferenceState{{Name: "Daily"}}}, "Daily") {
		t.Fatalf("expected conference existence")
	}
	if (*OperatorError)(nil).Error() != "" || operatorUnavailable("operator").Error() == "" || operatorInvalid("bad").Error() == "" {
		t.Fatalf("expected operator error helpers")
	}
	if formatOptionalOperatorTime(time.Time{}) != "" {
		t.Fatalf("expected empty optional time formatting for zero time")
	}
}

func TestOperatorServiceStoreFailures(t *testing.T) {
	serviceValue := NewOperatorService(failingOperatorStore{err: errors.New("operator failure")}, failingUserCommunicationPBXStore{err: errors.New("pbx failure")}, failingUserCommunicationAsteriskStore{err: errors.New("asterisk failure")})
	if _, err := serviceValue.Dashboard(context.Background()); err == nil {
		t.Fatalf("expected operator store failure")
	}
	if _, err := serviceValue.Queues(context.Background()); err == nil {
		t.Fatalf("expected operator store failure for queues")
	}
	if _, err := serviceValue.Agents(context.Background()); err == nil {
		t.Fatalf("expected operator store failure for agents")
	}
	if _, err := serviceValue.Trunks(context.Background()); err == nil {
		t.Fatalf("expected operator store failure for trunks")
	}
	if _, err := serviceValue.Conferences(context.Background()); err == nil {
		t.Fatalf("expected operator store failure for conferences")
	}
	if _, err := serviceValue.Parking(context.Background()); err == nil {
		t.Fatalf("expected operator store failure for parking")
	}
	if _, err := serviceValue.SupervisorActions(context.Background()); err == nil {
		t.Fatalf("expected operator store failure for supervisor actions")
	}
	if _, err := serviceValue.PreviewSupervisorAction(context.Background(), SupervisorActionRequest{
		Action:     "spy",
		TargetKind: "queue",
		TargetRef:  "Support",
	}); err == nil {
		t.Fatalf("expected operator store failure for preview")
	}

	memoryStore := store.NewMemoryStore()
	serviceValue = NewOperatorService(memoryStore, failingUserCommunicationPBXStore{err: errors.New("pbx failure")}, memoryStore)
	if _, err := serviceValue.Dashboard(context.Background()); err == nil {
		t.Fatalf("expected pbx store failure")
	}
	serviceValue = NewOperatorService(memoryStore, memoryStore, failingUserCommunicationAsteriskStore{err: errors.New("asterisk failure")})
	if _, err := serviceValue.Dashboard(context.Background()); err == nil {
		t.Fatalf("expected asterisk store failure")
	}
}

func TestOperatorServiceAdditionalBranches(t *testing.T) {
	memoryStore := store.NewMemoryStore()
	if _, err := memoryStore.SavePBXPlan(context.Background(), model.PBXPlan{
		Queues:      []model.Queue{{Name: "Support"}},
		Trunks:      []model.Trunk{{Name: "carrier", Technology: "pjsip", Endpoint: "carrier"}},
		Conferences: []model.Conference{{Name: "Daily"}},
	}); err != nil {
		t.Fatalf("save pbx plan: %v", err)
	}
	if _, err := memoryStore.SaveAsteriskState(context.Background(), model.AsteriskState{
		MinimumSupportedVersion: "12",
		DetectionStatus:         "detected",
		HealthStatus:            model.AsteriskHealthReady,
		Capabilities: []model.AsteriskCapability{
			{Key: "queues", Available: true},
			{Key: "conferences", Available: true},
		},
	}); err != nil {
		t.Fatalf("save asterisk state: %v", err)
	}
	if _, err := memoryStore.SaveOperatorRuntimeState(context.Background(), model.OperatorRuntimeState{
		Queues:      []model.OperatorQueueState{},
		Agents:      []model.OperatorAgentState{},
		Trunks:      []model.OperatorTrunkState{},
		Conferences: []model.OperatorConferenceState{},
		ParkedCalls: []model.ParkedCallState{},
		ActiveCalls: []model.OperatorActiveCall{},
	}); err != nil {
		t.Fatalf("save operator state: %v", err)
	}
	serviceValue := NewOperatorService(memoryStore, memoryStore, memoryStore)

	dashboard, err := serviceValue.Dashboard(context.Background())
	if err != nil || len(dashboard.Warnings) != 2 {
		t.Fatalf("expected dashboard warnings, got %v / %+v", err, dashboard)
	}
	queues, err := serviceValue.Queues(context.Background())
	if err != nil || len(queues.Warnings) != 1 {
		t.Fatalf("expected queue warning, got %v / %+v", err, queues)
	}
	agents, err := serviceValue.Agents(context.Background())
	if err != nil || len(agents.Agents) != 0 {
		t.Fatalf("expected empty agent wallboard, got %v / %+v", err, agents)
	}
	trunks, err := serviceValue.Trunks(context.Background())
	if err != nil || len(trunks.Trunks) != 0 {
		t.Fatalf("expected empty trunk wallboard, got %v / %+v", err, trunks)
	}
	conferences, err := serviceValue.Conferences(context.Background())
	if err != nil || len(conferences.Conferences) != 0 {
		t.Fatalf("expected empty conference wallboard, got %v / %+v", err, conferences)
	}
	parking, err := serviceValue.Parking(context.Background())
	if err != nil || len(parking.ParkedCalls) != 0 {
		t.Fatalf("expected empty parking wallboard, got %v / %+v", err, parking)
	}

	preview, err := serviceValue.PreviewSupervisorAction(context.Background(), SupervisorActionRequest{
		Action:     "pickup",
		TargetKind: "call",
		TargetRef:  "1",
	})
	if err != nil || preview.Available || len(preview.Validations) == 0 {
		t.Fatalf("expected telephony-unavailable preview, got %v / %+v", err, preview)
	}

	if _, err := memoryStore.SaveAsteriskState(context.Background(), model.AsteriskState{
		MinimumSupportedVersion: "12",
		DetectionStatus:         "detected",
		HealthStatus:            model.AsteriskHealthReady,
		ChannelDrivers:          []string{"pjsip"},
		EndpointStacks:          []string{"pjsip"},
		Capabilities: []model.AsteriskCapability{
			{Key: "conferences", Available: true},
		},
	}); err != nil {
		t.Fatalf("save telephony state: %v", err)
	}
	preview, err = serviceValue.PreviewSupervisorAction(context.Background(), SupervisorActionRequest{
		Action:     "spy",
		TargetKind: "queue",
		TargetRef:  "Support",
	})
	if err != nil || preview.Available {
		t.Fatalf("expected queue-capability-unavailable preview, got %v / %+v", err, preview)
	}

	preview, err = serviceValue.PreviewSupervisorAction(context.Background(), SupervisorActionRequest{
		Action:     "force_recording",
		TargetKind: "conference",
		TargetRef:  "Daily",
	})
	if err != nil || preview.Available {
		t.Fatalf("expected force_recording unavailable without recording capability, got %v / %+v", err, preview)
	}

	noConferenceStore := seededOperatorStore(t)
	if _, err := noConferenceStore.SaveAsteriskState(context.Background(), model.AsteriskState{
		MinimumSupportedVersion: "12",
		DetectionStatus:         "detected",
		HealthStatus:            model.AsteriskHealthReady,
		ChannelDrivers:          []string{"pjsip"},
		EndpointStacks:          []string{"pjsip"},
		Capabilities: []model.AsteriskCapability{
			{Key: "queues", Available: true},
			{Key: "recordings", Available: true},
		},
	}); err != nil {
		t.Fatalf("save no-conference state: %v", err)
	}
	noConferenceService := NewOperatorService(noConferenceStore, noConferenceStore, noConferenceStore)
	if _, err := noConferenceService.Conferences(context.Background()); err == nil {
		t.Fatalf("expected unavailable conferences branch")
	}
}

func seededOperatorStore(t *testing.T) *store.MemoryStore {
	t.Helper()
	memoryStore := store.NewMemoryStore()
	if _, err := memoryStore.SavePBXPlan(context.Background(), model.PBXPlan{
		Queues:      []model.Queue{{Name: "Support"}},
		Trunks:      []model.Trunk{{Name: "carrier", Technology: "pjsip", Endpoint: "carrier"}},
		Conferences: []model.Conference{{Name: "Daily"}},
	}); err != nil {
		t.Fatalf("save pbx plan: %v", err)
	}
	if _, err := memoryStore.SaveAsteriskState(context.Background(), model.AsteriskState{
		MinimumSupportedVersion: "12",
		DetectionStatus:         "detected",
		HealthStatus:            model.AsteriskHealthReady,
		ChannelDrivers:          []string{"pjsip"},
		EndpointStacks:          []string{"pjsip"},
		Capabilities: []model.AsteriskCapability{
			{Key: "queues", Available: true},
			{Key: "conferences", Available: true},
			{Key: "recordings", Available: true},
		},
	}); err != nil {
		t.Fatalf("save asterisk state: %v", err)
	}
	if _, err := memoryStore.SaveOperatorRuntimeState(context.Background(), model.OperatorRuntimeState{
		Queues:      []model.OperatorQueueState{{Name: "Support", WaitingCalls: 3, ActiveCalls: 2, AvailableAgents: 4, UpdatedAt: time.Unix(10, 0)}},
		Agents:      []model.OperatorAgentState{{ID: 1, QueueName: "Support", DisplayName: "Alice", ExtensionNumber: "1000", Status: "ready", LastChangeAt: time.Unix(11, 0)}},
		Trunks:      []model.OperatorTrunkState{{Name: "carrier", Technology: "pjsip", Registered: true, ActiveCalls: 2, Healthy: true, UpdatedAt: time.Unix(12, 0)}},
		Conferences: []model.OperatorConferenceState{{Name: "Daily", ParticipantCount: 4, Recording: true, UpdatedAt: time.Unix(13, 0)}},
		ParkedCalls: []model.ParkedCallState{{Slot: "701", Caller: "1002", DurationSeconds: 20, UpdatedAt: time.Unix(14, 0)}},
		ActiveCalls: []model.OperatorActiveCall{{ID: 1, Direction: "inbound", Source: "1002", Destination: "1000", QueueName: "Support", AgentExtension: "1000", DurationSeconds: 40, UpdatedAt: time.Unix(15, 0)}},
		UpdatedAt:   time.Unix(16, 0),
	}); err != nil {
		t.Fatalf("save operator runtime state: %v", err)
	}
	return memoryStore
}

type failingOperatorStore struct{ err error }

func (storeValue failingOperatorStore) SaveOperatorRuntimeState(context.Context, model.OperatorRuntimeState) (model.OperatorRuntimeState, error) {
	return model.OperatorRuntimeState{}, storeValue.err
}

func (storeValue failingOperatorStore) GetOperatorRuntimeState(context.Context) (model.OperatorRuntimeState, error) {
	return model.OperatorRuntimeState{}, storeValue.err
}
