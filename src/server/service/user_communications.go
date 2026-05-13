package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/casapps/caspbx/src/server/model"
	"github.com/casapps/caspbx/src/server/store"
)

type UserCommunicationsError struct {
	Code    string
	Message string
}

type UserCommunicationSurface struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Path  string `json:"path"`
}

type UserCommunicationDashboard struct {
	UserID          int64                      `json:"user_id"`
	Username        string                     `json:"username"`
	DisplayName     string                     `json:"display_name"`
	ExtensionNumber string                     `json:"extension_number,omitempty"`
	Technology      string                     `json:"technology,omitempty"`
	ContactCount    int                        `json:"contact_count"`
	VoicemailCount  int                        `json:"voicemail_count"`
	RecentCallCount int                        `json:"recent_call_count"`
	MessageCount    int                        `json:"message_count"`
	VisibleSurfaces []UserCommunicationSurface `json:"visible_surfaces"`
	Warnings        []string                   `json:"warnings,omitempty"`
}

type UserPresenceView struct {
	ExtensionNumber  string   `json:"extension_number,omitempty"`
	Status           string   `json:"status"`
	PresenceEnabled  bool     `json:"presence_enabled"`
	MessagingEnabled bool     `json:"messaging_enabled"`
	Transport        string   `json:"transport,omitempty"`
	Capabilities     []string `json:"capabilities,omitempty"`
}

type UserWebphoneView struct {
	ExtensionNumber string   `json:"extension_number,omitempty"`
	Technology      string   `json:"technology,omitempty"`
	Endpoint        string   `json:"endpoint,omitempty"`
	Enabled         bool     `json:"enabled"`
	SecureTransport bool     `json:"secure_transport"`
	Capabilities    []string `json:"capabilities,omitempty"`
}

type UserCommunicationsService struct {
	userStore          store.UserCredentialStore
	pbxStore           store.PBXStore
	asteriskStore      store.AsteriskStore
	communicationStore store.CommunicationStore
	now                func() time.Time
}

type userCommunicationContext struct {
	user      model.User
	settings  model.UserCommunicationSettings
	extension model.Extension
	hasExt    bool
	state     model.AsteriskState
}

func (errorValue *UserCommunicationsError) Error() string {
	if errorValue == nil {
		return ""
	}
	return errorValue.Message
}

func NewUserCommunicationsService(userStore store.UserCredentialStore, pbxStore store.PBXStore, asteriskStore store.AsteriskStore, communicationStore store.CommunicationStore) UserCommunicationsService {
	return UserCommunicationsService{
		userStore:          userStore,
		pbxStore:           pbxStore,
		asteriskStore:      asteriskStore,
		communicationStore: communicationStore,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (service UserCommunicationsService) Dashboard(ctx context.Context, userID int64) (UserCommunicationDashboard, error) {
	commCtx, err := service.loadContext(ctx, userID)
	if err != nil {
		return UserCommunicationDashboard{}, err
	}
	contacts, err := service.communicationStore.ListUserContacts(ctx, userID)
	if err != nil {
		return UserCommunicationDashboard{}, err
	}
	voicemails, err := service.communicationStore.ListUserVoicemails(ctx, userID)
	if err != nil {
		return UserCommunicationDashboard{}, err
	}
	callRecords, err := service.communicationStore.ListUserCallRecords(ctx, userID)
	if err != nil {
		return UserCommunicationDashboard{}, err
	}
	messages, err := service.communicationStore.ListUserMessages(ctx, userID)
	if err != nil {
		return UserCommunicationDashboard{}, err
	}

	dashboard := UserCommunicationDashboard{
		UserID:          commCtx.user.ID,
		Username:        commCtx.user.Username,
		DisplayName:     commCtx.user.DisplayName,
		ContactCount:    len(contacts),
		VoicemailCount:  len(voicemails),
		RecentCallCount: len(callRecords),
		MessageCount:    len(messages),
		VisibleSurfaces: visibleUserCommunicationSurfaces(commCtx),
	}
	if commCtx.hasExt {
		dashboard.ExtensionNumber = commCtx.extension.Number
		dashboard.Technology = commCtx.extension.Technology
	} else {
		dashboard.Warnings = append(dashboard.Warnings, "No PBX extension is assigned to this user yet.")
	}
	if !supportsTelephony(commCtx.state) {
		dashboard.Warnings = append(dashboard.Warnings, "Telephony drivers are not currently available, so live calling surfaces stay hidden.")
	}
	return dashboard, nil
}

func (service UserCommunicationsService) ListContacts(ctx context.Context, userID int64) ([]model.UserContact, error) {
	if _, err := service.loadContext(ctx, userID); err != nil {
		return nil, err
	}
	return service.communicationStore.ListUserContacts(ctx, userID)
}

func (service UserCommunicationsService) GetContact(ctx context.Context, userID int64, contactID int64) (model.UserContact, error) {
	if _, err := service.loadContext(ctx, userID); err != nil {
		return model.UserContact{}, err
	}
	contact, err := service.communicationStore.FindUserContact(ctx, userID, contactID)
	if errors.Is(err, store.ErrNotFound) {
		return model.UserContact{}, notFoundUserCommunicationsError("contact")
	}
	return contact, err
}

func (service UserCommunicationsService) CreateContact(ctx context.Context, userID int64, contact model.UserContact) (model.UserContact, error) {
	if _, err := service.loadContext(ctx, userID); err != nil {
		return model.UserContact{}, err
	}
	contact.UserID = userID
	contact = model.NormalizeUserContact(contact)
	if strings.TrimSpace(contact.DisplayName) == "" {
		return model.UserContact{}, invalidUserCommunicationsError("contact display_name is required")
	}
	if contact.Email != "" {
		if err := model.ValidateEmail(contact.Email); err != nil {
			return model.UserContact{}, invalidUserCommunicationsError("contact email must be valid")
		}
	}
	if contact.ExtensionNumber == "" && contact.PhoneNumber == "" && contact.Email == "" {
		return model.UserContact{}, invalidUserCommunicationsError("contact must include an extension_number, phone_number, or email")
	}
	contact.CreatedAt = service.now()
	contact.UpdatedAt = contact.CreatedAt
	return service.communicationStore.SaveUserContact(ctx, contact)
}

func (service UserCommunicationsService) DeleteContact(ctx context.Context, userID int64, contactID int64) error {
	if _, err := service.loadContext(ctx, userID); err != nil {
		return err
	}
	err := service.communicationStore.DeleteUserContact(ctx, userID, contactID)
	if errors.Is(err, store.ErrNotFound) {
		return notFoundUserCommunicationsError("contact")
	}
	return err
}

func (service UserCommunicationsService) ListVoicemails(ctx context.Context, userID int64) ([]model.UserVoicemail, error) {
	commCtx, err := service.loadContext(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !supportsVoicemail(commCtx) {
		return nil, unavailableUserCommunicationsError("voicemail")
	}
	return service.communicationStore.ListUserVoicemails(ctx, userID)
}

func (service UserCommunicationsService) ListCallHistory(ctx context.Context, userID int64) ([]model.UserCallRecord, error) {
	commCtx, err := service.loadContext(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !commCtx.hasExt || !supportsTelephony(commCtx.state) {
		return nil, unavailableUserCommunicationsError("call-history")
	}
	return service.communicationStore.ListUserCallRecords(ctx, userID)
}

func (service UserCommunicationsService) ListMessages(ctx context.Context, userID int64) ([]model.UserMessage, error) {
	commCtx, err := service.loadContext(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !supportsMessaging(commCtx.state) || !commCtx.settings.MessagingEnabled {
		return nil, unavailableUserCommunicationsError("messages")
	}
	return service.communicationStore.ListUserMessages(ctx, userID)
}

func (service UserCommunicationsService) Presence(ctx context.Context, userID int64) (UserPresenceView, error) {
	commCtx, err := service.loadContext(ctx, userID)
	if err != nil {
		return UserPresenceView{}, err
	}
	if !supportsMessaging(commCtx.state) {
		return UserPresenceView{}, unavailableUserCommunicationsError("presence")
	}
	view := UserPresenceView{
		Status:           "available",
		PresenceEnabled:  commCtx.settings.PresenceEnabled,
		MessagingEnabled: commCtx.settings.MessagingEnabled,
		Capabilities:     messagingCapabilities(commCtx.state),
		Transport:        messagingTransport(commCtx.state),
	}
	if commCtx.settings.DoNotDisturb {
		view.Status = "do_not_disturb"
	}
	if commCtx.hasExt {
		view.ExtensionNumber = commCtx.extension.Number
	}
	return view, nil
}

func (service UserCommunicationsService) Webphone(ctx context.Context, userID int64) (UserWebphoneView, error) {
	commCtx, err := service.loadContext(ctx, userID)
	if err != nil {
		return UserWebphoneView{}, err
	}
	if !supportsWebphone(commCtx) {
		return UserWebphoneView{}, unavailableUserCommunicationsError("webphone")
	}
	return UserWebphoneView{
		ExtensionNumber: commCtx.extension.Number,
		Technology:      commCtx.extension.Technology,
		Endpoint:        firstNonEmptyString(commCtx.settings.PreferredEndpoint, commCtx.extension.Endpoint),
		Enabled:         commCtx.settings.WebphoneEnabled,
		SecureTransport: hasAvailableCapability(commCtx.state, "tls"),
		Capabilities:    []string{"browser_calling"},
	}, nil
}

func (service UserCommunicationsService) GetSettings(ctx context.Context, userID int64) (model.UserCommunicationSettings, error) {
	commCtx, err := service.loadContext(ctx, userID)
	if err != nil {
		return model.UserCommunicationSettings{}, err
	}
	return commCtx.settings, nil
}

func (service UserCommunicationsService) UpdateSettings(ctx context.Context, userID int64, settings model.UserCommunicationSettings) (model.UserCommunicationSettings, error) {
	commCtx, err := service.loadContext(ctx, userID)
	if err != nil {
		return model.UserCommunicationSettings{}, err
	}
	merged := commCtx.settings
	merged.DoNotDisturb = settings.DoNotDisturb
	merged.CallForwardingTarget = strings.TrimSpace(settings.CallForwardingTarget)
	merged.VoicemailEnabled = settings.VoicemailEnabled
	merged.WebphoneEnabled = settings.WebphoneEnabled
	merged.PresenceEnabled = settings.PresenceEnabled
	merged.MessagingEnabled = settings.MessagingEnabled
	merged.PreferredEndpoint = strings.TrimSpace(settings.PreferredEndpoint)
	merged.PreferredContactEmail = model.NormalizeEmail(settings.PreferredContactEmail)
	if merged.PreferredContactEmail != "" {
		if err := model.ValidateEmail(merged.PreferredContactEmail); err != nil {
			return model.UserCommunicationSettings{}, invalidUserCommunicationsError("preferred_contact_email must be valid")
		}
	}
	if merged.WebphoneEnabled && !supportsWebphone(commCtx) {
		return model.UserCommunicationSettings{}, invalidUserCommunicationsError("webphone is unavailable for this user")
	}
	if merged.PresenceEnabled && !supportsMessaging(commCtx.state) {
		return model.UserCommunicationSettings{}, invalidUserCommunicationsError("presence is unavailable for this user")
	}
	if merged.MessagingEnabled && !supportsMessaging(commCtx.state) {
		return model.UserCommunicationSettings{}, invalidUserCommunicationsError("messaging is unavailable for this user")
	}
	if merged.VoicemailEnabled && !supportsVoicemail(commCtx) {
		return model.UserCommunicationSettings{}, invalidUserCommunicationsError("voicemail is unavailable for this user")
	}
	merged.UserID = userID
	if commCtx.hasExt {
		merged.ExtensionID = commCtx.extension.ID
		if merged.PreferredEndpoint == "" {
			merged.PreferredEndpoint = commCtx.extension.Endpoint
		}
	}
	merged.UpdatedAt = service.now()
	return service.communicationStore.SaveUserCommunicationSettings(ctx, merged)
}

func (service UserCommunicationsService) loadContext(ctx context.Context, userID int64) (userCommunicationContext, error) {
	user, err := service.userStore.FindUserByID(ctx, userID)
	if errors.Is(err, store.ErrNotFound) {
		return userCommunicationContext{}, notFoundUserCommunicationsError("user")
	}
	if err != nil {
		return userCommunicationContext{}, err
	}
	plan, err := service.pbxStore.GetPBXPlan(ctx)
	if err != nil {
		return userCommunicationContext{}, err
	}
	state, err := service.asteriskStore.GetAsteriskState(ctx)
	if err != nil {
		return userCommunicationContext{}, err
	}
	settings, settingsErr := service.communicationStore.FindUserCommunicationSettings(ctx, userID)
	if settingsErr != nil && !errors.Is(settingsErr, store.ErrNotFound) {
		return userCommunicationContext{}, settingsErr
	}
	if errors.Is(settingsErr, store.ErrNotFound) {
		settings = model.DefaultUserCommunicationSettings(userID)
	}
	settings = model.NormalizeUserCommunicationSettings(settings)

	extension, hasExtension := findUserExtension(plan, userID)
	if hasExtension {
		settings.ExtensionID = extension.ID
		if settings.PreferredEndpoint == "" {
			settings.PreferredEndpoint = extension.Endpoint
		}
		if !extension.VoicemailEnabled {
			settings.VoicemailEnabled = false
		}
	}
	if !supportsMessaging(state) {
		settings.MessagingEnabled = false
		settings.PresenceEnabled = false
	}
	if !hasAvailableCapability(state, "browser_calling") {
		settings.WebphoneEnabled = false
	}
	return userCommunicationContext{
		user:      user,
		settings:  settings,
		extension: extension,
		hasExt:    hasExtension,
		state:     state,
	}, nil
}

func visibleUserCommunicationSurfaces(commCtx userCommunicationContext) []UserCommunicationSurface {
	surfaces := []UserCommunicationSurface{
		{Key: "dashboard", Label: "Dashboard", Path: "dashboard"},
		{Key: "contacts", Label: "Contacts", Path: "contacts"},
		{Key: "settings", Label: "Communications Settings", Path: "communications/settings"},
	}
	if commCtx.hasExt && supportsTelephony(commCtx.state) {
		surfaces = append(surfaces, UserCommunicationSurface{Key: "call-history", Label: "Call History", Path: "call-history"})
	}
	if supportsVoicemail(commCtx) {
		surfaces = append(surfaces, UserCommunicationSurface{Key: "voicemail", Label: "Voicemail", Path: "voicemail"})
	}
	if supportsMessaging(commCtx.state) {
		surfaces = append(surfaces,
			UserCommunicationSurface{Key: "messages", Label: "Messages", Path: "messages"},
			UserCommunicationSurface{Key: "presence", Label: "Presence", Path: "presence"},
		)
	}
	if supportsWebphone(commCtx) {
		surfaces = append(surfaces, UserCommunicationSurface{Key: "webphone", Label: "Webphone", Path: "webphone"})
	}
	return surfaces
}

func findUserExtension(plan model.PBXPlan, userID int64) (model.Extension, bool) {
	for _, extension := range plan.Extensions {
		if extension.UserID == userID {
			return extension, true
		}
	}
	return model.Extension{}, false
}

func supportsTelephony(state model.AsteriskState) bool {
	return len(state.ChannelDrivers) > 0 || len(state.EndpointStacks) > 0
}

func supportsVoicemail(commCtx userCommunicationContext) bool {
	return commCtx.hasExt && commCtx.extension.VoicemailEnabled && hasAnyAvailableCapability(commCtx.state, "voicemail", "recordings")
}

func supportsMessaging(state model.AsteriskState) bool {
	return hasAnyAvailableCapability(state, "xmpp", "presence", "mail_delivery") || hasUserCommunicationSubsystem(state, "messaging_backend")
}

func supportsWebphone(commCtx userCommunicationContext) bool {
	return commCtx.hasExt && supportsTelephony(commCtx.state) && hasAvailableCapability(commCtx.state, "browser_calling")
}

func hasAvailableCapability(state model.AsteriskState, key string) bool {
	capability, found := state.Capability(key)
	return found && capability.Available
}

func hasAnyAvailableCapability(state model.AsteriskState, keys ...string) bool {
	for _, key := range keys {
		if hasAvailableCapability(state, key) {
			return true
		}
	}
	return false
}

func hasUserCommunicationSubsystem(state model.AsteriskState, key string) bool {
	normalized := strings.TrimSpace(strings.ToLower(key))
	for _, subsystem := range state.Subsystems {
		if subsystem.Key == normalized && strings.TrimSpace(subsystem.Provider) != "" {
			return true
		}
	}
	return false
}

func messagingCapabilities(state model.AsteriskState) []string {
	result := []string{}
	for _, key := range []string{"presence", "xmpp", "mail_delivery"} {
		if hasAvailableCapability(state, key) {
			result = append(result, key)
		}
	}
	if hasUserCommunicationSubsystem(state, "messaging_backend") {
		result = append(result, "messaging_backend")
	}
	return result
}

func messagingTransport(state model.AsteriskState) string {
	switch {
	case hasAvailableCapability(state, "xmpp"):
		return "xmpp"
	case hasAvailableCapability(state, "mail_delivery"):
		return "mail"
	case hasUserCommunicationSubsystem(state, "messaging_backend"):
		return "backend"
	default:
		return ""
	}
}

func invalidUserCommunicationsError(message string) error {
	return &UserCommunicationsError{Code: "COMMUNICATION_INVALID", Message: message}
}

func notFoundUserCommunicationsError(resource string) error {
	return &UserCommunicationsError{Code: "COMMUNICATION_NOT_FOUND", Message: fmt.Sprintf("%s not found", resource)}
}

func unavailableUserCommunicationsError(surface string) error {
	return &UserCommunicationsError{Code: "COMMUNICATION_UNAVAILABLE", Message: fmt.Sprintf("%s is not available for this deployment", surface)}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
