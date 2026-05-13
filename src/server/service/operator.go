package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/casapps/caspbx/src/server/model"
	"github.com/casapps/caspbx/src/server/store"
)

type OperatorError struct {
	Code    string
	Message string
}

type OperatorSurface struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Path  string `json:"path"`
}

type OperatorDashboard struct {
	UpdatedAt       string            `json:"updated_at,omitempty"`
	QueueCount      int               `json:"queue_count"`
	AgentCount      int               `json:"agent_count"`
	TrunkCount      int               `json:"trunk_count"`
	ConferenceCount int               `json:"conference_count"`
	ParkedCallCount int               `json:"parked_call_count"`
	ActiveCallCount int               `json:"active_call_count"`
	VisibleSurfaces []OperatorSurface `json:"visible_surfaces"`
	Warnings        []string          `json:"warnings,omitempty"`
}

type QueueWallboard struct {
	UpdatedAt string                     `json:"updated_at,omitempty"`
	Queues    []model.OperatorQueueState `json:"queues"`
	Warnings  []string                   `json:"warnings,omitempty"`
}

type AgentWallboard struct {
	UpdatedAt string                     `json:"updated_at,omitempty"`
	Agents    []model.OperatorAgentState `json:"agents"`
}

type TrunkWallboard struct {
	UpdatedAt string                     `json:"updated_at,omitempty"`
	Trunks    []model.OperatorTrunkState `json:"trunks"`
}

type ConferenceWallboard struct {
	UpdatedAt   string                          `json:"updated_at,omitempty"`
	Conferences []model.OperatorConferenceState `json:"conferences"`
}

type ParkingWallboard struct {
	UpdatedAt   string                  `json:"updated_at,omitempty"`
	ParkedCalls []model.ParkedCallState `json:"parked_calls"`
}

type SupervisorAction struct {
	Key           string   `json:"key"`
	Label         string   `json:"label"`
	RequiredScope string   `json:"required_scope"`
	Capabilities  []string `json:"capabilities,omitempty"`
}

type SupervisorActionPreview struct {
	Action        string   `json:"action"`
	TargetKind    string   `json:"target_kind"`
	TargetRef     string   `json:"target_ref"`
	RequiredScope string   `json:"required_scope"`
	Available     bool     `json:"available"`
	Validations   []string `json:"validations,omitempty"`
}

type SupervisorActionRequest struct {
	Action     string `json:"action"`
	TargetKind string `json:"target_kind"`
	TargetRef  string `json:"target_ref"`
}

type OperatorService struct {
	operatorStore store.OperatorStore
	pbxStore      store.PBXStore
	asteriskStore store.AsteriskStore
	now           func() time.Time
}

func (errorValue *OperatorError) Error() string {
	if errorValue == nil {
		return ""
	}
	return errorValue.Message
}

func NewOperatorService(operatorStore store.OperatorStore, pbxStore store.PBXStore, asteriskStore store.AsteriskStore) OperatorService {
	return OperatorService{
		operatorStore: operatorStore,
		pbxStore:      pbxStore,
		asteriskStore: asteriskStore,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (service OperatorService) Dashboard(ctx context.Context) (OperatorDashboard, error) {
	runtimeState, plan, asteriskState, err := service.loadState(ctx)
	if err != nil {
		return OperatorDashboard{}, err
	}
	if !supportsOperatorSurface(asteriskState, runtimeState, plan) {
		return OperatorDashboard{}, operatorUnavailable("operator")
	}
	dashboard := OperatorDashboard{
		UpdatedAt:       formatOptionalOperatorTime(runtimeState.UpdatedAt),
		QueueCount:      len(runtimeState.Queues),
		AgentCount:      len(runtimeState.Agents),
		TrunkCount:      len(runtimeState.Trunks),
		ConferenceCount: len(runtimeState.Conferences),
		ParkedCallCount: len(runtimeState.ParkedCalls),
		ActiveCallCount: len(runtimeState.ActiveCalls),
		VisibleSurfaces: visibleOperatorSurfaces(asteriskState, runtimeState, plan),
	}
	if len(runtimeState.ActiveCalls) == 0 {
		dashboard.Warnings = append(dashboard.Warnings, "No live calls are currently visible to the switchboard.")
	}
	if len(runtimeState.ParkedCalls) == 0 {
		dashboard.Warnings = append(dashboard.Warnings, "No parked calls are currently active.")
	}
	return dashboard, nil
}

func (service OperatorService) Queues(ctx context.Context) (QueueWallboard, error) {
	runtimeState, plan, asteriskState, err := service.loadState(ctx)
	if err != nil {
		return QueueWallboard{}, err
	}
	if !supportsCallCenterSurface(asteriskState, plan) {
		return QueueWallboard{}, operatorUnavailable("callcenter")
	}
	wallboard := QueueWallboard{
		UpdatedAt: formatOptionalOperatorTime(runtimeState.UpdatedAt),
		Queues:    runtimeState.Queues,
	}
	if len(runtimeState.Queues) == 0 {
		wallboard.Warnings = append(wallboard.Warnings, "No queue runtime state is currently available.")
	}
	return wallboard, nil
}

func (service OperatorService) Agents(ctx context.Context) (AgentWallboard, error) {
	runtimeState, plan, asteriskState, err := service.loadState(ctx)
	if err != nil {
		return AgentWallboard{}, err
	}
	if !supportsCallCenterSurface(asteriskState, plan) {
		return AgentWallboard{}, operatorUnavailable("callcenter")
	}
	return AgentWallboard{
		UpdatedAt: formatOptionalOperatorTime(runtimeState.UpdatedAt),
		Agents:    runtimeState.Agents,
	}, nil
}

func (service OperatorService) Trunks(ctx context.Context) (TrunkWallboard, error) {
	runtimeState, plan, asteriskState, err := service.loadState(ctx)
	if err != nil {
		return TrunkWallboard{}, err
	}
	if !supportsOperatorSurface(asteriskState, runtimeState, plan) {
		return TrunkWallboard{}, operatorUnavailable("operator")
	}
	return TrunkWallboard{
		UpdatedAt: formatOptionalOperatorTime(runtimeState.UpdatedAt),
		Trunks:    runtimeState.Trunks,
	}, nil
}

func (service OperatorService) Conferences(ctx context.Context) (ConferenceWallboard, error) {
	runtimeState, _, asteriskState, err := service.loadState(ctx)
	if err != nil {
		return ConferenceWallboard{}, err
	}
	if !hasAvailableCapability(asteriskState, "conferences") {
		return ConferenceWallboard{}, operatorUnavailable("conferences")
	}
	return ConferenceWallboard{
		UpdatedAt:   formatOptionalOperatorTime(runtimeState.UpdatedAt),
		Conferences: runtimeState.Conferences,
	}, nil
}

func (service OperatorService) Parking(ctx context.Context) (ParkingWallboard, error) {
	runtimeState, plan, asteriskState, err := service.loadState(ctx)
	if err != nil {
		return ParkingWallboard{}, err
	}
	if !supportsOperatorSurface(asteriskState, runtimeState, plan) {
		return ParkingWallboard{}, operatorUnavailable("operator")
	}
	return ParkingWallboard{
		UpdatedAt:   formatOptionalOperatorTime(runtimeState.UpdatedAt),
		ParkedCalls: runtimeState.ParkedCalls,
	}, nil
}

func (service OperatorService) SupervisorActions(ctx context.Context) ([]SupervisorAction, error) {
	_, plan, asteriskState, err := service.loadState(ctx)
	if err != nil {
		return nil, err
	}
	if !supportsCallCenterSurface(asteriskState, plan) {
		return nil, operatorUnavailable("callcenter")
	}
	actions := []SupervisorAction{
		{Key: "pickup", Label: "Pickup", RequiredScope: "operator", Capabilities: []string{"telephony"}},
		{Key: "blind_transfer", Label: "Blind transfer", RequiredScope: "operator", Capabilities: []string{"telephony"}},
		{Key: "attended_transfer", Label: "Attended transfer", RequiredScope: "operator", Capabilities: []string{"telephony"}},
		{Key: "voicemail_transfer", Label: "Voicemail transfer", RequiredScope: "operator", Capabilities: []string{"telephony", "voicemail"}},
		{Key: "hangup", Label: "Hangup", RequiredScope: "operator", Capabilities: []string{"telephony"}},
		{Key: "park", Label: "Park", RequiredScope: "operator", Capabilities: []string{"telephony"}},
		{Key: "spy", Label: "Spy", RequiredScope: "supervisor", Capabilities: []string{"queues"}},
		{Key: "whisper", Label: "Whisper", RequiredScope: "supervisor", Capabilities: []string{"queues"}},
		{Key: "barge", Label: "Barge", RequiredScope: "supervisor", Capabilities: []string{"queues"}},
		{Key: "force_recording", Label: "Force recording", RequiredScope: "supervisor", Capabilities: []string{"queues", "recordings"}},
		{Key: "queue_agent_pause", Label: "Pause queue agent", RequiredScope: "supervisor", Capabilities: []string{"queues"}},
		{Key: "queue_agent_unpause", Label: "Unpause queue agent", RequiredScope: "supervisor", Capabilities: []string{"queues"}},
	}
	return actions, nil
}

func (service OperatorService) PreviewSupervisorAction(ctx context.Context, request SupervisorActionRequest) (SupervisorActionPreview, error) {
	runtimeState, plan, asteriskState, err := service.loadState(ctx)
	if err != nil {
		return SupervisorActionPreview{}, err
	}
	if !supportsCallCenterSurface(asteriskState, plan) {
		return SupervisorActionPreview{}, operatorUnavailable("callcenter")
	}
	action := strings.TrimSpace(strings.ToLower(request.Action))
	targetKind := strings.TrimSpace(strings.ToLower(request.TargetKind))
	targetRef := strings.TrimSpace(request.TargetRef)
	if action == "" || targetKind == "" || targetRef == "" {
		return SupervisorActionPreview{}, operatorInvalid("action, target_kind, and target_ref are required")
	}
	preview := SupervisorActionPreview{
		Action:        action,
		TargetKind:    targetKind,
		TargetRef:     targetRef,
		Available:     true,
		RequiredScope: operatorRequiredScope(action),
		Validations:   []string{},
	}
	if !operatorActionExists(action) {
		return SupervisorActionPreview{}, operatorInvalid("unsupported operator action")
	}
	if operatorRequiresTelephony(action) && !supportsTelephony(asteriskState) {
		preview.Available = false
		preview.Validations = append(preview.Validations, "Telephony drivers are not currently available.")
	}
	if operatorRequiresQueueCapability(action) && !hasAvailableCapability(asteriskState, "queues") {
		preview.Available = false
		preview.Validations = append(preview.Validations, "Queue capability is required for this action.")
	}
	if action == "force_recording" && !hasAnyAvailableCapability(asteriskState, "recordings", "voicemail") {
		preview.Available = false
		preview.Validations = append(preview.Validations, "Recording capability is required for force_recording.")
	}
	if targetKind == "queue" && !queueExists(plan, runtimeState, targetRef) {
		preview.Available = false
		preview.Validations = append(preview.Validations, "Referenced queue is not defined.")
	}
	if targetKind == "agent" && !agentExists(runtimeState, targetRef) {
		preview.Available = false
		preview.Validations = append(preview.Validations, "Referenced agent is not visible in runtime state.")
	}
	if targetKind == "call" && !callExists(runtimeState, targetRef) {
		preview.Available = false
		preview.Validations = append(preview.Validations, "Referenced call is not visible in runtime state.")
	}
	if targetKind == "conference" && !conferenceExists(runtimeState, targetRef) {
		preview.Available = false
		preview.Validations = append(preview.Validations, "Referenced conference is not visible in runtime state.")
	}
	if len(preview.Validations) == 0 {
		preview.Validations = append(preview.Validations, "Action prerequisites satisfied for preview.")
	}
	return preview, nil
}

func (service OperatorService) loadState(ctx context.Context) (model.OperatorRuntimeState, model.PBXPlan, model.AsteriskState, error) {
	runtimeState, err := service.operatorStore.GetOperatorRuntimeState(ctx)
	if err != nil {
		return model.OperatorRuntimeState{}, model.PBXPlan{}, model.AsteriskState{}, err
	}
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return model.OperatorRuntimeState{}, model.PBXPlan{}, model.AsteriskState{}, err
	}
	asteriskState, err := service.asteriskStore.GetAsteriskState(ctx)
	if err != nil {
		return model.OperatorRuntimeState{}, model.PBXPlan{}, model.AsteriskState{}, err
	}
	return runtimeState, plan, asteriskState, nil
}

func visibleOperatorSurfaces(asteriskState model.AsteriskState, runtimeState model.OperatorRuntimeState, plan model.PBXPlan) []OperatorSurface {
	surfaces := []OperatorSurface{{Key: "operator", Label: "Operator Dashboard", Path: "operator"}}
	if supportsCallCenterSurface(asteriskState, plan) {
		surfaces = append(surfaces,
			OperatorSurface{Key: "callcenter-queues", Label: "Queue Wallboard", Path: "callcenter/queues"},
			OperatorSurface{Key: "callcenter-agents", Label: "Agent Wallboard", Path: "callcenter/agents"},
			OperatorSurface{Key: "callcenter-supervisor-actions", Label: "Supervisor Actions", Path: "callcenter/supervisor-actions"},
		)
	}
	if hasAvailableCapability(asteriskState, "conferences") {
		surfaces = append(surfaces, OperatorSurface{Key: "operator-conferences", Label: "Conference Visibility", Path: "operator/conferences"})
	}
	surfaces = append(surfaces,
		OperatorSurface{Key: "operator-trunks", Label: "Trunk Visibility", Path: "operator/trunks"},
		OperatorSurface{Key: "operator-parked-calls", Label: "Parked Calls", Path: "operator/parked-calls"},
	)
	if len(runtimeState.ActiveCalls) == 0 {
		return surfaces
	}
	return surfaces
}

func supportsOperatorSurface(asteriskState model.AsteriskState, runtimeState model.OperatorRuntimeState, plan model.PBXPlan) bool {
	return supportsTelephony(asteriskState) || supportsCallCenterSurface(asteriskState, plan) || len(runtimeState.ActiveCalls) > 0 || len(plan.Trunks) > 0 || len(plan.Conferences) > 0
}

func supportsCallCenterSurface(asteriskState model.AsteriskState, plan model.PBXPlan) bool {
	return hasAvailableCapability(asteriskState, "queues") || len(plan.Queues) > 0
}

func operatorRequiredScope(action string) string {
	if operatorRequiresQueueCapability(action) || action == "force_recording" {
		return "supervisor"
	}
	return "operator"
}

func operatorActionExists(action string) bool {
	switch action {
	case "pickup", "blind_transfer", "attended_transfer", "voicemail_transfer", "hangup", "park", "spy", "whisper", "barge", "force_recording", "queue_agent_pause", "queue_agent_unpause":
		return true
	default:
		return false
	}
}

func operatorRequiresTelephony(action string) bool {
	switch action {
	case "pickup", "blind_transfer", "attended_transfer", "voicemail_transfer", "hangup", "park":
		return true
	default:
		return false
	}
}

func operatorRequiresQueueCapability(action string) bool {
	switch action {
	case "spy", "whisper", "barge", "force_recording", "queue_agent_pause", "queue_agent_unpause":
		return true
	default:
		return false
	}
}

func queueExists(plan model.PBXPlan, runtimeState model.OperatorRuntimeState, targetRef string) bool {
	for _, queue := range plan.Queues {
		if queue.Name == targetRef {
			return true
		}
	}
	for _, queue := range runtimeState.Queues {
		if queue.Name == targetRef {
			return true
		}
	}
	return false
}

func agentExists(runtimeState model.OperatorRuntimeState, targetRef string) bool {
	for _, agent := range runtimeState.Agents {
		if agent.ExtensionNumber == targetRef || agent.DisplayName == targetRef {
			return true
		}
	}
	return false
}

func callExists(runtimeState model.OperatorRuntimeState, targetRef string) bool {
	for _, call := range runtimeState.ActiveCalls {
		if fmt.Sprintf("%d", call.ID) == targetRef {
			return true
		}
	}
	return false
}

func conferenceExists(runtimeState model.OperatorRuntimeState, targetRef string) bool {
	for _, conference := range runtimeState.Conferences {
		if conference.Name == targetRef {
			return true
		}
	}
	return false
}

func operatorUnavailable(surface string) error {
	return &OperatorError{Code: "OPERATOR_UNAVAILABLE", Message: fmt.Sprintf("%s surface is not available", surface)}
}

func operatorInvalid(message string) error {
	return &OperatorError{Code: "OPERATOR_INVALID", Message: message}
}

func formatOptionalOperatorTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
