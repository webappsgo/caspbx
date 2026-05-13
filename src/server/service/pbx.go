package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/casapps/caspbx/src/server/model"
	"github.com/casapps/caspbx/src/server/store"
)

type PBXError struct {
	Code    string
	Message string
}

type PBXResourceSummary struct {
	Resource string `json:"resource"`
	Count    int    `json:"count"`
}

type PBXApplyArtifact struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Status  string `json:"status"`
	Summary string `json:"summary"`
}

type PBXApplyPreview struct {
	UpdatedAt   string               `json:"updated_at,omitempty"`
	Summaries   []PBXResourceSummary `json:"summaries"`
	Artifacts   []PBXApplyArtifact   `json:"artifacts"`
	Validations []string             `json:"validations,omitempty"`
	Actions     []string             `json:"actions"`
}

type PBXService struct {
	pbxStore      store.PBXStore
	asteriskStore store.AsteriskStore
	now           func() time.Time
}

func (errorValue *PBXError) Error() string {
	if errorValue == nil {
		return ""
	}
	return errorValue.Message
}

func NewPBXService(pbxStore store.PBXStore, asteriskStore store.AsteriskStore) PBXService {
	return PBXService{
		pbxStore:      pbxStore,
		asteriskStore: asteriskStore,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (service PBXService) ListExtensions(ctx context.Context) ([]model.Extension, error) {
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return nil, err
	}
	return plan.Extensions, nil
}

func (service PBXService) GetExtension(ctx context.Context, id int64) (model.Extension, error) {
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return model.Extension{}, err
	}
	for _, entity := range plan.Extensions {
		if entity.ID == id {
			return entity, nil
		}
	}
	return model.Extension{}, notFoundPBXError("extension")
}

func (service PBXService) CreateExtension(ctx context.Context, entity model.Extension) (model.Extension, error) {
	if err := validateExtension(entity); err != nil {
		return model.Extension{}, err
	}
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return model.Extension{}, err
	}
	entity.ID = nextExtensionID(plan.Extensions)
	entity.Number = model.NormalizePBXField(entity.Number)
	entity.DisplayName = model.NormalizePBXField(entity.DisplayName)
	entity.Technology = strings.ToLower(model.NormalizePBXField(entity.Technology))
	entity.Endpoint = model.NormalizePBXField(entity.Endpoint)
	entity.CreatedAt = service.now()
	entity.UpdatedAt = entity.CreatedAt
	plan.Extensions = append(plan.Extensions, entity)
	if _, err = service.savePlan(ctx, plan); err != nil {
		return model.Extension{}, err
	}
	return entity, nil
}

func (service PBXService) DeleteExtension(ctx context.Context, id int64) error {
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return err
	}
	var removed bool
	plan.Extensions, removed = deleteFromSlice(plan.Extensions, func(entity model.Extension) bool { return entity.ID == id })
	if !removed {
		return notFoundPBXError("extension")
	}
	_, err = service.savePlan(ctx, plan)
	return err
}

func (service PBXService) ListTrunks(ctx context.Context) ([]model.Trunk, error) {
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return nil, err
	}
	return plan.Trunks, nil
}

func (service PBXService) GetTrunk(ctx context.Context, id int64) (model.Trunk, error) {
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return model.Trunk{}, err
	}
	for _, entity := range plan.Trunks {
		if entity.ID == id {
			return entity, nil
		}
	}
	return model.Trunk{}, notFoundPBXError("trunk")
}

func (service PBXService) CreateTrunk(ctx context.Context, entity model.Trunk) (model.Trunk, error) {
	if err := validateTrunk(entity); err != nil {
		return model.Trunk{}, err
	}
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return model.Trunk{}, err
	}
	entity.ID = nextTrunkID(plan.Trunks)
	entity.Name = model.NormalizePBXField(entity.Name)
	entity.Technology = strings.ToLower(model.NormalizePBXField(entity.Technology))
	entity.Endpoint = model.NormalizePBXField(entity.Endpoint)
	entity.CreatedAt = service.now()
	entity.UpdatedAt = entity.CreatedAt
	plan.Trunks = append(plan.Trunks, entity)
	if _, err = service.savePlan(ctx, plan); err != nil {
		return model.Trunk{}, err
	}
	return entity, nil
}

func (service PBXService) DeleteTrunk(ctx context.Context, id int64) error {
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return err
	}
	var removed bool
	plan.Trunks, removed = deleteFromSlice(plan.Trunks, func(entity model.Trunk) bool { return entity.ID == id })
	if !removed {
		return notFoundPBXError("trunk")
	}
	_, err = service.savePlan(ctx, plan)
	return err
}

func (service PBXService) ListRoutes(ctx context.Context) ([]model.CallRoute, error) {
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return nil, err
	}
	return plan.Routes, nil
}

func (service PBXService) GetRoute(ctx context.Context, id int64) (model.CallRoute, error) {
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return model.CallRoute{}, err
	}
	for _, entity := range plan.Routes {
		if entity.ID == id {
			return entity, nil
		}
	}
	return model.CallRoute{}, notFoundPBXError("route")
}

func (service PBXService) CreateRoute(ctx context.Context, entity model.CallRoute) (model.CallRoute, error) {
	if err := validateRoute(entity); err != nil {
		return model.CallRoute{}, err
	}
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return model.CallRoute{}, err
	}
	entity.ID = nextRouteID(plan.Routes)
	entity.Name = model.NormalizePBXField(entity.Name)
	entity.Direction = strings.ToLower(model.NormalizePBXField(entity.Direction))
	entity.Match = model.NormalizePBXField(entity.Match)
	entity.Destination = model.NormalizePBXField(entity.Destination)
	entity.CreatedAt = service.now()
	entity.UpdatedAt = entity.CreatedAt
	plan.Routes = append(plan.Routes, entity)
	if _, err = service.savePlan(ctx, plan); err != nil {
		return model.CallRoute{}, err
	}
	return entity, nil
}

func (service PBXService) DeleteRoute(ctx context.Context, id int64) error {
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return err
	}
	var removed bool
	plan.Routes, removed = deleteFromSlice(plan.Routes, func(entity model.CallRoute) bool { return entity.ID == id })
	if !removed {
		return notFoundPBXError("route")
	}
	_, err = service.savePlan(ctx, plan)
	return err
}

func (service PBXService) ListQueues(ctx context.Context) ([]model.Queue, error) {
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return nil, err
	}
	return plan.Queues, nil
}

func (service PBXService) GetQueue(ctx context.Context, id int64) (model.Queue, error) {
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return model.Queue{}, err
	}
	for _, entity := range plan.Queues {
		if entity.ID == id {
			return entity, nil
		}
	}
	return model.Queue{}, notFoundPBXError("queue")
}

func (service PBXService) CreateQueue(ctx context.Context, entity model.Queue) (model.Queue, error) {
	if err := validateQueue(entity); err != nil {
		return model.Queue{}, err
	}
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return model.Queue{}, err
	}
	entity.ID = nextQueueID(plan.Queues)
	entity.Name = model.NormalizePBXField(entity.Name)
	entity.Strategy = model.NormalizePBXField(entity.Strategy)
	entity.CreatedAt = service.now()
	entity.UpdatedAt = entity.CreatedAt
	plan.Queues = append(plan.Queues, entity)
	if _, err = service.savePlan(ctx, plan); err != nil {
		return model.Queue{}, err
	}
	return entity, nil
}

func (service PBXService) DeleteQueue(ctx context.Context, id int64) error {
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return err
	}
	var removed bool
	plan.Queues, removed = deleteFromSlice(plan.Queues, func(entity model.Queue) bool { return entity.ID == id })
	if !removed {
		return notFoundPBXError("queue")
	}
	_, err = service.savePlan(ctx, plan)
	return err
}

func (service PBXService) ListConferences(ctx context.Context) ([]model.Conference, error) {
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return nil, err
	}
	return plan.Conferences, nil
}

func (service PBXService) GetConference(ctx context.Context, id int64) (model.Conference, error) {
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return model.Conference{}, err
	}
	for _, entity := range plan.Conferences {
		if entity.ID == id {
			return entity, nil
		}
	}
	return model.Conference{}, notFoundPBXError("conference")
}

func (service PBXService) CreateConference(ctx context.Context, entity model.Conference) (model.Conference, error) {
	if err := validateConference(entity); err != nil {
		return model.Conference{}, err
	}
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return model.Conference{}, err
	}
	entity.ID = nextConferenceID(plan.Conferences)
	entity.Name = model.NormalizePBXField(entity.Name)
	entity.AccessCode = model.NormalizePBXField(entity.AccessCode)
	entity.CreatedAt = service.now()
	entity.UpdatedAt = entity.CreatedAt
	plan.Conferences = append(plan.Conferences, entity)
	if _, err = service.savePlan(ctx, plan); err != nil {
		return model.Conference{}, err
	}
	return entity, nil
}

func (service PBXService) DeleteConference(ctx context.Context, id int64) error {
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return err
	}
	var removed bool
	plan.Conferences, removed = deleteFromSlice(plan.Conferences, func(entity model.Conference) bool { return entity.ID == id })
	if !removed {
		return notFoundPBXError("conference")
	}
	_, err = service.savePlan(ctx, plan)
	return err
}

func (service PBXService) ListIVRs(ctx context.Context) ([]model.IVR, error) {
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return nil, err
	}
	return plan.IVRs, nil
}

func (service PBXService) GetIVR(ctx context.Context, id int64) (model.IVR, error) {
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return model.IVR{}, err
	}
	for _, entity := range plan.IVRs {
		if entity.ID == id {
			return entity, nil
		}
	}
	return model.IVR{}, notFoundPBXError("ivr")
}

func (service PBXService) CreateIVR(ctx context.Context, entity model.IVR) (model.IVR, error) {
	if err := validateIVR(entity); err != nil {
		return model.IVR{}, err
	}
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return model.IVR{}, err
	}
	entity.ID = nextIVRID(plan.IVRs)
	entity.Name = model.NormalizePBXField(entity.Name)
	entity.RootPrompt = model.NormalizePBXField(entity.RootPrompt)
	entity.DefaultDestination = model.NormalizePBXField(entity.DefaultDestination)
	entity.CreatedAt = service.now()
	entity.UpdatedAt = entity.CreatedAt
	plan.IVRs = append(plan.IVRs, entity)
	if _, err = service.savePlan(ctx, plan); err != nil {
		return model.IVR{}, err
	}
	return entity, nil
}

func (service PBXService) DeleteIVR(ctx context.Context, id int64) error {
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return err
	}
	var removed bool
	plan.IVRs, removed = deleteFromSlice(plan.IVRs, func(entity model.IVR) bool { return entity.ID == id })
	if !removed {
		return notFoundPBXError("ivr")
	}
	_, err = service.savePlan(ctx, plan)
	return err
}

func (service PBXService) ListPromptAssignments(ctx context.Context) ([]model.PromptAssignment, error) {
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return nil, err
	}
	return plan.PromptAssignments, nil
}

func (service PBXService) GetPromptAssignment(ctx context.Context, id int64) (model.PromptAssignment, error) {
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return model.PromptAssignment{}, err
	}
	for _, entity := range plan.PromptAssignments {
		if entity.ID == id {
			return entity, nil
		}
	}
	return model.PromptAssignment{}, notFoundPBXError("prompt_assignment")
}

func (service PBXService) CreatePromptAssignment(ctx context.Context, entity model.PromptAssignment) (model.PromptAssignment, error) {
	if err := validatePromptAssignment(entity); err != nil {
		return model.PromptAssignment{}, err
	}
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return model.PromptAssignment{}, err
	}
	entity.ID = nextPromptAssignmentID(plan.PromptAssignments)
	entity.Name = model.NormalizePBXField(entity.Name)
	entity.PromptName = model.NormalizePBXField(entity.PromptName)
	entity.TargetKind = strings.ToLower(model.NormalizePBXField(entity.TargetKind))
	entity.TargetRef = model.NormalizePBXField(entity.TargetRef)
	entity.CreatedAt = service.now()
	entity.UpdatedAt = entity.CreatedAt
	plan.PromptAssignments = append(plan.PromptAssignments, entity)
	if _, err = service.savePlan(ctx, plan); err != nil {
		return model.PromptAssignment{}, err
	}
	return entity, nil
}

func (service PBXService) DeletePromptAssignment(ctx context.Context, id int64) error {
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return err
	}
	var removed bool
	plan.PromptAssignments, removed = deleteFromSlice(plan.PromptAssignments, func(entity model.PromptAssignment) bool { return entity.ID == id })
	if !removed {
		return notFoundPBXError("prompt_assignment")
	}
	_, err = service.savePlan(ctx, plan)
	return err
}

func (service PBXService) ListProvisioningProfiles(ctx context.Context) ([]model.ProvisioningProfile, error) {
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return nil, err
	}
	return plan.ProvisioningProfiles, nil
}

func (service PBXService) GetProvisioningProfile(ctx context.Context, id int64) (model.ProvisioningProfile, error) {
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return model.ProvisioningProfile{}, err
	}
	for _, entity := range plan.ProvisioningProfiles {
		if entity.ID == id {
			return entity, nil
		}
	}
	return model.ProvisioningProfile{}, notFoundPBXError("provisioning_profile")
}

func (service PBXService) CreateProvisioningProfile(ctx context.Context, entity model.ProvisioningProfile) (model.ProvisioningProfile, error) {
	if err := validateProvisioningProfile(entity); err != nil {
		return model.ProvisioningProfile{}, err
	}
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return model.ProvisioningProfile{}, err
	}
	entity.ID = nextProvisioningProfileID(plan.ProvisioningProfiles)
	entity.Name = model.NormalizePBXField(entity.Name)
	entity.Technology = strings.ToLower(model.NormalizePBXField(entity.Technology))
	entity.Template = model.NormalizePBXField(entity.Template)
	entity.CreatedAt = service.now()
	entity.UpdatedAt = entity.CreatedAt
	plan.ProvisioningProfiles = append(plan.ProvisioningProfiles, entity)
	if _, err = service.savePlan(ctx, plan); err != nil {
		return model.ProvisioningProfile{}, err
	}
	return entity, nil
}

func (service PBXService) DeleteProvisioningProfile(ctx context.Context, id int64) error {
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return err
	}
	var removed bool
	plan.ProvisioningProfiles, removed = deleteFromSlice(plan.ProvisioningProfiles, func(entity model.ProvisioningProfile) bool { return entity.ID == id })
	if !removed {
		return notFoundPBXError("provisioning_profile")
	}
	_, err = service.savePlan(ctx, plan)
	return err
}

func (service PBXService) ApplyPreview(ctx context.Context) (PBXApplyPreview, error) {
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return PBXApplyPreview{}, err
	}
	state, err := service.asteriskStore.GetAsteriskState(ctx)
	if err != nil {
		return PBXApplyPreview{}, err
	}
	preview := PBXApplyPreview{
		UpdatedAt: formatPBXTime(plan.UpdatedAt),
		Summaries: []PBXResourceSummary{
			{Resource: "extensions", Count: len(plan.Extensions)},
			{Resource: "trunks", Count: len(plan.Trunks)},
			{Resource: "routes", Count: len(plan.Routes)},
			{Resource: "queues", Count: len(plan.Queues)},
			{Resource: "conferences", Count: len(plan.Conferences)},
			{Resource: "ivrs", Count: len(plan.IVRs)},
			{Resource: "prompt-assignments", Count: len(plan.PromptAssignments)},
			{Resource: "provisioning-profiles", Count: len(plan.ProvisioningProfiles)},
		},
		Artifacts: buildApplyArtifacts(plan),
		Actions: []string{
			"validate capabilities against the active Asterisk inventory",
			"render configuration and dialplan intent from persisted PBX objects",
			"show apply diff before reload or restart actions",
			"run post-apply health checks after backend activation",
		},
	}
	if len(plan.Queues) > 0 && !hasAnyCapability(state, "queues") {
		preview.Validations = append(preview.Validations, "queue objects exist but queue capability is unavailable")
	}
	if len(plan.Conferences) > 0 && !hasAnyCapability(state, "conferences") {
		preview.Validations = append(preview.Validations, "conference objects exist but conference capability is unavailable")
	}
	if (len(plan.IVRs) > 0 || len(plan.PromptAssignments) > 0) && !hasAnyCapability(state, "prompts", "recordings") {
		preview.Validations = append(preview.Validations, "ivr or prompt assignments exist but prompt/media capabilities are unavailable")
	}
	if (len(plan.Extensions) > 0 || len(plan.Trunks) > 0 || len(plan.ProvisioningProfiles) > 0) && len(state.ChannelDrivers) == 0 {
		preview.Validations = append(preview.Validations, "telephony objects exist but no channel drivers were detected")
	}
	for _, technology := range referencedTechnologies(plan) {
		if !listContainsFold(state.ChannelDrivers, technology) && !listContainsFold(state.EndpointStacks, technology) {
			preview.Validations = append(preview.Validations, fmt.Sprintf("%s is referenced by the PBX plan but not present in detected drivers or endpoint stacks", technology))
		}
	}
	return preview, nil
}

func (service PBXService) savePlan(ctx context.Context, plan model.PBXPlan) (model.PBXPlan, error) {
	plan = model.NormalizePBXPlan(plan)
	plan.UpdatedAt = service.now()
	model.SortPBXPlan(&plan)
	return service.pbxStore.SavePBXPlan(ctx, plan)
}

func validateExtension(entity model.Extension) error {
	if err := model.ValidatePBXField(entity.Number); err != nil {
		return invalidPBXError("extension.number")
	}
	if err := model.ValidatePBXField(entity.DisplayName); err != nil {
		return invalidPBXError("extension.display_name")
	}
	if err := model.ValidatePBXField(entity.Technology); err != nil {
		return invalidPBXError("extension.technology")
	}
	return nil
}

func validateTrunk(entity model.Trunk) error {
	if err := model.ValidatePBXField(entity.Name); err != nil {
		return invalidPBXError("trunk.name")
	}
	if err := model.ValidatePBXField(entity.Technology); err != nil {
		return invalidPBXError("trunk.technology")
	}
	if err := model.ValidatePBXField(entity.Endpoint); err != nil {
		return invalidPBXError("trunk.endpoint")
	}
	return nil
}

func validateRoute(entity model.CallRoute) error {
	if err := model.ValidatePBXField(entity.Name); err != nil {
		return invalidPBXError("route.name")
	}
	if err := model.ValidatePBXField(entity.Direction); err != nil {
		return invalidPBXError("route.direction")
	}
	if err := model.ValidatePBXField(entity.Destination); err != nil {
		return invalidPBXError("route.destination")
	}
	return nil
}

func validateQueue(entity model.Queue) error {
	if err := model.ValidatePBXField(entity.Name); err != nil {
		return invalidPBXError("queue.name")
	}
	return nil
}

func validateConference(entity model.Conference) error {
	if err := model.ValidatePBXField(entity.Name); err != nil {
		return invalidPBXError("conference.name")
	}
	return nil
}

func validateIVR(entity model.IVR) error {
	if err := model.ValidatePBXField(entity.Name); err != nil {
		return invalidPBXError("ivr.name")
	}
	if err := model.ValidatePBXField(entity.DefaultDestination); err != nil {
		return invalidPBXError("ivr.default_destination")
	}
	return nil
}

func validatePromptAssignment(entity model.PromptAssignment) error {
	if err := model.ValidatePBXField(entity.Name); err != nil {
		return invalidPBXError("prompt_assignment.name")
	}
	if err := model.ValidatePBXField(entity.PromptName); err != nil {
		return invalidPBXError("prompt_assignment.prompt_name")
	}
	if err := model.ValidatePBXField(entity.TargetKind); err != nil {
		return invalidPBXError("prompt_assignment.target_kind")
	}
	if err := model.ValidatePBXField(entity.TargetRef); err != nil {
		return invalidPBXError("prompt_assignment.target_ref")
	}
	return nil
}

func validateProvisioningProfile(entity model.ProvisioningProfile) error {
	if err := model.ValidatePBXField(entity.Name); err != nil {
		return invalidPBXError("provisioning_profile.name")
	}
	if err := model.ValidatePBXField(entity.Technology); err != nil {
		return invalidPBXError("provisioning_profile.technology")
	}
	if err := model.ValidatePBXField(entity.Template); err != nil {
		return invalidPBXError("provisioning_profile.template")
	}
	return nil
}

func nextExtensionID(values []model.Extension) int64 {
	return nextIDFromInts(len(values), func(index int) int64 { return values[index].ID })
}
func nextTrunkID(values []model.Trunk) int64 {
	return nextIDFromInts(len(values), func(index int) int64 { return values[index].ID })
}
func nextRouteID(values []model.CallRoute) int64 {
	return nextIDFromInts(len(values), func(index int) int64 { return values[index].ID })
}
func nextQueueID(values []model.Queue) int64 {
	return nextIDFromInts(len(values), func(index int) int64 { return values[index].ID })
}
func nextConferenceID(values []model.Conference) int64 {
	return nextIDFromInts(len(values), func(index int) int64 { return values[index].ID })
}
func nextIVRID(values []model.IVR) int64 {
	return nextIDFromInts(len(values), func(index int) int64 { return values[index].ID })
}
func nextPromptAssignmentID(values []model.PromptAssignment) int64 {
	return nextIDFromInts(len(values), func(index int) int64 { return values[index].ID })
}
func nextProvisioningProfileID(values []model.ProvisioningProfile) int64 {
	return nextIDFromInts(len(values), func(index int) int64 { return values[index].ID })
}

func nextIDFromInts(length int, get func(index int) int64) int64 {
	var maxID int64
	for index := 0; index < length; index++ {
		if value := get(index); value > maxID {
			maxID = value
		}
	}
	return maxID + 1
}

func deleteFromSlice[T any](values []T, match func(T) bool) ([]T, bool) {
	filtered := make([]T, 0, len(values))
	var removed bool
	for _, value := range values {
		if !removed && match(value) {
			removed = true
			continue
		}
		filtered = append(filtered, value)
	}
	return filtered, removed
}

func buildApplyArtifacts(plan model.PBXPlan) []PBXApplyArtifact {
	artifacts := []PBXApplyArtifact{}
	if len(plan.Extensions) > 0 || len(plan.Trunks) > 0 {
		artifacts = append(artifacts, PBXApplyArtifact{Key: "endpoints", Label: "Endpoint inventory", Status: "ready", Summary: "Extension and trunk identities will render endpoint-facing backend configuration."})
	}
	if len(plan.Routes) > 0 || len(plan.IVRs) > 0 {
		artifacts = append(artifacts, PBXApplyArtifact{Key: "dialplan", Label: "Dialplan", Status: "ready", Summary: "Routes and IVRs compile into dialplan intent and routing actions."})
	}
	if len(plan.Queues) > 0 {
		artifacts = append(artifacts, PBXApplyArtifact{Key: "queues", Label: "Queue definitions", Status: "ready", Summary: "Queue objects render call-center membership and queue behavior."})
	}
	if len(plan.Conferences) > 0 {
		artifacts = append(artifacts, PBXApplyArtifact{Key: "conferences", Label: "Conference profiles", Status: "ready", Summary: "Conference objects render moderation and room policy configuration."})
	}
	if len(plan.PromptAssignments) > 0 {
		artifacts = append(artifacts, PBXApplyArtifact{Key: "prompts", Label: "Prompt assignments", Status: "ready", Summary: "Prompt bindings link managed media to IVRs and telephony workflows."})
	}
	if len(plan.ProvisioningProfiles) > 0 {
		artifacts = append(artifacts, PBXApplyArtifact{Key: "provisioning", Label: "Provisioning output", Status: "ready", Summary: "Provisioning profiles render device-facing templates for managed endpoints."})
	}
	if len(artifacts) == 0 {
		artifacts = append(artifacts, PBXApplyArtifact{Key: "plan", Label: "PBX plan", Status: "pending", Summary: "No PBX entities have been defined yet."})
	}
	return artifacts
}

func referencedTechnologies(plan model.PBXPlan) []string {
	values := []string{}
	seen := map[string]struct{}{}
	record := func(value string) {
		normalized := strings.ToLower(model.NormalizePBXField(value))
		if normalized == "" {
			return
		}
		if _, found := seen[normalized]; found {
			return
		}
		seen[normalized] = struct{}{}
		values = append(values, normalized)
	}
	for _, entity := range plan.Extensions {
		record(entity.Technology)
	}
	for _, entity := range plan.Trunks {
		record(entity.Technology)
	}
	for _, entity := range plan.ProvisioningProfiles {
		record(entity.Technology)
	}
	return values
}

func listContainsFold(values []string, target string) bool {
	normalizedTarget := strings.ToLower(strings.TrimSpace(target))
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == normalizedTarget {
			return true
		}
	}
	return false
}

func invalidPBXError(field string) error {
	return &PBXError{Code: "PBX_INVALID", Message: fmt.Sprintf("%s is required", field)}
}

func notFoundPBXError(resource string) error {
	return &PBXError{Code: "PBX_NOT_FOUND", Message: fmt.Sprintf("%s not found", resource)}
}

func formatPBXTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func parsePBXID(value string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || id <= 0 {
		return 0, &PBXError{Code: "PBX_INVALID", Message: "invalid resource id"}
	}
	return id, nil
}
