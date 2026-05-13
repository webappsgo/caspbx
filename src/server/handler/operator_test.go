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
	"github.com/casapps/caspbx/src/server/store"
)

func TestOperatorAdminAndAPIHandlers(t *testing.T) {
	memoryStore, authService := newTestRuntimeStore(t)
	seedOperatorHandlerState(t, memoryStore)
	operatorService := service.NewOperatorService(memoryStore, memoryStore, memoryStore)
	adminCookie := SessionCookieConfig{Name: "admin_session", Path: "/admin", HTTPOnly: true, Secure: "auto", SameSite: http.SameSiteLaxMode}

	adminHandler := NewAdminHandler("/admin", authService, testDomainService(), testAsteriskService(), testPBXService(), adminCookie, operatorService)
	apiAdminHandler := NewAPIAdminHandler("/api/v1/admin", authService, testDomainService(), testAsteriskService(), testPBXService(), operatorService)

	loginRequest := httptest.NewRequest(http.MethodPost, "/admin", strings.NewReader("username=root-admin&password=correct+horse+battery+staple"))
	loginRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("expected admin login status 200, got %d", loginResponse.Code)
	}
	adminSessionCookie := loginResponse.Result().Cookies()[0]

	operatorRequest := httptest.NewRequest(http.MethodGet, "/admin/server/asterisk/operator", nil)
	operatorRequest.AddCookie(adminSessionCookie)
	operatorRequest.Header.Set("Accept", "application/json")
	operatorResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(operatorResponse, operatorRequest)
	if !strings.Contains(operatorResponse.Body.String(), "\"queue_count\":1") {
		t.Fatalf("unexpected operator dashboard response %q", operatorResponse.Body.String())
	}

	queueRequest := httptest.NewRequest(http.MethodGet, "/admin/server/asterisk/callcenter/queues", nil)
	queueRequest.AddCookie(adminSessionCookie)
	queueResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(queueResponse, queueRequest)
	if !strings.Contains(queueResponse.Body.String(), "Support") {
		t.Fatalf("unexpected queue wallboard response %q", queueResponse.Body.String())
	}

	conferenceRequest := httptest.NewRequest(http.MethodGet, "/admin/server/asterisk/operator/conferences", nil)
	conferenceRequest.AddCookie(adminSessionCookie)
	conferenceRequest.Header.Set("Accept", "application/json")
	conferenceResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(conferenceResponse, conferenceRequest)
	if !strings.Contains(conferenceResponse.Body.String(), `"surface":"operator/conferences"`) {
		t.Fatalf("unexpected conference response %q", conferenceResponse.Body.String())
	}

	parkedRequest := httptest.NewRequest(http.MethodGet, "/admin/server/asterisk/operator/parked-calls", nil)
	parkedRequest.AddCookie(adminSessionCookie)
	parkedRequest.Header.Set("Accept", "application/json")
	parkedResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(parkedResponse, parkedRequest)
	if !strings.Contains(parkedResponse.Body.String(), `"surface":"operator/parked-calls"`) {
		t.Fatalf("unexpected parked response %q", parkedResponse.Body.String())
	}

	agentRequest := httptest.NewRequest(http.MethodGet, "/admin/server/asterisk/callcenter/agents", nil)
	agentRequest.AddCookie(adminSessionCookie)
	agentRequest.Header.Set("Accept", "application/json")
	agentResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(agentResponse, agentRequest)
	if !strings.Contains(agentResponse.Body.String(), `"surface":"callcenter/agents"`) {
		t.Fatalf("unexpected agent response %q", agentResponse.Body.String())
	}

	previewRequest := httptest.NewRequest(http.MethodPost, "/admin/server/asterisk/callcenter/supervisor-actions/preview", strings.NewReader(`{"action":"spy","target_kind":"queue","target_ref":"Support"}`))
	previewRequest.AddCookie(adminSessionCookie)
	previewRequest.Header.Set("Content-Type", "application/json")
	previewRequest.Header.Set("Accept", "application/json")
	previewResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(previewResponse, previewRequest)
	if previewResponse.Code != http.StatusOK || !strings.Contains(previewResponse.Body.String(), `"available":true`) {
		t.Fatalf("unexpected preview response %d %q", previewResponse.Code, previewResponse.Body.String())
	}

	previewTextRequest := httptest.NewRequest(http.MethodPost, "/admin/server/asterisk/callcenter/supervisor-actions/preview", strings.NewReader(`{"action":"spy","target_kind":"queue","target_ref":"Support"}`))
	previewTextRequest.AddCookie(adminSessionCookie)
	previewTextRequest.Header.Set("Content-Type", "application/json")
	previewTextResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(previewTextResponse, previewTextRequest)
	if !strings.Contains(previewTextResponse.Body.String(), "Operator surface: callcenter/supervisor-actions") {
		t.Fatalf("unexpected preview text response %q", previewTextResponse.Body.String())
	}

	badPreviewRequest := httptest.NewRequest(http.MethodPost, "/admin/server/asterisk/callcenter/supervisor-actions/preview", strings.NewReader("{"))
	badPreviewRequest.AddCookie(adminSessionCookie)
	badPreviewRequest.Header.Set("Content-Type", "application/json")
	badPreviewResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(badPreviewResponse, badPreviewRequest)
	if badPreviewResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected bad preview status 400, got %d", badPreviewResponse.Code)
	}

	previewMethodRequest := httptest.NewRequest(http.MethodGet, "/admin/server/asterisk/callcenter/supervisor-actions/preview", nil)
	previewMethodRequest.AddCookie(adminSessionCookie)
	previewMethodResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(previewMethodResponse, previewMethodRequest)
	if previewMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected preview method status 405, got %d", previewMethodResponse.Code)
	}

	methodRequest := httptest.NewRequest(http.MethodDelete, "/admin/server/asterisk/operator", nil)
	methodRequest.AddCookie(adminSessionCookie)
	methodResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(methodResponse, methodRequest)
	if methodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected operator method status 405, got %d", methodResponse.Code)
	}

	_, adminToken, tokenError := authService.AuthenticateAdminAPI(context.Background(), "root-admin", "correct horse battery staple")
	if tokenError != nil {
		t.Fatalf("authenticate admin api: %v", tokenError)
	}

	apiOperatorRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/server/asterisk/operator/trunks", nil)
	apiOperatorRequest.Header.Set("Authorization", "Bearer "+adminToken.Value)
	apiOperatorResponse := httptest.NewRecorder()
	apiAdminHandler.ServeHTTP(apiOperatorResponse, apiOperatorRequest)
	if !strings.Contains(apiOperatorResponse.Body.String(), `"surface":"operator/trunks"`) {
		t.Fatalf("unexpected api operator response %q", apiOperatorResponse.Body.String())
	}

	apiCallcenterRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/server/asterisk/callcenter", nil)
	apiCallcenterRequest.Header.Set("Authorization", "Bearer "+adminToken.Value)
	apiCallcenterResponse := httptest.NewRecorder()
	apiAdminHandler.ServeHTTP(apiCallcenterResponse, apiCallcenterRequest)
	if !strings.Contains(apiCallcenterResponse.Body.String(), `"surface":"callcenter"`) {
		t.Fatalf("unexpected api callcenter response %q", apiCallcenterResponse.Body.String())
	}

	apiActionsRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/server/asterisk/callcenter/supervisor-actions", nil)
	apiActionsRequest.Header.Set("Authorization", "Bearer "+adminToken.Value)
	apiActionsResponse := httptest.NewRecorder()
	apiAdminHandler.ServeHTTP(apiActionsResponse, apiActionsRequest)
	if !strings.Contains(apiActionsResponse.Body.String(), `"key":"pickup"`) {
		t.Fatalf("unexpected api actions response %q", apiActionsResponse.Body.String())
	}

	apiPreviewRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/server/asterisk/callcenter/supervisor-actions/preview", strings.NewReader("action=pickup&target_kind=call&target_ref=1"))
	apiPreviewRequest.Header.Set("Authorization", "Bearer "+adminToken.Value)
	apiPreviewRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	apiPreviewResponse := httptest.NewRecorder()
	apiAdminHandler.ServeHTTP(apiPreviewResponse, apiPreviewRequest)
	if apiPreviewResponse.Code != http.StatusOK || !strings.Contains(apiPreviewResponse.Body.String(), `"available":true`) {
		t.Fatalf("unexpected api preview response %d %q", apiPreviewResponse.Code, apiPreviewResponse.Body.String())
	}

	apiPreviewMethodRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/server/asterisk/callcenter/supervisor-actions/preview", nil)
	apiPreviewMethodRequest.Header.Set("Authorization", "Bearer "+adminToken.Value)
	apiPreviewMethodResponse := httptest.NewRecorder()
	apiAdminHandler.ServeHTTP(apiPreviewMethodResponse, apiPreviewMethodRequest)
	if apiPreviewMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected api preview method status 405, got %d", apiPreviewMethodResponse.Code)
	}

	apiReadOnlyMethodRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/server/asterisk/operator/trunks", nil)
	apiReadOnlyMethodRequest.Header.Set("Authorization", "Bearer "+adminToken.Value)
	apiReadOnlyMethodResponse := httptest.NewRecorder()
	apiAdminHandler.ServeHTTP(apiReadOnlyMethodResponse, apiReadOnlyMethodRequest)
	if apiReadOnlyMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected api read-only method status 405, got %d", apiReadOnlyMethodResponse.Code)
	}
}

func TestOperatorHelpersAndErrorBranches(t *testing.T) {
	if route, ok := parseOperatorRoute("operator/parked-calls"); !ok || route.surface != "operator/parked-calls" || route.preview {
		t.Fatalf("expected parsed operator route, got %+v / %t", route, ok)
	}
	if route, ok := parseOperatorRoute("callcenter/supervisor-actions/preview"); !ok || !route.preview {
		t.Fatalf("expected parsed operator preview route, got %+v / %t", route, ok)
	}
	if _, ok := parseOperatorRoute("missing"); ok {
		t.Fatalf("expected missing operator route parse failure")
	}

	jsonRequest := httptest.NewRequest(http.MethodPost, "/preview", strings.NewReader(`{"action":"spy","target_kind":"queue","target_ref":"Support"}`))
	jsonRequest.Header.Set("Content-Type", "application/json")
	previewRequest, err := readSupervisorActionPreviewRequest(jsonRequest)
	if err != nil || previewRequest.Action != "spy" {
		t.Fatalf("expected json preview request, got %v / %+v", err, previewRequest)
	}

	formRequest := httptest.NewRequest(http.MethodPost, "/preview", strings.NewReader("action=pickup&target_kind=call&target_ref=1"))
	formRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	previewRequest, err = readSupervisorActionPreviewRequest(formRequest)
	if err != nil || previewRequest.TargetKind != "call" {
		t.Fatalf("expected form preview request, got %v / %+v", err, previewRequest)
	}

	badJSONRequest := httptest.NewRequest(http.MethodPost, "/preview", nil)
	badJSONRequest.Body = errReadCloser{}
	badJSONRequest.Header.Set("Content-Type", "application/json")
	if _, err := readSupervisorActionPreviewRequest(badJSONRequest); err == nil {
		t.Fatalf("expected json preview parse error")
	}

	badFormRequest := httptest.NewRequest(http.MethodPost, "/preview", nil)
	badFormRequest.Body = errReadCloser{}
	badFormRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if _, err := readSupervisorActionPreviewRequest(badFormRequest); err == nil {
		t.Fatalf("expected form preview parse error")
	}

	response := httptest.NewRecorder()
	writeOperatorServiceError(response, &service.OperatorError{Code: "OPERATOR_INVALID", Message: "invalid"})
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid error mapping, got %d", response.Code)
	}
	response = httptest.NewRecorder()
	writeOperatorServiceError(response, &service.OperatorError{Code: "OPERATOR_UNAVAILABLE", Message: "missing"})
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected unavailable error mapping, got %d", response.Code)
	}
	response = httptest.NewRecorder()
	writeOperatorServiceError(response, &service.OperatorError{Code: "OPERATOR_UNKNOWN", Message: "boom"})
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected unknown error mapping, got %d", response.Code)
	}
	response = httptest.NewRecorder()
	writeOperatorServiceError(response, context.Canceled)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected generic operator error mapping, got %d", response.Code)
	}

	response = httptest.NewRecorder()
	writeOperatorTextSurface(response, "root-admin", "operator", map[string]string{"status": "ok"})
	if !strings.Contains(response.Body.String(), "Operator surface: operator") {
		t.Fatalf("unexpected operator text response %q", response.Body.String())
	}

	operatorService := testOperatorService()
	if _, err := lookupOperatorSurface(context.Background(), operatorService, operatorRoute{surface: "callcenter"}); err != nil {
		t.Fatalf("expected callcenter dashboard lookup, got %v", err)
	}
	if _, err := lookupOperatorSurface(context.Background(), operatorService, operatorRoute{surface: "operator/conferences"}); err != nil {
		t.Fatalf("expected operator conferences lookup, got %v", err)
	}
	if _, err := lookupOperatorSurface(context.Background(), operatorService, operatorRoute{surface: "operator/parked-calls"}); err != nil {
		t.Fatalf("expected operator parked-calls lookup, got %v", err)
	}
	if _, err := lookupOperatorSurface(context.Background(), operatorService, operatorRoute{surface: "callcenter/queues"}); err != nil {
		t.Fatalf("expected callcenter queues lookup, got %v", err)
	}
	if _, err := lookupOperatorSurface(context.Background(), operatorService, operatorRoute{surface: "callcenter/agents"}); err != nil {
		t.Fatalf("expected callcenter agents lookup, got %v", err)
	}
	if _, err := lookupOperatorSurface(context.Background(), operatorService, operatorRoute{surface: "missing"}); err == nil {
		t.Fatalf("expected missing operator surface lookup error")
	}
}

func TestOperatorHandlerServiceErrorBranches(t *testing.T) {
	memoryStore, authService := newTestRuntimeStore(t)
	adminCookie := SessionCookieConfig{Name: "admin_session", Path: "/admin", HTTPOnly: true, Secure: "auto", SameSite: http.SameSiteLaxMode}

	loginRequest := httptest.NewRequest(http.MethodPost, "/admin", strings.NewReader("username=root-admin&password=correct+horse+battery+staple"))
	loginRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginResponse := httptest.NewRecorder()
	NewAdminHandler("/admin", authService, testDomainService(), testAsteriskService(), testPBXService(), adminCookie, service.NewOperatorService(memoryStore, memoryStore, memoryStore)).ServeHTTP(loginResponse, loginRequest)
	adminSessionCookie := loginResponse.Result().Cookies()[0]

	unavailableService := service.NewOperatorService(memoryStore, memoryStore, memoryStore)
	adminHandler := NewAdminHandler("/admin", authService, testDomainService(), testAsteriskService(), testPBXService(), adminCookie, unavailableService)

	operatorRequest := httptest.NewRequest(http.MethodGet, "/admin/server/asterisk/operator", nil)
	operatorRequest.AddCookie(adminSessionCookie)
	operatorResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(operatorResponse, operatorRequest)
	if operatorResponse.Code != http.StatusNotFound {
		t.Fatalf("expected unavailable operator status 404, got %d", operatorResponse.Code)
	}

	previewRequest := httptest.NewRequest(http.MethodPost, "/admin/server/asterisk/callcenter/supervisor-actions/preview", strings.NewReader(`{"action":"spy","target_kind":"queue","target_ref":"Support"}`))
	previewRequest.AddCookie(adminSessionCookie)
	previewRequest.Header.Set("Content-Type", "application/json")
	previewResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(previewResponse, previewRequest)
	if previewResponse.Code != http.StatusNotFound {
		t.Fatalf("expected unavailable preview status 404, got %d", previewResponse.Code)
	}

	_, adminToken, tokenError := authService.AuthenticateAdminAPI(context.Background(), "root-admin", "correct horse battery staple")
	if tokenError != nil {
		t.Fatalf("authenticate admin api: %v", tokenError)
	}
	apiAdminHandler := NewAPIAdminHandler("/api/v1/admin", authService, testDomainService(), testAsteriskService(), testPBXService(), unavailableService)

	apiOperatorRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/server/asterisk/operator/trunks", nil)
	apiOperatorRequest.Header.Set("Authorization", "Bearer "+adminToken.Value)
	apiOperatorResponse := httptest.NewRecorder()
	apiAdminHandler.ServeHTTP(apiOperatorResponse, apiOperatorRequest)
	if apiOperatorResponse.Code != http.StatusNotFound {
		t.Fatalf("expected unavailable api operator status 404, got %d", apiOperatorResponse.Code)
	}

	apiBadPreviewRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/server/asterisk/callcenter/supervisor-actions/preview", strings.NewReader("{"))
	apiBadPreviewRequest.Header.Set("Authorization", "Bearer "+adminToken.Value)
	apiBadPreviewRequest.Header.Set("Content-Type", "application/json")
	apiBadPreviewResponse := httptest.NewRecorder()
	apiAdminHandler.ServeHTTP(apiBadPreviewResponse, apiBadPreviewRequest)
	if apiBadPreviewResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected api bad preview status 400, got %d", apiBadPreviewResponse.Code)
	}

	apiPreviewRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/server/asterisk/callcenter/supervisor-actions/preview", strings.NewReader(`{"action":"spy","target_kind":"queue","target_ref":"Support"}`))
	apiPreviewRequest.Header.Set("Authorization", "Bearer "+adminToken.Value)
	apiPreviewRequest.Header.Set("Content-Type", "application/json")
	apiPreviewResponse := httptest.NewRecorder()
	apiAdminHandler.ServeHTTP(apiPreviewResponse, apiPreviewRequest)
	if apiPreviewResponse.Code != http.StatusNotFound {
		t.Fatalf("expected unavailable api preview status 404, got %d", apiPreviewResponse.Code)
	}
}

func seedOperatorHandlerState(t *testing.T, memoryStore operatorHandlerStore) {
	t.Helper()
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
		Queues:      []model.OperatorQueueState{{Name: "Support", WaitingCalls: 3, ActiveCalls: 2, AvailableAgents: 4}},
		Agents:      []model.OperatorAgentState{{ID: 1, QueueName: "Support", DisplayName: "Alice", ExtensionNumber: "1000", Status: "ready"}},
		Trunks:      []model.OperatorTrunkState{{Name: "carrier", Technology: "pjsip", Registered: true, ActiveCalls: 2, Healthy: true}},
		Conferences: []model.OperatorConferenceState{{Name: "Daily", ParticipantCount: 4, Recording: true}},
		ParkedCalls: []model.ParkedCallState{{Slot: "701", Caller: "1002", DurationSeconds: 20}},
		ActiveCalls: []model.OperatorActiveCall{{ID: 1, Direction: "inbound", Source: "1002", Destination: "1000", QueueName: "Support", AgentExtension: "1000", DurationSeconds: 40}},
		UpdatedAt:   time.Unix(100, 0),
	}); err != nil {
		t.Fatalf("save operator state: %v", err)
	}
}

func testOperatorService() service.OperatorService {
	memoryStore := store.NewMemoryStore()
	_, _ = memoryStore.SavePBXPlan(context.Background(), model.PBXPlan{
		Queues:      []model.Queue{{Name: "Support"}},
		Trunks:      []model.Trunk{{Name: "carrier", Technology: "pjsip", Endpoint: "carrier"}},
		Conferences: []model.Conference{{Name: "Daily"}},
	})
	_, _ = memoryStore.SaveAsteriskState(context.Background(), model.AsteriskState{
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
	})
	_, _ = memoryStore.SaveOperatorRuntimeState(context.Background(), model.OperatorRuntimeState{
		Queues:      []model.OperatorQueueState{{Name: "Support"}},
		Agents:      []model.OperatorAgentState{{ID: 1, QueueName: "Support", DisplayName: "Alice", ExtensionNumber: "1000"}},
		Trunks:      []model.OperatorTrunkState{{Name: "carrier", Technology: "pjsip", Registered: true, Healthy: true}},
		Conferences: []model.OperatorConferenceState{{Name: "Daily", ParticipantCount: 4}},
		ParkedCalls: []model.ParkedCallState{{Slot: "701", Caller: "1002"}},
		ActiveCalls: []model.OperatorActiveCall{{ID: 1, Source: "1002", Destination: "1000"}},
	})
	return service.NewOperatorService(memoryStore, memoryStore, memoryStore)
}

type operatorHandlerStore interface {
	SavePBXPlan(context.Context, model.PBXPlan) (model.PBXPlan, error)
	SaveAsteriskState(context.Context, model.AsteriskState) (model.AsteriskState, error)
	SaveOperatorRuntimeState(context.Context, model.OperatorRuntimeState) (model.OperatorRuntimeState, error)
	GetOperatorRuntimeState(context.Context) (model.OperatorRuntimeState, error)
	GetPBXPlan(context.Context) (model.PBXPlan, error)
	GetAsteriskState(context.Context) (model.AsteriskState, error)
}
