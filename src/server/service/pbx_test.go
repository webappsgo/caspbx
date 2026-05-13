package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/casapps/caspbx/src/server/model"
	"github.com/casapps/caspbx/src/server/store"
)

type failingPBXStore struct {
	plan model.PBXPlan
	err  error
}

type saveFailPBXStore struct {
	plan    model.PBXPlan
	saveErr error
}

func (pbxStore failingPBXStore) SavePBXPlan(context.Context, model.PBXPlan) (model.PBXPlan, error) {
	if pbxStore.err != nil {
		return model.PBXPlan{}, pbxStore.err
	}
	return pbxStore.plan, nil
}

func (pbxStore failingPBXStore) GetPBXPlan(context.Context) (model.PBXPlan, error) {
	if pbxStore.err != nil {
		return model.PBXPlan{}, pbxStore.err
	}
	return pbxStore.plan, nil
}

func (pbxStore saveFailPBXStore) SavePBXPlan(context.Context, model.PBXPlan) (model.PBXPlan, error) {
	return model.PBXPlan{}, pbxStore.saveErr
}

func (pbxStore saveFailPBXStore) GetPBXPlan(context.Context) (model.PBXPlan, error) {
	return pbxStore.plan, nil
}

func TestPBXServiceCRUDAndApplyPreview(t *testing.T) {
	memoryStore := store.NewMemoryStore()
	if _, err := memoryStore.SaveAsteriskState(context.Background(), model.AsteriskState{
		MinimumSupportedVersion: "12",
		DetectionStatus:         "detected",
		HealthStatus:            model.AsteriskHealthReady,
		ChannelDrivers:          []string{"pjsip"},
		EndpointStacks:          []string{"pjsip"},
		Capabilities: []model.AsteriskCapability{
			{Key: "queues", Available: true},
			{Key: "conferences", Available: true},
			{Key: "prompts", Available: true},
		},
	}); err != nil {
		t.Fatalf("save asterisk state: %v", err)
	}
	pbxService := NewPBXService(memoryStore, memoryStore)
	pbxService.now = func() time.Time { return time.Unix(1_700_500_000, 0).UTC() }

	extension, err := pbxService.CreateExtension(context.Background(), model.Extension{Number: "1000", DisplayName: "Alice", Technology: "PJSIP", Endpoint: "alice"})
	if err != nil || extension.ID != 1 {
		t.Fatalf("create extension: %v / %+v", err, extension)
	}
	trunk, err := pbxService.CreateTrunk(context.Background(), model.Trunk{Name: "Primary Carrier", Technology: "pjsip", Endpoint: "sip.provider.example", Active: true})
	if err != nil || trunk.ID != 1 {
		t.Fatalf("create trunk: %v / %+v", err, trunk)
	}
	route, err := pbxService.CreateRoute(context.Background(), model.CallRoute{Name: "Main Outbound", Direction: "outbound", Match: "_NXXNXXXXXX", Destination: "trunk:1"})
	if err != nil || route.ID != 1 {
		t.Fatalf("create route: %v / %+v", err, route)
	}
	queue, err := pbxService.CreateQueue(context.Background(), model.Queue{Name: "Support", Strategy: "ringall", MemberExtensionNumbers: []string{"1000"}})
	if err != nil || queue.ID != 1 {
		t.Fatalf("create queue: %v / %+v", err, queue)
	}
	conference, err := pbxService.CreateConference(context.Background(), model.Conference{Name: "Daily Standup", AccessCode: "7000", RecordingEnabled: true})
	if err != nil || conference.ID != 1 {
		t.Fatalf("create conference: %v / %+v", err, conference)
	}
	ivr, err := pbxService.CreateIVR(context.Background(), model.IVR{Name: "Main IVR", RootPrompt: "welcome-main", DefaultDestination: "queue:1", TimeoutSeconds: 5})
	if err != nil || ivr.ID != 1 {
		t.Fatalf("create ivr: %v / %+v", err, ivr)
	}
	promptAssignment, err := pbxService.CreatePromptAssignment(context.Background(), model.PromptAssignment{Name: "Main Greeting", PromptName: "welcome-main", TargetKind: "ivr", TargetRef: "1"})
	if err != nil || promptAssignment.ID != 1 {
		t.Fatalf("create prompt assignment: %v / %+v", err, promptAssignment)
	}
	profile, err := pbxService.CreateProvisioningProfile(context.Background(), model.ProvisioningProfile{Name: "Yealink Default", Technology: "pjsip", Template: "yealink-t46"})
	if err != nil || profile.ID != 1 {
		t.Fatalf("create provisioning profile: %v / %+v", err, profile)
	}

	if listed, err := pbxService.ListExtensions(context.Background()); err != nil || len(listed) != 1 {
		t.Fatalf("list extensions: %v / %+v", err, listed)
	}
	if listed, err := pbxService.ListTrunks(context.Background()); err != nil || len(listed) != 1 {
		t.Fatalf("list trunks: %v / %+v", err, listed)
	}
	if listed, err := pbxService.ListRoutes(context.Background()); err != nil || len(listed) != 1 {
		t.Fatalf("list routes: %v / %+v", err, listed)
	}
	if listed, err := pbxService.ListQueues(context.Background()); err != nil || len(listed) != 1 {
		t.Fatalf("list queues: %v / %+v", err, listed)
	}
	if listed, err := pbxService.ListConferences(context.Background()); err != nil || len(listed) != 1 {
		t.Fatalf("list conferences: %v / %+v", err, listed)
	}
	if listed, err := pbxService.ListIVRs(context.Background()); err != nil || len(listed) != 1 {
		t.Fatalf("list ivrs: %v / %+v", err, listed)
	}
	if listed, err := pbxService.ListPromptAssignments(context.Background()); err != nil || len(listed) != 1 {
		t.Fatalf("list prompt assignments: %v / %+v", err, listed)
	}
	if listed, err := pbxService.ListProvisioningProfiles(context.Background()); err != nil || len(listed) != 1 {
		t.Fatalf("list provisioning profiles: %v / %+v", err, listed)
	}
	if found, err := pbxService.GetExtension(context.Background(), extension.ID); err != nil || found.DisplayName != "Alice" {
		t.Fatalf("get extension: %v / %+v", err, found)
	}
	if _, err := pbxService.GetTrunk(context.Background(), trunk.ID); err != nil {
		t.Fatalf("get trunk: %v", err)
	}
	if _, err := pbxService.GetRoute(context.Background(), route.ID); err != nil {
		t.Fatalf("get route: %v", err)
	}
	if _, err := pbxService.GetQueue(context.Background(), queue.ID); err != nil {
		t.Fatalf("get queue: %v", err)
	}
	if _, err := pbxService.GetConference(context.Background(), conference.ID); err != nil {
		t.Fatalf("get conference: %v", err)
	}
	if _, err := pbxService.GetIVR(context.Background(), ivr.ID); err != nil {
		t.Fatalf("get ivr: %v", err)
	}
	if _, err := pbxService.GetPromptAssignment(context.Background(), promptAssignment.ID); err != nil {
		t.Fatalf("get prompt assignment: %v", err)
	}
	if _, err := pbxService.GetProvisioningProfile(context.Background(), profile.ID); err != nil {
		t.Fatalf("get provisioning profile: %v", err)
	}

	preview, err := pbxService.ApplyPreview(context.Background())
	if err != nil || len(preview.Summaries) != 8 || len(preview.Artifacts) < 4 || len(preview.Actions) != 4 {
		t.Fatalf("unexpected apply preview %v / %+v", err, preview)
	}
	if preview.Validations != nil {
		t.Fatalf("expected no validations with supported capabilities, got %+v", preview.Validations)
	}

	if err := pbxService.DeleteExtension(context.Background(), extension.ID); err != nil {
		t.Fatalf("delete extension: %v", err)
	}
	if err := pbxService.DeleteTrunk(context.Background(), trunk.ID); err != nil {
		t.Fatalf("delete trunk: %v", err)
	}
	if err := pbxService.DeleteRoute(context.Background(), route.ID); err != nil {
		t.Fatalf("delete route: %v", err)
	}
	if err := pbxService.DeleteQueue(context.Background(), queue.ID); err != nil {
		t.Fatalf("delete queue: %v", err)
	}
	if err := pbxService.DeleteConference(context.Background(), conference.ID); err != nil {
		t.Fatalf("delete conference: %v", err)
	}
	if err := pbxService.DeleteIVR(context.Background(), ivr.ID); err != nil {
		t.Fatalf("delete ivr: %v", err)
	}
	if err := pbxService.DeletePromptAssignment(context.Background(), promptAssignment.ID); err != nil {
		t.Fatalf("delete prompt assignment: %v", err)
	}
	if err := pbxService.DeleteProvisioningProfile(context.Background(), profile.ID); err != nil {
		t.Fatalf("delete provisioning profile: %v", err)
	}
}

func TestPBXServiceErrorsAndHelpers(t *testing.T) {
	state := model.AsteriskState{
		MinimumSupportedVersion: "12",
		DetectionStatus:         "detected",
		HealthStatus:            model.AsteriskHealthDegraded,
		Capabilities:            []model.AsteriskCapability{{Key: "queues", Available: false}},
	}
	plan := model.PBXPlan{
		Extensions:           []model.Extension{{ID: 2, Technology: "iax2"}},
		Queues:               []model.Queue{{ID: 3, Name: "Support"}},
		Conferences:          []model.Conference{{ID: 4, Name: "Conf"}},
		IVRs:                 []model.IVR{{ID: 5, Name: "Main", DefaultDestination: "queue:3"}},
		PromptAssignments:    []model.PromptAssignment{{ID: 6, Name: "Greeting", PromptName: "greeting", TargetKind: "ivr", TargetRef: "5"}},
		ProvisioningProfiles: []model.ProvisioningProfile{{ID: 7, Name: "Phone", Technology: "sccp", Template: "phone"}},
	}
	if (*PBXError)(nil).Error() != "" || (&PBXError{Message: "boom"}).Error() != "boom" {
		t.Fatalf("expected pbx error string helpers")
	}
	if _, err := parsePBXID("bad"); err == nil {
		t.Fatalf("expected invalid id")
	}
	if id, err := parsePBXID("8"); err != nil || id != 8 {
		t.Fatalf("expected parsed pbx id, got %v / %d", err, id)
	}
	if !listContainsFold([]string{"PJSIP"}, "pjsip") || listContainsFold(nil, "pjsip") {
		t.Fatalf("unexpected listContainsFold helper")
	}
	if values := referencedTechnologies(plan); len(values) != 2 {
		t.Fatalf("unexpected referenced technologies %+v", values)
	}
	if values := referencedTechnologies(model.PBXPlan{Extensions: []model.Extension{{Technology: ""}}, Trunks: []model.Trunk{{Technology: "PJSIP"}}, ProvisioningProfiles: []model.ProvisioningProfile{{Technology: "pjsip"}}}); len(values) != 1 {
		t.Fatalf("expected deduplicated referenced technologies %+v", values)
	}
	if artifacts := buildApplyArtifacts(model.DefaultPBXPlan()); len(artifacts) != 1 || artifacts[0].Status != "pending" {
		t.Fatalf("unexpected empty apply artifacts %+v", artifacts)
	}
	if invalidPBXError("field").(*PBXError).Code != "PBX_INVALID" || notFoundPBXError("thing").(*PBXError).Code != "PBX_NOT_FOUND" {
		t.Fatalf("unexpected pbx error helpers")
	}
	if formatPBXTime(time.Time{}) != "" || formatPBXTime(time.Unix(10, 0)) == "" {
		t.Fatalf("unexpected pbx time formatting")
	}
	if nextIDFromInts(2, func(index int) int64 { return []int64{2, 9}[index] }) != 10 {
		t.Fatalf("unexpected next id helper")
	}
	if nextExtensionID([]model.Extension{{ID: 2}}) != 3 || nextTrunkID([]model.Trunk{{ID: 3}}) != 4 || nextRouteID([]model.CallRoute{{ID: 4}}) != 5 || nextQueueID([]model.Queue{{ID: 5}}) != 6 || nextConferenceID([]model.Conference{{ID: 6}}) != 7 || nextIVRID([]model.IVR{{ID: 7}}) != 8 || nextPromptAssignmentID([]model.PromptAssignment{{ID: 8}}) != 9 || nextProvisioningProfileID([]model.ProvisioningProfile{{ID: 9}}) != 10 {
		t.Fatalf("unexpected next id wrapper results")
	}
	if filtered, removed := deleteFromSlice([]int{1, 2, 3}, func(value int) bool { return value == 2 }); !removed || len(filtered) != 2 {
		t.Fatalf("unexpected deleteFromSlice result %+v / %t", filtered, removed)
	}

	failing := NewPBXService(failingPBXStore{err: errors.New("lookup failed")}, failingAsteriskStore{err: errors.New("lookup failed")})
	if _, err := failing.ListExtensions(context.Background()); err == nil || err.Error() != "lookup failed" {
		t.Fatalf("expected list error, got %v", err)
	}
	for _, action := range []func(context.Context) error{
		func(ctx context.Context) error { _, err := failing.ListTrunks(ctx); return err },
		func(ctx context.Context) error { _, err := failing.ListRoutes(ctx); return err },
		func(ctx context.Context) error { _, err := failing.ListQueues(ctx); return err },
		func(ctx context.Context) error { _, err := failing.ListConferences(ctx); return err },
		func(ctx context.Context) error { _, err := failing.ListIVRs(ctx); return err },
		func(ctx context.Context) error { _, err := failing.ListPromptAssignments(ctx); return err },
		func(ctx context.Context) error { _, err := failing.ListProvisioningProfiles(ctx); return err },
		func(ctx context.Context) error { _, err := failing.GetExtension(ctx, 1); return err },
		func(ctx context.Context) error { _, err := failing.GetTrunk(ctx, 1); return err },
		func(ctx context.Context) error { _, err := failing.GetRoute(ctx, 1); return err },
		func(ctx context.Context) error { _, err := failing.GetQueue(ctx, 1); return err },
		func(ctx context.Context) error { _, err := failing.GetConference(ctx, 1); return err },
		func(ctx context.Context) error { _, err := failing.GetIVR(ctx, 1); return err },
		func(ctx context.Context) error { _, err := failing.GetPromptAssignment(ctx, 1); return err },
		func(ctx context.Context) error { _, err := failing.GetProvisioningProfile(ctx, 1); return err },
		func(ctx context.Context) error {
			_, err := failing.CreateExtension(ctx, model.Extension{Number: "1000", DisplayName: "Alice", Technology: "pjsip"})
			return err
		},
		func(ctx context.Context) error {
			_, err := failing.CreateTrunk(ctx, model.Trunk{Name: "Primary", Technology: "pjsip", Endpoint: "sip.example"})
			return err
		},
		func(ctx context.Context) error {
			_, err := failing.CreateRoute(ctx, model.CallRoute{Name: "Main", Direction: "outbound", Destination: "trunk:1"})
			return err
		},
		func(ctx context.Context) error {
			_, err := failing.CreateQueue(ctx, model.Queue{Name: "Support"})
			return err
		},
		func(ctx context.Context) error {
			_, err := failing.CreateConference(ctx, model.Conference{Name: "Daily"})
			return err
		},
		func(ctx context.Context) error {
			_, err := failing.CreateIVR(ctx, model.IVR{Name: "Main", DefaultDestination: "queue:1"})
			return err
		},
		func(ctx context.Context) error {
			_, err := failing.CreatePromptAssignment(ctx, model.PromptAssignment{Name: "Greeting", PromptName: "welcome", TargetKind: "ivr", TargetRef: "1"})
			return err
		},
		func(ctx context.Context) error {
			_, err := failing.CreateProvisioningProfile(ctx, model.ProvisioningProfile{Name: "Desk", Technology: "pjsip", Template: "yealink"})
			return err
		},
		func(ctx context.Context) error { return failing.DeleteExtension(ctx, 1) },
		func(ctx context.Context) error { return failing.DeleteTrunk(ctx, 1) },
		func(ctx context.Context) error { return failing.DeleteRoute(ctx, 1) },
		func(ctx context.Context) error { return failing.DeleteQueue(ctx, 1) },
		func(ctx context.Context) error { return failing.DeleteConference(ctx, 1) },
		func(ctx context.Context) error { return failing.DeleteIVR(ctx, 1) },
		func(ctx context.Context) error { return failing.DeletePromptAssignment(ctx, 1) },
		func(ctx context.Context) error { return failing.DeleteProvisioningProfile(ctx, 1) },
	} {
		if err := action(context.Background()); err == nil || err.Error() != "lookup failed" {
			t.Fatalf("expected lookup error, got %v", err)
		}
	}
	if _, err := failing.ApplyPreview(context.Background()); err == nil || err.Error() != "lookup failed" {
		t.Fatalf("expected preview error, got %v", err)
	}
	if _, err := NewPBXService(failingPBXStore{plan: model.DefaultPBXPlan()}, failingAsteriskStore{err: errors.New("asterisk failed")}).ApplyPreview(context.Background()); err == nil || err.Error() != "asterisk failed" {
		t.Fatalf("expected asterisk preview error, got %v", err)
	}

	pbxService := NewPBXService(failingPBXStore{plan: plan}, failingAsteriskStore{state: state})
	for _, testCase := range []struct {
		name string
		err  error
	}{
		{"extension", validateExtension(model.Extension{})},
		{"extension-display", validateExtension(model.Extension{Number: "1000"})},
		{"extension-technology", validateExtension(model.Extension{Number: "1000", DisplayName: "Alice"})},
		{"trunk", validateTrunk(model.Trunk{})},
		{"trunk-technology", validateTrunk(model.Trunk{Name: "Primary"})},
		{"trunk-endpoint", validateTrunk(model.Trunk{Name: "Primary", Technology: "pjsip"})},
		{"route", validateRoute(model.CallRoute{})},
		{"route-direction", validateRoute(model.CallRoute{Name: "Main"})},
		{"route-destination", validateRoute(model.CallRoute{Name: "Main", Direction: "outbound"})},
		{"queue", validateQueue(model.Queue{})},
		{"conference", validateConference(model.Conference{})},
		{"ivr", validateIVR(model.IVR{})},
		{"ivr-destination", validateIVR(model.IVR{Name: "Main"})},
		{"prompt", validatePromptAssignment(model.PromptAssignment{})},
		{"prompt-name", validatePromptAssignment(model.PromptAssignment{Name: "Main"})},
		{"prompt-target-kind", validatePromptAssignment(model.PromptAssignment{Name: "Main", PromptName: "welcome"})},
		{"prompt-target-ref", validatePromptAssignment(model.PromptAssignment{Name: "Main", PromptName: "welcome", TargetKind: "ivr"})},
		{"provisioning", validateProvisioningProfile(model.ProvisioningProfile{})},
		{"provisioning-technology", validateProvisioningProfile(model.ProvisioningProfile{Name: "Desk"})},
		{"provisioning-template", validateProvisioningProfile(model.ProvisioningProfile{Name: "Desk", Technology: "pjsip"})},
	} {
		if testCase.err == nil || testCase.err.(*PBXError).Code != "PBX_INVALID" {
			t.Fatalf("expected validation error for %s, got %v", testCase.name, testCase.err)
		}
	}
	for _, action := range []func(context.Context) error{
		func(ctx context.Context) error {
			_, err := pbxService.CreateExtension(ctx, model.Extension{})
			return err
		},
		func(ctx context.Context) error { _, err := pbxService.CreateTrunk(ctx, model.Trunk{}); return err },
		func(ctx context.Context) error { _, err := pbxService.CreateRoute(ctx, model.CallRoute{}); return err },
		func(ctx context.Context) error { _, err := pbxService.CreateQueue(ctx, model.Queue{}); return err },
		func(ctx context.Context) error {
			_, err := pbxService.CreateConference(ctx, model.Conference{})
			return err
		},
		func(ctx context.Context) error { _, err := pbxService.CreateIVR(ctx, model.IVR{}); return err },
		func(ctx context.Context) error {
			_, err := pbxService.CreatePromptAssignment(ctx, model.PromptAssignment{})
			return err
		},
		func(ctx context.Context) error {
			_, err := pbxService.CreateProvisioningProfile(ctx, model.ProvisioningProfile{})
			return err
		},
	} {
		if err := action(context.Background()); err == nil || err.(*PBXError).Code != "PBX_INVALID" {
			t.Fatalf("expected create validation error, got %v", err)
		}
	}
	for _, action := range []func(context.Context, int64) error{
		pbxService.DeleteExtension,
		pbxService.DeleteTrunk,
		pbxService.DeleteRoute,
		pbxService.DeleteQueue,
		pbxService.DeleteConference,
		pbxService.DeleteIVR,
		pbxService.DeletePromptAssignment,
		pbxService.DeleteProvisioningProfile,
	} {
		if err := action(context.Background(), 999); err == nil || err.(*PBXError).Code != "PBX_NOT_FOUND" {
			t.Fatalf("expected delete not-found error, got %v", err)
		}
	}
	for _, action := range []func(context.Context, int64) error{
		func(ctx context.Context, id int64) error { _, err := pbxService.GetExtension(ctx, id); return err },
		func(ctx context.Context, id int64) error { _, err := pbxService.GetTrunk(ctx, id); return err },
		func(ctx context.Context, id int64) error { _, err := pbxService.GetRoute(ctx, id); return err },
		func(ctx context.Context, id int64) error { _, err := pbxService.GetQueue(ctx, id); return err },
		func(ctx context.Context, id int64) error { _, err := pbxService.GetConference(ctx, id); return err },
		func(ctx context.Context, id int64) error { _, err := pbxService.GetIVR(ctx, id); return err },
		func(ctx context.Context, id int64) error {
			_, err := pbxService.GetPromptAssignment(ctx, id)
			return err
		},
		func(ctx context.Context, id int64) error {
			_, err := pbxService.GetProvisioningProfile(ctx, id)
			return err
		},
	} {
		if err := action(context.Background(), 999); err == nil || err.(*PBXError).Code != "PBX_NOT_FOUND" {
			t.Fatalf("expected get not-found error, got %v", err)
		}
	}
	preview, err := pbxService.ApplyPreview(context.Background())
	if err != nil || len(preview.Validations) < 4 {
		t.Fatalf("expected validation-rich apply preview, got %v / %+v", err, preview)
	}

	saveFail := NewPBXService(saveFailPBXStore{plan: model.PBXPlan{
		Extensions:           []model.Extension{{ID: 1, Number: "1000", DisplayName: "Alice", Technology: "pjsip"}},
		Trunks:               []model.Trunk{{ID: 1, Name: "Primary", Technology: "pjsip", Endpoint: "sip.example"}},
		Routes:               []model.CallRoute{{ID: 1, Name: "Main", Direction: "outbound", Destination: "trunk:1"}},
		Queues:               []model.Queue{{ID: 1, Name: "Support"}},
		Conferences:          []model.Conference{{ID: 1, Name: "Daily"}},
		IVRs:                 []model.IVR{{ID: 1, Name: "Main", DefaultDestination: "queue:1"}},
		PromptAssignments:    []model.PromptAssignment{{ID: 1, Name: "Greeting", PromptName: "welcome", TargetKind: "ivr", TargetRef: "1"}},
		ProvisioningProfiles: []model.ProvisioningProfile{{ID: 1, Name: "Desk", Technology: "pjsip", Template: "yealink"}},
	}, saveErr: errors.New("save failed")}, failingAsteriskStore{state: state})
	for _, action := range []func(context.Context) error{
		func(ctx context.Context) error {
			_, err := saveFail.CreateExtension(ctx, model.Extension{Number: "2000", DisplayName: "Bob", Technology: "pjsip"})
			return err
		},
		func(ctx context.Context) error {
			_, err := saveFail.CreateTrunk(ctx, model.Trunk{Name: "Backup", Technology: "pjsip", Endpoint: "sip.backup"})
			return err
		},
		func(ctx context.Context) error {
			_, err := saveFail.CreateRoute(ctx, model.CallRoute{Name: "Backup", Direction: "outbound", Destination: "trunk:1"})
			return err
		},
		func(ctx context.Context) error {
			_, err := saveFail.CreateQueue(ctx, model.Queue{Name: "Sales"})
			return err
		},
		func(ctx context.Context) error {
			_, err := saveFail.CreateConference(ctx, model.Conference{Name: "All Hands"})
			return err
		},
		func(ctx context.Context) error {
			_, err := saveFail.CreateIVR(ctx, model.IVR{Name: "Night", DefaultDestination: "queue:1"})
			return err
		},
		func(ctx context.Context) error {
			_, err := saveFail.CreatePromptAssignment(ctx, model.PromptAssignment{Name: "Night", PromptName: "night", TargetKind: "ivr", TargetRef: "1"})
			return err
		},
		func(ctx context.Context) error {
			_, err := saveFail.CreateProvisioningProfile(ctx, model.ProvisioningProfile{Name: "Lobby", Technology: "pjsip", Template: "phone"})
			return err
		},
		func(ctx context.Context) error { return saveFail.DeleteExtension(ctx, 1) },
		func(ctx context.Context) error { return saveFail.DeleteTrunk(ctx, 1) },
		func(ctx context.Context) error { return saveFail.DeleteRoute(ctx, 1) },
		func(ctx context.Context) error { return saveFail.DeleteQueue(ctx, 1) },
		func(ctx context.Context) error { return saveFail.DeleteConference(ctx, 1) },
		func(ctx context.Context) error { return saveFail.DeleteIVR(ctx, 1) },
		func(ctx context.Context) error { return saveFail.DeletePromptAssignment(ctx, 1) },
		func(ctx context.Context) error { return saveFail.DeleteProvisioningProfile(ctx, 1) },
	} {
		if err := action(context.Background()); err == nil || err.Error() != "save failed" {
			t.Fatalf("expected save failure, got %v", err)
		}
	}
}
