package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/casapps/caspbx/src/server/model"
	"github.com/casapps/caspbx/src/server/store"
)

func TestUserCommunicationsServiceDashboardAndSurfaces(t *testing.T) {
	memoryStore := seededUserCommunicationsStore(t)
	serviceValue := NewUserCommunicationsService(memoryStore, memoryStore, memoryStore, memoryStore)
	serviceValue.now = func() time.Time { return time.Unix(200, 0).UTC() }

	dashboard, err := serviceValue.Dashboard(context.Background(), 1)
	if err != nil || dashboard.ExtensionNumber != "1000" || dashboard.ContactCount != 1 || len(dashboard.VisibleSurfaces) < 6 {
		t.Fatalf("expected dashboard summary, got %v / %+v", err, dashboard)
	}

	contacts, err := serviceValue.ListContacts(context.Background(), 1)
	if err != nil || len(contacts) != 1 {
		t.Fatalf("expected contacts, got %v / %+v", err, contacts)
	}
	contact, err := serviceValue.GetContact(context.Background(), 1, contacts[0].ID)
	if err != nil || contact.DisplayName != "Bob Example" {
		t.Fatalf("expected contact lookup, got %v / %+v", err, contact)
	}

	createdContact, err := serviceValue.CreateContact(context.Background(), 1, model.UserContact{
		DisplayName:     "Carol Example",
		ExtensionNumber: "1002",
	})
	if err != nil || createdContact.ID == 0 {
		t.Fatalf("expected created contact, got %v / %+v", err, createdContact)
	}
	if err := serviceValue.DeleteContact(context.Background(), 1, createdContact.ID); err != nil {
		t.Fatalf("expected deleted contact, got %v", err)
	}

	voicemails, err := serviceValue.ListVoicemails(context.Background(), 1)
	if err != nil || len(voicemails) != 1 {
		t.Fatalf("expected voicemails, got %v / %+v", err, voicemails)
	}

	callHistory, err := serviceValue.ListCallHistory(context.Background(), 1)
	if err != nil || len(callHistory) != 1 {
		t.Fatalf("expected call history, got %v / %+v", err, callHistory)
	}

	messages, err := serviceValue.ListMessages(context.Background(), 1)
	if err != nil || len(messages) != 1 {
		t.Fatalf("expected messages, got %v / %+v", err, messages)
	}

	presence, err := serviceValue.Presence(context.Background(), 1)
	if err != nil || presence.ExtensionNumber != "1000" || presence.Transport != "xmpp" {
		t.Fatalf("expected presence view, got %v / %+v", err, presence)
	}

	webphone, err := serviceValue.Webphone(context.Background(), 1)
	if err != nil || !webphone.Enabled || webphone.Endpoint != "alice-web" || !webphone.SecureTransport {
		t.Fatalf("expected webphone view, got %v / %+v", err, webphone)
	}

	settings, err := serviceValue.GetSettings(context.Background(), 1)
	if err != nil || settings.ExtensionID == 0 || settings.PreferredEndpoint != "alice-web" {
		t.Fatalf("expected settings lookup, got %v / %+v", err, settings)
	}

	updatedSettings, err := serviceValue.UpdateSettings(context.Background(), 1, model.UserCommunicationSettings{
		DoNotDisturb:          true,
		CallForwardingTarget:  "1001",
		VoicemailEnabled:      true,
		WebphoneEnabled:       true,
		PresenceEnabled:       true,
		MessagingEnabled:      true,
		PreferredContactEmail: "notify@example.com",
	})
	if err != nil || !updatedSettings.DoNotDisturb || updatedSettings.PreferredContactEmail != "notify@example.com" {
		t.Fatalf("expected updated settings, got %v / %+v", err, updatedSettings)
	}

	presence, err = serviceValue.Presence(context.Background(), 1)
	if err != nil || presence.Status != "do_not_disturb" {
		t.Fatalf("expected DND presence state, got %v / %+v", err, presence)
	}
}

func TestUserCommunicationsServiceErrorsAndHelpers(t *testing.T) {
	memoryStore := seededUserCommunicationsStore(t)
	serviceValue := NewUserCommunicationsService(memoryStore, memoryStore, memoryStore, memoryStore)

	if _, err := serviceValue.GetContact(context.Background(), 1, 999); err == nil {
		t.Fatalf("expected missing contact error")
	}
	if err := serviceValue.DeleteContact(context.Background(), 1, 999); err == nil {
		t.Fatalf("expected missing delete contact error")
	}
	if _, err := serviceValue.CreateContact(context.Background(), 1, model.UserContact{}); err == nil {
		t.Fatalf("expected invalid contact error")
	}
	if _, err := serviceValue.CreateContact(context.Background(), 1, model.UserContact{DisplayName: "Bad Email", Email: "invalid"}); err == nil {
		t.Fatalf("expected bad email error")
	}
	if _, err := serviceValue.CreateContact(context.Background(), 1, model.UserContact{DisplayName: "No Destination"}); err == nil {
		t.Fatalf("expected missing destination contact error")
	}
	if _, err := serviceValue.UpdateSettings(context.Background(), 1, model.UserCommunicationSettings{
		VoicemailEnabled: true,
		WebphoneEnabled:  true,
		PresenceEnabled:  true,
		MessagingEnabled: true,
		PreferredContactEmail: "not-an-email",
	}); err == nil {
		t.Fatalf("expected bad preferred email error")
	}

	noCapabilityStore := store.NewMemoryStore()
	if _, err := noCapabilityStore.SaveUser(context.Background(), model.User{Username: "alice", DisplayName: "Alice", AccountEmail: "alice@example.com", Enabled: true}); err != nil {
		t.Fatalf("save user: %v", err)
	}
	if _, err := noCapabilityStore.SavePBXPlan(context.Background(), model.PBXPlan{
		Extensions: []model.Extension{{ID: 1, UserID: 1, Number: "1000", Technology: "pjsip", Endpoint: "alice-web", VoicemailEnabled: true}},
	}); err != nil {
		t.Fatalf("save plan: %v", err)
	}
	if _, err := noCapabilityStore.SaveAsteriskState(context.Background(), model.AsteriskState{
		MinimumSupportedVersion: "12",
		DetectionStatus:         "detected",
		HealthStatus:            model.AsteriskHealthReady,
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}
	noCapabilityService := NewUserCommunicationsService(noCapabilityStore, noCapabilityStore, noCapabilityStore, noCapabilityStore)
	if _, err := noCapabilityService.ListVoicemails(context.Background(), 1); err == nil {
		t.Fatalf("expected unavailable voicemail")
	}
	if _, err := noCapabilityService.ListCallHistory(context.Background(), 1); err == nil {
		t.Fatalf("expected unavailable call history")
	}
	if _, err := noCapabilityService.ListMessages(context.Background(), 1); err == nil {
		t.Fatalf("expected unavailable messages")
	}
	if _, err := noCapabilityService.Presence(context.Background(), 1); err == nil {
		t.Fatalf("expected unavailable presence")
	}
	if _, err := noCapabilityService.Webphone(context.Background(), 1); err == nil {
		t.Fatalf("expected unavailable webphone")
	}
	if _, err := noCapabilityService.UpdateSettings(context.Background(), 1, model.UserCommunicationSettings{
		VoicemailEnabled: true,
		WebphoneEnabled:  true,
		PresenceEnabled:  true,
		MessagingEnabled: true,
	}); err == nil {
		t.Fatalf("expected invalid unavailable settings")
	}
	dashboard, err := noCapabilityService.Dashboard(context.Background(), 1)
	if err != nil || len(dashboard.Warnings) == 0 {
		t.Fatalf("expected dashboard warning, got %v / %+v", err, dashboard)
	}

	if _, err := noCapabilityService.Dashboard(context.Background(), 99); err == nil {
		t.Fatalf("expected missing user error")
	}

	if !hasAvailableCapability(model.AsteriskState{Capabilities: []model.AsteriskCapability{{Key: "presence", Available: true}}}, "presence") {
		t.Fatalf("expected capability helper")
	}
	if !hasAnyAvailableCapability(model.AsteriskState{Capabilities: []model.AsteriskCapability{{Key: "presence", Available: true}}}, "xmpp", "presence") {
		t.Fatalf("expected any capability helper")
	}
	if !hasUserCommunicationSubsystem(model.AsteriskState{Subsystems: []model.AsteriskManagedSubsystem{{Key: "messaging_backend", Provider: "xmpp"}}}, "messaging_backend") {
		t.Fatalf("expected subsystem helper")
	}
	if len(messagingCapabilities(model.AsteriskState{
		Capabilities: []model.AsteriskCapability{{Key: "presence", Available: true}, {Key: "mail_delivery", Available: true}},
		Subsystems:   []model.AsteriskManagedSubsystem{{Key: "messaging_backend", Provider: "xmpp"}},
	})) != 3 {
		t.Fatalf("expected messaging capabilities helper")
	}
	if transport := messagingTransport(model.AsteriskState{Capabilities: []model.AsteriskCapability{{Key: "mail_delivery", Available: true}}}); transport != "mail" {
		t.Fatalf("expected mail transport, got %q", transport)
	}
	if transport := messagingTransport(model.AsteriskState{Subsystems: []model.AsteriskManagedSubsystem{{Key: "messaging_backend", Provider: "xmpp"}}}); transport != "backend" {
		t.Fatalf("expected backend transport, got %q", transport)
	}
	if invalidUserCommunicationsError("bad").Error() != "bad" || notFoundUserCommunicationsError("user").Error() == "" || unavailableUserCommunicationsError("webphone").Error() == "" {
		t.Fatalf("expected service error helpers to render messages")
	}
	if (*UserCommunicationsError)(nil).Error() != "" {
		t.Fatalf("expected nil error receiver to render empty string")
	}
	if firstNonEmptyString("", " secondary ") != "secondary" || firstNonEmptyString("", "  ") != "" {
		t.Fatalf("expected firstNonEmptyString helper behavior")
	}
}

func TestUserCommunicationsServiceStoreFailures(t *testing.T) {
	serviceValue := NewUserCommunicationsService(failingUserStore{userError: errors.New("user failure")}, failingUserCommunicationPBXStore{err: errors.New("pbx failure")}, failingUserCommunicationAsteriskStore{err: errors.New("asterisk failure")}, failingCommunicationStore{err: errors.New("communication failure")})
	if _, err := serviceValue.Dashboard(context.Background(), 1); err == nil {
		t.Fatalf("expected user lookup failure")
	}

	memoryStore := store.NewMemoryStore()
	if _, err := memoryStore.SaveUser(context.Background(), model.User{Username: "alice", DisplayName: "Alice", AccountEmail: "alice@example.com", Enabled: true}); err != nil {
		t.Fatalf("save user: %v", err)
	}
	serviceValue = NewUserCommunicationsService(memoryStore, failingUserCommunicationPBXStore{err: errors.New("pbx failure")}, memoryStore, memoryStore)
	if _, err := serviceValue.Dashboard(context.Background(), 1); err == nil {
		t.Fatalf("expected pbx store failure")
	}
	serviceValue = NewUserCommunicationsService(memoryStore, memoryStore, failingUserCommunicationAsteriskStore{err: errors.New("asterisk failure")}, memoryStore)
	if _, err := serviceValue.Dashboard(context.Background(), 1); err == nil {
		t.Fatalf("expected asterisk store failure")
	}
	serviceValue = NewUserCommunicationsService(memoryStore, memoryStore, memoryStore, failingCommunicationStore{err: errors.New("communication failure")})
	if _, err := serviceValue.Dashboard(context.Background(), 1); err == nil {
		t.Fatalf("expected communication store failure")
	}
}

func TestUserCommunicationsServiceAdditionalBranches(t *testing.T) {
	memoryStore := store.NewMemoryStore()
	if _, err := memoryStore.SaveUser(context.Background(), model.User{Username: "alice", DisplayName: "Alice", AccountEmail: "alice@example.com", Enabled: true}); err != nil {
		t.Fatalf("save user: %v", err)
	}
	if _, err := memoryStore.SaveAsteriskState(context.Background(), model.AsteriskState{
		MinimumSupportedVersion: "12",
		DetectionStatus:         "detected",
		HealthStatus:            model.AsteriskHealthReady,
		Capabilities: []model.AsteriskCapability{
			{Key: "browser_calling", Available: true},
			{Key: "tls", Available: true},
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	serviceValue := NewUserCommunicationsService(memoryStore, memoryStore, memoryStore, memoryStore)
	created, err := serviceValue.CreateContact(context.Background(), 1, model.UserContact{DisplayName: "Carol", PhoneNumber: "18005550123"})
	if err != nil || created.CreatedAt.IsZero() {
		t.Fatalf("expected create contact with default now, got %v / %+v", err, created)
	}
	dashboard, err := serviceValue.Dashboard(context.Background(), 1)
	if err != nil || len(dashboard.Warnings) == 0 {
		t.Fatalf("expected dashboard warning without extension, got %v / %+v", err, dashboard)
	}

	if _, err := memoryStore.SavePBXPlan(context.Background(), model.PBXPlan{
		Extensions: []model.Extension{{ID: 1, UserID: 1, Number: "1000", Technology: "pjsip", Endpoint: "alice", VoicemailEnabled: false}},
	}); err != nil {
		t.Fatalf("save plan: %v", err)
	}
	settings, err := serviceValue.GetSettings(context.Background(), 1)
	if err != nil || settings.VoicemailEnabled {
		t.Fatalf("expected voicemail disabled from extension, got %v / %+v", err, settings)
	}
	if _, err := serviceValue.UpdateSettings(context.Background(), 1, model.UserCommunicationSettings{VoicemailEnabled: true}); err == nil {
		t.Fatalf("expected voicemail-unavailable validation")
	}
	if _, err := serviceValue.UpdateSettings(context.Background(), 1, model.UserCommunicationSettings{WebphoneEnabled: true}); err == nil {
		t.Fatalf("expected webphone-unavailable validation without telephony")
	}

	stateWithMessaging := model.AsteriskState{Capabilities: []model.AsteriskCapability{{Key: "presence", Available: true}}}
	if transport := messagingTransport(stateWithMessaging); transport != "" {
		t.Fatalf("expected empty default transport, got %q", transport)
	}
	if _, found := findUserExtension(model.PBXPlan{}, 1); found {
		t.Fatalf("expected missing user extension")
	}

	branchStore := seededUserCommunicationsStore(t)
	dashboardService := NewUserCommunicationsService(branchStore, branchStore, branchStore, branchAwareCommunicationStore{CommunicationStore: branchStore, failContacts: errors.New("contacts"), settingsNotFound: true})
	if _, err := dashboardService.Dashboard(context.Background(), 1); err == nil {
		t.Fatalf("expected dashboard contacts failure")
	}
	dashboardService = NewUserCommunicationsService(branchStore, branchStore, branchStore, branchAwareCommunicationStore{CommunicationStore: branchStore, failVoicemails: errors.New("voicemails"), settingsNotFound: true})
	if _, err := dashboardService.Dashboard(context.Background(), 1); err == nil {
		t.Fatalf("expected dashboard voicemail failure")
	}
	dashboardService = NewUserCommunicationsService(branchStore, branchStore, branchStore, branchAwareCommunicationStore{CommunicationStore: branchStore, failCallRecords: errors.New("calls"), settingsNotFound: true})
	if _, err := dashboardService.Dashboard(context.Background(), 1); err == nil {
		t.Fatalf("expected dashboard call failure")
	}
	dashboardService = NewUserCommunicationsService(branchStore, branchStore, branchStore, branchAwareCommunicationStore{CommunicationStore: branchStore, failMessages: errors.New("messages"), settingsNotFound: true})
	if _, err := dashboardService.Dashboard(context.Background(), 1); err == nil {
		t.Fatalf("expected dashboard message failure")
	}

	loadFailService := NewUserCommunicationsService(failingUserStore{userError: errors.New("lookup failure")}, branchStore, branchStore, branchStore)
	if _, err := loadFailService.ListContacts(context.Background(), 1); err == nil {
		t.Fatalf("expected list contacts load error")
	}
	if _, err := loadFailService.GetContact(context.Background(), 1, 1); err == nil {
		t.Fatalf("expected get contact load error")
	}
	if _, err := loadFailService.CreateContact(context.Background(), 1, model.UserContact{DisplayName: "Carol", PhoneNumber: "1"}); err == nil {
		t.Fatalf("expected create contact load error")
	}
	if err := loadFailService.DeleteContact(context.Background(), 1, 1); err == nil {
		t.Fatalf("expected delete contact load error")
	}
	if _, err := loadFailService.ListVoicemails(context.Background(), 1); err == nil {
		t.Fatalf("expected list voicemails load error")
	}
	if _, err := loadFailService.ListCallHistory(context.Background(), 1); err == nil {
		t.Fatalf("expected list call history load error")
	}
	if _, err := loadFailService.ListMessages(context.Background(), 1); err == nil {
		t.Fatalf("expected list messages load error")
	}
	if _, err := loadFailService.Presence(context.Background(), 1); err == nil {
		t.Fatalf("expected presence load error")
	}
	if _, err := loadFailService.Webphone(context.Background(), 1); err == nil {
		t.Fatalf("expected webphone load error")
	}
	if _, err := loadFailService.GetSettings(context.Background(), 1); err == nil {
		t.Fatalf("expected get settings load error")
	}
	if _, err := loadFailService.UpdateSettings(context.Background(), 1, model.UserCommunicationSettings{}); err == nil {
		t.Fatalf("expected update settings load error")
	}

	messagingStateStore := seededUserCommunicationsStore(t)
	if _, err := messagingStateStore.SaveUserCommunicationSettings(context.Background(), model.UserCommunicationSettings{
		UserID:           1,
		ExtensionID:      1,
		VoicemailEnabled: true,
		WebphoneEnabled:  true,
		PresenceEnabled:  true,
		MessagingEnabled: false,
	}); err != nil {
		t.Fatalf("save disabled messaging settings: %v", err)
	}
	messagingStateService := NewUserCommunicationsService(messagingStateStore, messagingStateStore, messagingStateStore, messagingStateStore)
	if _, err := messagingStateService.ListMessages(context.Background(), 1); err == nil {
		t.Fatalf("expected disabled messaging surface")
	}
	if _, err := messagingStateService.UpdateSettings(context.Background(), 1, model.UserCommunicationSettings{PresenceEnabled: true, MessagingEnabled: false, VoicemailEnabled: false, WebphoneEnabled: false}); err != nil {
		t.Fatalf("expected presence-enabled settings when messaging exists, got %v", err)
	}
	if _, err := messagingStateStore.SaveAsteriskState(context.Background(), model.AsteriskState{
		MinimumSupportedVersion: "12",
		DetectionStatus:         "detected",
		HealthStatus:            model.AsteriskHealthReady,
		ChannelDrivers:          []string{"pjsip"},
		EndpointStacks:          []string{"pjsip"},
		Capabilities: []model.AsteriskCapability{
			{Key: "browser_calling", Available: true},
			{Key: "tls", Available: true},
		},
	}); err != nil {
		t.Fatalf("save state without messaging: %v", err)
	}
	if _, err := messagingStateService.UpdateSettings(context.Background(), 1, model.UserCommunicationSettings{PresenceEnabled: true}); err == nil {
		t.Fatalf("expected presence unavailable validation")
	}
	if _, err := messagingStateService.UpdateSettings(context.Background(), 1, model.UserCommunicationSettings{MessagingEnabled: true}); err == nil {
		t.Fatalf("expected messaging unavailable validation")
	}
}

func seededUserCommunicationsStore(t *testing.T) *store.MemoryStore {
	t.Helper()
	memoryStore := store.NewMemoryStore()
	if _, err := memoryStore.SaveUser(context.Background(), model.User{
		Username:     "alice",
		DisplayName:  "Alice Example",
		AccountEmail: "alice@example.com",
		Enabled:      true,
		Visibility:   model.UserVisibilityPublic,
	}); err != nil {
		t.Fatalf("save user: %v", err)
	}
	if _, err := memoryStore.SavePBXPlan(context.Background(), model.PBXPlan{
		Extensions: []model.Extension{
			{ID: 1, UserID: 1, Number: "1000", DisplayName: "Alice", Technology: "pjsip", Endpoint: "alice-web", VoicemailEnabled: true},
		},
	}); err != nil {
		t.Fatalf("save plan: %v", err)
	}
	if _, err := memoryStore.SaveAsteriskState(context.Background(), model.AsteriskState{
		MinimumSupportedVersion: "12",
		DetectionStatus:         "detected",
		HealthStatus:            model.AsteriskHealthReady,
		ChannelDrivers:          []string{"pjsip"},
		EndpointStacks:          []string{"pjsip"},
		Capabilities: []model.AsteriskCapability{
			{Key: "browser_calling", Available: true},
			{Key: "tls", Available: true},
			{Key: "voicemail", Available: true},
			{Key: "presence", Available: true},
			{Key: "xmpp", Available: true},
			{Key: "mail_delivery", Available: true},
		},
		Subsystems: []model.AsteriskManagedSubsystem{
			{Key: "messaging_backend", Provider: "xmpp", Healthy: true},
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}
	if _, err := memoryStore.SaveUserCommunicationSettings(context.Background(), model.UserCommunicationSettings{
		UserID:            1,
		ExtensionID:       1,
		VoicemailEnabled:  true,
		WebphoneEnabled:   true,
		PresenceEnabled:   true,
		MessagingEnabled:  true,
		PreferredEndpoint: "alice-web",
	}); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	if _, err := memoryStore.SaveUserContact(context.Background(), model.UserContact{UserID: 1, DisplayName: "Bob Example", ExtensionNumber: "1001"}); err != nil {
		t.Fatalf("save contact: %v", err)
	}
	if _, err := memoryStore.SaveUserVoicemail(context.Background(), model.UserVoicemail{UserID: 1, From: "1002", ReceivedAt: time.Unix(100, 0), Recording: true}); err != nil {
		t.Fatalf("save voicemail: %v", err)
	}
	if _, err := memoryStore.SaveUserCallRecord(context.Background(), model.UserCallRecord{UserID: 1, Counterparty: "1003", StartedAt: time.Unix(101, 0), Direction: "inbound"}); err != nil {
		t.Fatalf("save call record: %v", err)
	}
	if _, err := memoryStore.SaveUserMessage(context.Background(), model.UserMessage{UserID: 1, Counterparty: "operator", Body: "Need help", ReceivedAt: time.Unix(102, 0), Transport: "xmpp"}); err != nil {
		t.Fatalf("save message: %v", err)
	}
	return memoryStore
}

type failingUserStore struct{ userError error }

func (storeValue failingUserStore) SaveUser(context.Context, model.User) (model.User, error) { return model.User{}, storeValue.userError }
func (storeValue failingUserStore) FindUserByUsername(context.Context, string) (model.User, error) {
	return model.User{}, storeValue.userError
}
func (storeValue failingUserStore) FindUserByEmail(context.Context, string) (model.User, error) {
	return model.User{}, storeValue.userError
}
func (storeValue failingUserStore) FindUserByID(context.Context, int64) (model.User, error) {
	return model.User{}, storeValue.userError
}

type failingUserCommunicationPBXStore struct{ err error }

func (storeValue failingUserCommunicationPBXStore) SavePBXPlan(context.Context, model.PBXPlan) (model.PBXPlan, error) {
	return model.PBXPlan{}, storeValue.err
}
func (storeValue failingUserCommunicationPBXStore) GetPBXPlan(context.Context) (model.PBXPlan, error) {
	return model.PBXPlan{}, storeValue.err
}

type failingUserCommunicationAsteriskStore struct{ err error }

func (storeValue failingUserCommunicationAsteriskStore) SaveAsteriskState(context.Context, model.AsteriskState) (model.AsteriskState, error) {
	return model.AsteriskState{}, storeValue.err
}
func (storeValue failingUserCommunicationAsteriskStore) GetAsteriskState(context.Context) (model.AsteriskState, error) {
	return model.AsteriskState{}, storeValue.err
}

type failingCommunicationStore struct{ err error }

func (storeValue failingCommunicationStore) SaveUserCommunicationSettings(context.Context, model.UserCommunicationSettings) (model.UserCommunicationSettings, error) {
	return model.UserCommunicationSettings{}, storeValue.err
}
func (storeValue failingCommunicationStore) FindUserCommunicationSettings(context.Context, int64) (model.UserCommunicationSettings, error) {
	return model.UserCommunicationSettings{}, storeValue.err
}
func (storeValue failingCommunicationStore) SaveUserContact(context.Context, model.UserContact) (model.UserContact, error) {
	return model.UserContact{}, storeValue.err
}
func (storeValue failingCommunicationStore) FindUserContact(context.Context, int64, int64) (model.UserContact, error) {
	return model.UserContact{}, storeValue.err
}
func (storeValue failingCommunicationStore) ListUserContacts(context.Context, int64) ([]model.UserContact, error) {
	return nil, storeValue.err
}
func (storeValue failingCommunicationStore) DeleteUserContact(context.Context, int64, int64) error {
	return storeValue.err
}
func (storeValue failingCommunicationStore) SaveUserVoicemail(context.Context, model.UserVoicemail) (model.UserVoicemail, error) {
	return model.UserVoicemail{}, storeValue.err
}
func (storeValue failingCommunicationStore) ListUserVoicemails(context.Context, int64) ([]model.UserVoicemail, error) {
	return nil, storeValue.err
}
func (storeValue failingCommunicationStore) SaveUserCallRecord(context.Context, model.UserCallRecord) (model.UserCallRecord, error) {
	return model.UserCallRecord{}, storeValue.err
}
func (storeValue failingCommunicationStore) ListUserCallRecords(context.Context, int64) ([]model.UserCallRecord, error) {
	return nil, storeValue.err
}
func (storeValue failingCommunicationStore) SaveUserMessage(context.Context, model.UserMessage) (model.UserMessage, error) {
	return model.UserMessage{}, storeValue.err
}
func (storeValue failingCommunicationStore) ListUserMessages(context.Context, int64) ([]model.UserMessage, error) {
	return nil, storeValue.err
}

type branchAwareCommunicationStore struct {
	store.CommunicationStore
	settingsNotFound bool
	failContacts     error
	failVoicemails   error
	failCallRecords  error
	failMessages     error
}

func (storeValue branchAwareCommunicationStore) FindUserCommunicationSettings(ctx context.Context, userID int64) (model.UserCommunicationSettings, error) {
	if storeValue.settingsNotFound {
		return model.UserCommunicationSettings{}, store.ErrNotFound
	}
	return storeValue.CommunicationStore.FindUserCommunicationSettings(ctx, userID)
}
func (storeValue branchAwareCommunicationStore) ListUserContacts(ctx context.Context, userID int64) ([]model.UserContact, error) {
	if storeValue.failContacts != nil {
		return nil, storeValue.failContacts
	}
	return storeValue.CommunicationStore.ListUserContacts(ctx, userID)
}
func (storeValue branchAwareCommunicationStore) ListUserVoicemails(ctx context.Context, userID int64) ([]model.UserVoicemail, error) {
	if storeValue.failVoicemails != nil {
		return nil, storeValue.failVoicemails
	}
	return storeValue.CommunicationStore.ListUserVoicemails(ctx, userID)
}
func (storeValue branchAwareCommunicationStore) ListUserCallRecords(ctx context.Context, userID int64) ([]model.UserCallRecord, error) {
	if storeValue.failCallRecords != nil {
		return nil, storeValue.failCallRecords
	}
	return storeValue.CommunicationStore.ListUserCallRecords(ctx, userID)
}
func (storeValue branchAwareCommunicationStore) ListUserMessages(ctx context.Context, userID int64) ([]model.UserMessage, error) {
	if storeValue.failMessages != nil {
		return nil, storeValue.failMessages
	}
	return storeValue.CommunicationStore.ListUserMessages(ctx, userID)
}
