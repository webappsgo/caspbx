package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/casapps/caspbx/src/server/model"
	"github.com/casapps/caspbx/src/server/service"
)

func TestUserCommunicationHandlers(t *testing.T) {
	memoryStore, authService := newTestRuntimeStore(t)
	seedHandlerUserCommunications(t, memoryStore)
	communicationService := service.NewUserCommunicationsService(memoryStore, memoryStore, memoryStore, memoryStore)
	userCookie := SessionCookieConfig{Name: "user_session", Path: "/", HTTPOnly: true, Secure: "auto", SameSite: http.SameSiteLaxMode}

	authHandler := NewAuthHandler("/auth", authService, userCookie, model.RegistrationModePrivate)
	userHandler := NewUserHandler("/users", authService, testDomainService(), userCookie, communicationService)
	apiUserHandler := NewAPIUserHandler("/api/v1/users", authService, testDomainService(), communicationService)

	loginRequest := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader("identifier=alice&password=correct+horse+battery+staple"))
	loginRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginResponse := httptest.NewRecorder()
	authHandler.ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d", loginResponse.Code)
	}
	userSessionCookie := loginResponse.Result().Cookies()[0]

	dashboardRequest := httptest.NewRequest(http.MethodGet, "/users/dashboard", nil)
	dashboardRequest.AddCookie(userSessionCookie)
	dashboardRequest.Header.Set("Accept", "application/json")
	dashboardResponse := httptest.NewRecorder()
	userHandler.ServeHTTP(dashboardResponse, dashboardRequest)
	if !strings.Contains(dashboardResponse.Body.String(), "\"extension_number\":\"1000\"") {
		t.Fatalf("unexpected dashboard response %q", dashboardResponse.Body.String())
	}

	contactRequest := httptest.NewRequest(http.MethodPost, "/users/contacts", strings.NewReader(`{"display_name":"Carol Example","phone_number":"18005551212"}`))
	contactRequest.AddCookie(userSessionCookie)
	contactRequest.Header.Set("Content-Type", "application/json")
	contactRequest.Header.Set("Accept", "application/json")
	contactResponse := httptest.NewRecorder()
	userHandler.ServeHTTP(contactResponse, contactRequest)
	if contactResponse.Code != http.StatusCreated || !strings.Contains(contactResponse.Body.String(), "\"display_name\":\"Carol Example\"") {
		t.Fatalf("unexpected create contact response %d %q", contactResponse.Code, contactResponse.Body.String())
	}

	voicemailRequest := httptest.NewRequest(http.MethodGet, "/users/voicemail", nil)
	voicemailRequest.AddCookie(userSessionCookie)
	voicemailResponse := httptest.NewRecorder()
	userHandler.ServeHTTP(voicemailResponse, voicemailRequest)
	if !strings.Contains(voicemailResponse.Body.String(), "Surface: voicemail") {
		t.Fatalf("unexpected voicemail text response %q", voicemailResponse.Body.String())
	}

	settingsRequest := httptest.NewRequest(http.MethodPost, "/users/communications/settings", strings.NewReader("do_not_disturb=true&voicemail_enabled=true&webphone_enabled=true&presence_enabled=true&messaging_enabled=true"))
	settingsRequest.AddCookie(userSessionCookie)
	settingsRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	settingsRequest.Header.Set("Accept", "application/json")
	settingsResponse := httptest.NewRecorder()
	userHandler.ServeHTTP(settingsResponse, settingsRequest)
	if settingsResponse.Code != http.StatusOK || !strings.Contains(settingsResponse.Body.String(), "\"do_not_disturb\":true") {
		t.Fatalf("unexpected settings response %d %q", settingsResponse.Code, settingsResponse.Body.String())
	}

	badContactRequest := httptest.NewRequest(http.MethodPost, "/users/contacts", strings.NewReader("{"))
	badContactRequest.AddCookie(userSessionCookie)
	badContactRequest.Header.Set("Content-Type", "application/json")
	badContactResponse := httptest.NewRecorder()
	userHandler.ServeHTTP(badContactResponse, badContactRequest)
	if badContactResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected bad contact status 400, got %d", badContactResponse.Code)
	}

	deleteCollectionRequest := httptest.NewRequest(http.MethodDelete, "/users/contacts", nil)
	deleteCollectionRequest.AddCookie(userSessionCookie)
	deleteCollectionResponse := httptest.NewRecorder()
	userHandler.ServeHTTP(deleteCollectionResponse, deleteCollectionRequest)
	if deleteCollectionResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected collection delete status 405, got %d", deleteCollectionResponse.Code)
	}

	_, userToken, tokenError := authService.AuthenticateUserAPI(context.Background(), "alice", "correct horse battery staple")
	if tokenError != nil {
		t.Fatalf("authenticate api user: %v", tokenError)
	}

	apiDashboardRequest := httptest.NewRequest(http.MethodGet, "/api/v1/users/dashboard", nil)
	apiDashboardRequest.Header.Set("Authorization", "Bearer "+userToken.Value)
	apiDashboardResponse := httptest.NewRecorder()
	apiUserHandler.ServeHTTP(apiDashboardResponse, apiDashboardRequest)
	if !strings.Contains(apiDashboardResponse.Body.String(), "\"surface\":\"dashboard\"") {
		t.Fatalf("unexpected api dashboard response %q", apiDashboardResponse.Body.String())
	}

	apiMessagesRequest := httptest.NewRequest(http.MethodGet, "/api/v1/users/messages", nil)
	apiMessagesRequest.Header.Set("Authorization", "Bearer "+userToken.Value)
	apiMessagesResponse := httptest.NewRecorder()
	apiUserHandler.ServeHTTP(apiMessagesResponse, apiMessagesRequest)
	if !strings.Contains(apiMessagesResponse.Body.String(), "\"body\":\"Need help\"") {
		t.Fatalf("unexpected api messages response %q", apiMessagesResponse.Body.String())
	}

	apiContactRequest := httptest.NewRequest(http.MethodPost, "/api/v1/users/contacts", strings.NewReader("display_name=Desk+Phone&extension_number=1001"))
	apiContactRequest.Header.Set("Authorization", "Bearer "+userToken.Value)
	apiContactRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	apiContactResponse := httptest.NewRecorder()
	apiUserHandler.ServeHTTP(apiContactResponse, apiContactRequest)
	if apiContactResponse.Code != http.StatusCreated {
		t.Fatalf("expected api create contact status 201, got %d", apiContactResponse.Code)
	}

	apiMissingContactRequest := httptest.NewRequest(http.MethodGet, "/api/v1/users/contacts/999", nil)
	apiMissingContactRequest.Header.Set("Authorization", "Bearer "+userToken.Value)
	apiMissingContactResponse := httptest.NewRecorder()
	apiUserHandler.ServeHTTP(apiMissingContactResponse, apiMissingContactRequest)
	if apiMissingContactResponse.Code != http.StatusNotFound {
		t.Fatalf("expected api missing contact status 404, got %d", apiMissingContactResponse.Code)
	}

	apiSettingsRequest := httptest.NewRequest(http.MethodPost, "/api/v1/users/communications/settings", strings.NewReader(`{"do_not_disturb":false,"voicemail_enabled":true,"webphone_enabled":true,"presence_enabled":true,"messaging_enabled":true,"preferred_contact_email":"notify@example.com"}`))
	apiSettingsRequest.Header.Set("Authorization", "Bearer "+userToken.Value)
	apiSettingsRequest.Header.Set("Content-Type", "application/json")
	apiSettingsResponse := httptest.NewRecorder()
	apiUserHandler.ServeHTTP(apiSettingsResponse, apiSettingsRequest)
	if apiSettingsResponse.Code != http.StatusOK || !strings.Contains(apiSettingsResponse.Body.String(), "\"preferred_contact_email\":\"notify@example.com\"") {
		t.Fatalf("unexpected api settings response %d %q", apiSettingsResponse.Code, apiSettingsResponse.Body.String())
	}
}

func TestUserCommunicationHelpers(t *testing.T) {
	if route, ok := parseUserCommunicationRoute("contacts/5"); !ok || !route.hasID || route.id != 5 {
		t.Fatalf("expected parsed contact detail route, got %+v / %t", route, ok)
	}
	if route, ok := parseUserCommunicationRoute("communications/settings"); !ok || route.surface != "communications/settings" {
		t.Fatalf("expected parsed settings route, got %+v / %t", route, ok)
	}
	if _, ok := parseUserCommunicationRoute("missing"); ok {
		t.Fatalf("expected missing route parse failure")
	}

	jsonContactRequest := httptest.NewRequest(http.MethodPost, "/users/contacts", strings.NewReader(`{"display_name":"Carol","favorite":true}`))
	jsonContactRequest.Header.Set("Content-Type", "application/json")
	contact, err := readUserContactRequest(jsonContactRequest)
	if err != nil || !contact.Favorite || contact.DisplayName != "Carol" {
		t.Fatalf("expected parsed json contact, got %v / %+v", err, contact)
	}

	formContactRequest := httptest.NewRequest(http.MethodPost, "/users/contacts", strings.NewReader("display_name=Desk+Phone&favorite=yes"))
	formContactRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	contact, err = readUserContactRequest(formContactRequest)
	if err != nil || !contact.Favorite || contact.DisplayName != "Desk Phone" {
		t.Fatalf("expected parsed form contact, got %v / %+v", err, contact)
	}

	badContactRequest := httptest.NewRequest(http.MethodPost, "/users/contacts", nil)
	badContactRequest.Body = errReadCloser{}
	badContactRequest.Header.Set("Content-Type", "application/json")
	if _, err := readUserContactRequest(badContactRequest); err == nil {
		t.Fatalf("expected json contact parse error")
	}

	jsonSettingsRequest := httptest.NewRequest(http.MethodPost, "/users/communications/settings", strings.NewReader(`{"do_not_disturb":true}`))
	jsonSettingsRequest.Header.Set("Content-Type", "application/json")
	settings, err := readUserCommunicationSettingsRequest(jsonSettingsRequest)
	if err != nil || !settings.DoNotDisturb {
		t.Fatalf("expected parsed json settings, got %v / %+v", err, settings)
	}

	formSettingsRequest := httptest.NewRequest(http.MethodPost, "/users/communications/settings", strings.NewReader("do_not_disturb=on&presence_enabled=yes"))
	formSettingsRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	settings, err = readUserCommunicationSettingsRequest(formSettingsRequest)
	if err != nil || !settings.DoNotDisturb || !settings.PresenceEnabled {
		t.Fatalf("expected parsed form settings, got %v / %+v", err, settings)
	}

	badSettingsRequest := httptest.NewRequest(http.MethodPost, "/users/communications/settings", nil)
	badSettingsRequest.Body = errReadCloser{}
	badSettingsRequest.Header.Set("Content-Type", "application/json")
	if _, err := readUserCommunicationSettingsRequest(badSettingsRequest); err == nil {
		t.Fatalf("expected json settings parse error")
	}

	response := httptest.NewRecorder()
	writeUserCommunicationsError(response, &service.UserCommunicationsError{Code: "COMMUNICATION_INVALID", Message: "bad request"})
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid error mapping, got %d", response.Code)
	}

	response = httptest.NewRecorder()
	writeUserCommunicationsError(response, &service.UserCommunicationsError{Code: "COMMUNICATION_UNAVAILABLE", Message: "hidden"})
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected unavailable error mapping, got %d", response.Code)
	}

	response = httptest.NewRecorder()
	writeUserCommunicationsError(response, &service.UserCommunicationsError{Code: "COMMUNICATION_UNKNOWN", Message: "boom"})
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected unknown error mapping, got %d", response.Code)
	}

	response = httptest.NewRecorder()
	writeUserCommunicationsError(response, context.Canceled)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected generic error mapping, got %d", response.Code)
	}

	response = httptest.NewRecorder()
	writeUserCommunicationText(response, "alice", "dashboard", map[string]string{"status": "ok"})
	if !strings.Contains(response.Body.String(), "Surface: dashboard") {
		t.Fatalf("unexpected text response %q", response.Body.String())
	}

	formBoolRequest := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("favorite=yes"))
	formBoolRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if !formBool(formBoolRequest, "favorite") {
		t.Fatalf("expected truthy form bool")
	}
	if camelKey("preferred_contact_email") != "preferredContactEmail" || camelKey("a__b") != "aB" {
		t.Fatalf("unexpected camel key conversion")
	}
	if formBool(httptest.NewRequest(http.MethodPost, "/", nil), "favorite") {
		t.Fatalf("expected false form bool default")
	}
}

func TestUserCommunicationDirectBranches(t *testing.T) {
	memoryStore, authService := newTestRuntimeStore(t)
	seedHandlerUserCommunications(t, memoryStore)
	communicationService := service.NewUserCommunicationsService(memoryStore, memoryStore, memoryStore, memoryStore)
	userCookie := SessionCookieConfig{Name: "user_session", Path: "/", HTTPOnly: true, Secure: "auto", SameSite: http.SameSiteLaxMode}
	userHandler := NewUserHandler("/users", authService, testDomainService(), userCookie, communicationService)
	apiUserHandler := NewAPIUserHandler("/api/v1/users", authService, testDomainService(), communicationService)

	loginRequest := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader("identifier=alice&password=correct+horse+battery+staple"))
	loginRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginResponse := httptest.NewRecorder()
	NewAuthHandler("/auth", authService, userCookie, model.RegistrationModePrivate).ServeHTTP(loginResponse, loginRequest)
	userSessionCookie := loginResponse.Result().Cookies()[0]
	_, userToken, _ := authService.AuthenticateUserAPI(context.Background(), "alice", "correct horse battery staple")

	for _, tc := range []struct {
		method  string
		path    string
		useAPI  bool
		accept  string
		body    string
		ct      string
		want    int
	}{
		{method: http.MethodPost, path: "/users/dashboard", want: http.StatusMethodNotAllowed},
		{method: http.MethodGet, path: "/users/contacts", accept: "application/json", want: http.StatusOK},
		{method: http.MethodGet, path: "/users/contacts/1", want: http.StatusOK},
		{method: http.MethodPost, path: "/users/contacts/1", want: http.StatusMethodNotAllowed},
		{method: http.MethodDelete, path: "/users/contacts/1", want: http.StatusOK},
		{method: http.MethodPut, path: "/users/contacts/1", want: http.StatusMethodNotAllowed},
		{method: http.MethodGet, path: "/users/communications/settings", want: http.StatusOK},
		{method: http.MethodGet, path: "/users/communications/settings", accept: "application/json", want: http.StatusOK},
		{method: http.MethodPost, path: "/users/communications/settings", body: "{", ct: "application/json", want: http.StatusBadRequest},
		{method: http.MethodPost, path: "/users/communications/settings", body: `{"preferred_contact_email":"bad"}`, ct: "application/json", accept: "application/json", want: http.StatusBadRequest},
		{method: http.MethodDelete, path: "/users/communications/settings", want: http.StatusMethodNotAllowed},
		{method: http.MethodPost, path: "/api/v1/users/dashboard", useAPI: true, want: http.StatusMethodNotAllowed},
		{method: http.MethodGet, path: "/api/v1/users/contacts", useAPI: true, want: http.StatusOK},
		{method: http.MethodPost, path: "/api/v1/users/contacts", useAPI: true, body: "{", ct: "application/json", want: http.StatusBadRequest},
		{method: http.MethodPost, path: "/api/v1/users/contacts/1", useAPI: true, want: http.StatusMethodNotAllowed},
		{method: http.MethodDelete, path: "/api/v1/users/contacts/1", useAPI: true, want: http.StatusNotFound},
		{method: http.MethodDelete, path: "/api/v1/users/contacts", useAPI: true, want: http.StatusMethodNotAllowed},
		{method: http.MethodPut, path: "/api/v1/users/contacts/1", useAPI: true, want: http.StatusMethodNotAllowed},
		{method: http.MethodGet, path: "/api/v1/users/communications/settings", useAPI: true, want: http.StatusOK},
		{method: http.MethodPost, path: "/api/v1/users/communications/settings", useAPI: true, body: "{", ct: "application/json", want: http.StatusBadRequest},
		{method: http.MethodPost, path: "/api/v1/users/communications/settings", useAPI: true, body: `{"preferred_contact_email":"bad"}`, ct: "application/json", want: http.StatusBadRequest},
		{method: http.MethodDelete, path: "/api/v1/users/communications/settings", useAPI: true, want: http.StatusMethodNotAllowed},
	} {
		request := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		if tc.ct != "" {
			request.Header.Set("Content-Type", tc.ct)
		}
		if tc.accept != "" {
			request.Header.Set("Accept", tc.accept)
		}
		response := httptest.NewRecorder()
		if tc.useAPI {
			request.Header.Set("Authorization", "Bearer "+userToken.Value)
			apiUserHandler.ServeHTTP(response, request)
		} else {
			request.AddCookie(userSessionCookie)
			userHandler.ServeHTTP(response, request)
		}
		if response.Code != tc.want {
			t.Fatalf("%s %s: expected %d, got %d (%q)", tc.method, tc.path, tc.want, response.Code, response.Body.String())
		}
	}

	if _, err := lookupUserCommunicationSurface(context.Background(), communicationService, userCommunicationRoute{surface: "missing"}, 1); err == nil {
		t.Fatalf("expected unknown surface lookup error")
	}
	if _, err := lookupUserCommunicationSurface(context.Background(), communicationService, userCommunicationRoute{surface: "call-history"}, 1); err != nil {
		t.Fatalf("expected call-history lookup, got %v", err)
	}
	if _, err := lookupUserCommunicationSurface(context.Background(), communicationService, userCommunicationRoute{surface: "communications/settings"}, 1); err != nil {
		t.Fatalf("expected settings lookup, got %v", err)
	}

	formParseRequest := httptest.NewRequest(http.MethodPost, "/users/contacts", nil)
	formParseRequest.Body = errReadCloser{}
	formParseRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if _, err := readUserContactRequest(formParseRequest); err == nil {
		t.Fatalf("expected form contact parse error")
	}

	formSettingsParseRequest := httptest.NewRequest(http.MethodPost, "/users/communications/settings", nil)
	formSettingsParseRequest.Body = errReadCloser{}
	formSettingsParseRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if _, err := readUserCommunicationSettingsRequest(formSettingsParseRequest); err == nil {
		t.Fatalf("expected form settings parse error")
	}

	deleteMissingRequest := httptest.NewRequest(http.MethodDelete, "/users/contacts/999", nil)
	deleteMissingRequest.AddCookie(userSessionCookie)
	deleteMissingResponse := httptest.NewRecorder()
	userHandler.ServeHTTP(deleteMissingResponse, deleteMissingRequest)
	if deleteMissingResponse.Code != http.StatusNotFound {
		t.Fatalf("expected missing web contact delete to return 404, got %d", deleteMissingResponse.Code)
	}

	createTextRequest := httptest.NewRequest(http.MethodPost, "/users/contacts", strings.NewReader("display_name=Text+Contact&phone_number=18005550000"))
	createTextRequest.AddCookie(userSessionCookie)
	createTextRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	createTextResponse := httptest.NewRecorder()
	userHandler.ServeHTTP(createTextResponse, createTextRequest)
	if createTextResponse.Code != http.StatusOK || !strings.Contains(createTextResponse.Body.String(), "Text Contact") {
		t.Fatalf("unexpected text contact create response %d %q", createTextResponse.Code, createTextResponse.Body.String())
	}

	createErrorRequest := httptest.NewRequest(http.MethodPost, "/users/contacts", strings.NewReader(`{"display_name":"No Route"}`))
	createErrorRequest.AddCookie(userSessionCookie)
	createErrorRequest.Header.Set("Content-Type", "application/json")
	createErrorResponse := httptest.NewRecorder()
	userHandler.ServeHTTP(createErrorResponse, createErrorRequest)
	if createErrorResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected web contact validation error, got %d", createErrorResponse.Code)
	}

	apiCreateErrorRequest := httptest.NewRequest(http.MethodPost, "/api/v1/users/contacts", strings.NewReader(`{"display_name":"No Route"}`))
	apiCreateErrorRequest.Header.Set("Authorization", "Bearer "+userToken.Value)
	apiCreateErrorRequest.Header.Set("Content-Type", "application/json")
	apiCreateErrorResponse := httptest.NewRecorder()
	apiUserHandler.ServeHTTP(apiCreateErrorResponse, apiCreateErrorRequest)
	if apiCreateErrorResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected api contact validation error, got %d", apiCreateErrorResponse.Code)
	}

	apiDeleteCreateRequest := httptest.NewRequest(http.MethodPost, "/api/v1/users/contacts", strings.NewReader(`{"display_name":"Delete Me","phone_number":"18005551212"}`))
	apiDeleteCreateRequest.Header.Set("Authorization", "Bearer "+userToken.Value)
	apiDeleteCreateRequest.Header.Set("Content-Type", "application/json")
	apiDeleteCreateResponse := httptest.NewRecorder()
	apiUserHandler.ServeHTTP(apiDeleteCreateResponse, apiDeleteCreateRequest)
	if apiDeleteCreateResponse.Code != http.StatusCreated {
		t.Fatalf("expected api create before delete, got %d", apiDeleteCreateResponse.Code)
	}
	apiDeleteRequest := httptest.NewRequest(http.MethodDelete, "/api/v1/users/contacts/3", nil)
	apiDeleteRequest.Header.Set("Authorization", "Bearer "+userToken.Value)
	apiDeleteResponse := httptest.NewRecorder()
	apiUserHandler.ServeHTTP(apiDeleteResponse, apiDeleteRequest)
	if apiDeleteResponse.Code != http.StatusOK {
		t.Fatalf("expected api contact delete success, got %d", apiDeleteResponse.Code)
	}

	settingsTextRequest := httptest.NewRequest(http.MethodPost, "/users/communications/settings", strings.NewReader("voicemail_enabled=false"))
	settingsTextRequest.AddCookie(userSessionCookie)
	settingsTextRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	settingsTextResponse := httptest.NewRecorder()
	userHandler.ServeHTTP(settingsTextResponse, settingsTextRequest)
	if settingsTextResponse.Code != http.StatusOK || !strings.Contains(settingsTextResponse.Body.String(), "communications/settings") {
		t.Fatalf("unexpected text settings response %d %q", settingsTextResponse.Code, settingsTextResponse.Body.String())
	}

	failingCommService := service.NewUserCommunicationsService(handlerFailingUserStore{err: context.Canceled}, memoryStore, memoryStore, memoryStore)
	failingUserHandler := NewUserHandler("/users", authService, testDomainService(), userCookie, failingCommService)
	failingAPIUserHandler := NewAPIUserHandler("/api/v1/users", authService, testDomainService(), failingCommService)
	for _, tc := range []struct {
		path   string
		useAPI bool
	}{
		{path: "/users/presence"},
		{path: "/users/contacts/1"},
		{path: "/users/communications/settings"},
		{path: "/api/v1/users/webphone", useAPI: true},
		{path: "/api/v1/users/contacts/1", useAPI: true},
		{path: "/api/v1/users/communications/settings", useAPI: true},
	} {
		request := httptest.NewRequest(http.MethodGet, tc.path, nil)
		response := httptest.NewRecorder()
		if tc.useAPI {
			request.Header.Set("Authorization", "Bearer "+userToken.Value)
			failingAPIUserHandler.ServeHTTP(response, request)
		} else {
			request.AddCookie(userSessionCookie)
			failingUserHandler.ServeHTTP(response, request)
		}
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("%s: expected 500, got %d", tc.path, response.Code)
		}
	}
}

func seedHandlerUserCommunications(t *testing.T, memoryStore serviceAwareUserCommStore) {
	t.Helper()
	if _, err := memoryStore.SavePBXPlan(context.Background(), model.PBXPlan{
		Extensions: []model.Extension{
			{ID: 1, UserID: 1, Number: "1000", DisplayName: "Alice", Technology: "pjsip", Endpoint: "alice-web", VoicemailEnabled: true},
		},
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
			{Key: "browser_calling", Available: true},
			{Key: "tls", Available: true},
			{Key: "voicemail", Available: true},
			{Key: "presence", Available: true},
			{Key: "xmpp", Available: true},
		},
		Subsystems: []model.AsteriskManagedSubsystem{
			{Key: "messaging_backend", Provider: "xmpp", Healthy: true},
		},
	}); err != nil {
		t.Fatalf("save asterisk state: %v", err)
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
		t.Fatalf("save communication settings: %v", err)
	}
	if _, err := memoryStore.SaveUserContact(context.Background(), model.UserContact{UserID: 1, DisplayName: "Bob Example", ExtensionNumber: "1001"}); err != nil {
		t.Fatalf("save user contact: %v", err)
	}
	if _, err := memoryStore.SaveUserVoicemail(context.Background(), model.UserVoicemail{UserID: 1, From: "1002", ReceivedAt: time.Unix(50, 0)}); err != nil {
		t.Fatalf("save voicemail: %v", err)
	}
	if _, err := memoryStore.SaveUserCallRecord(context.Background(), model.UserCallRecord{UserID: 1, Counterparty: "1003", StartedAt: time.Unix(51, 0), Direction: "inbound"}); err != nil {
		t.Fatalf("save call record: %v", err)
	}
	if _, err := memoryStore.SaveUserMessage(context.Background(), model.UserMessage{UserID: 1, Counterparty: "operator", Body: "Need help", ReceivedAt: time.Unix(52, 0), Transport: "xmpp"}); err != nil {
		t.Fatalf("save message: %v", err)
	}
}

type serviceAwareUserCommStore interface {
	SavePBXPlan(context.Context, model.PBXPlan) (model.PBXPlan, error)
	SaveAsteriskState(context.Context, model.AsteriskState) (model.AsteriskState, error)
	SaveUserCommunicationSettings(context.Context, model.UserCommunicationSettings) (model.UserCommunicationSettings, error)
	SaveUserContact(context.Context, model.UserContact) (model.UserContact, error)
	SaveUserVoicemail(context.Context, model.UserVoicemail) (model.UserVoicemail, error)
	SaveUserCallRecord(context.Context, model.UserCallRecord) (model.UserCallRecord, error)
	SaveUserMessage(context.Context, model.UserMessage) (model.UserMessage, error)
}

type handlerFailingUserStore struct{ err error }

func (storeValue handlerFailingUserStore) SaveUser(context.Context, model.User) (model.User, error) {
	return model.User{}, storeValue.err
}

func (storeValue handlerFailingUserStore) FindUserByUsername(context.Context, string) (model.User, error) {
	return model.User{}, storeValue.err
}

func (storeValue handlerFailingUserStore) FindUserByEmail(context.Context, string) (model.User, error) {
	return model.User{}, storeValue.err
}

func (storeValue handlerFailingUserStore) FindUserByID(context.Context, int64) (model.User, error) {
	return model.User{}, storeValue.err
}
