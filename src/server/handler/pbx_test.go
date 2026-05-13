package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/casapps/caspbx/src/server/model"
	"github.com/casapps/caspbx/src/server/service"
)

type failingPBXStateStore struct {
	err error
}

func (pbxStore failingPBXStateStore) SavePBXPlan(context.Context, model.PBXPlan) (model.PBXPlan, error) {
	return model.PBXPlan{}, pbxStore.err
}

func (pbxStore failingPBXStateStore) GetPBXPlan(context.Context) (model.PBXPlan, error) {
	return model.PBXPlan{}, pbxStore.err
}

type failingAsteriskLookupStore struct {
	err error
}

func (asteriskStore failingAsteriskLookupStore) SaveAsteriskState(context.Context, model.AsteriskState) (model.AsteriskState, error) {
	return model.AsteriskState{}, asteriskStore.err
}

func (asteriskStore failingAsteriskLookupStore) GetAsteriskState(context.Context) (model.AsteriskState, error) {
	return model.AsteriskState{}, asteriskStore.err
}

func TestPBXHelpers(t *testing.T) {
	if route, ok := parsePBXRoute("extensions/12"); !ok || route.resource != "extensions" || !route.hasID || route.id != 12 {
		t.Fatalf("unexpected parsed pbx route %+v / %t", route, ok)
	}
	if route, ok := parsePBXRoute("apply-preview"); !ok || route.resource != "apply-preview" {
		t.Fatalf("unexpected apply-preview route %+v / %t", route, ok)
	}
	if _, ok := parsePBXRoute("extensions/bad"); ok {
		t.Fatalf("expected invalid pbx route id to fail")
	}
	if _, ok := parsePBXRoute("fax"); ok {
		t.Fatalf("expected non-pbx asterisk surface not to parse as pbx route")
	}

	textResponse := httptest.NewRecorder()
	writePBXEntityText(textResponse, "root-admin", "extensions", []model.Extension{{ID: 1, Number: "1000", DisplayName: "Alice", Technology: "pjsip"}})
	if !strings.Contains(textResponse.Body.String(), "PBX resource: extensions") {
		t.Fatalf("unexpected pbx text body %q", textResponse.Body.String())
	}
	previewText := httptest.NewRecorder()
	writePBXPreviewText(previewText, "root-admin", service.PBXApplyPreview{
		UpdatedAt:   "2024-01-01T00:00:00Z",
		Summaries:   []service.PBXResourceSummary{{Resource: "extensions", Count: 1}},
		Artifacts:   []service.PBXApplyArtifact{{Label: "Dialplan", Status: "ready", Summary: "render"}},
		Validations: []string{"warning"},
		Actions:     []string{"validate"},
	})
	if !strings.Contains(previewText.Body.String(), "apply-preview") {
		t.Fatalf("unexpected pbx preview text %q", previewText.Body.String())
	}

	internalError := httptest.NewRecorder()
	writePBXServiceError(internalError, errors.New("boom"))
	if internalError.Code != http.StatusInternalServerError {
		t.Fatalf("expected pbx internal error response, got %d", internalError.Code)
	}
	invalidError := httptest.NewRecorder()
	writePBXServiceError(invalidError, &service.PBXError{Code: "PBX_INVALID", Message: "bad"})
	if invalidError.Code != http.StatusBadRequest {
		t.Fatalf("expected pbx invalid response, got %d", invalidError.Code)
	}
	notFoundError := httptest.NewRecorder()
	writePBXServiceError(notFoundError, &service.PBXError{Code: "PBX_NOT_FOUND", Message: "missing"})
	if notFoundError.Code != http.StatusNotFound {
		t.Fatalf("expected pbx not-found response, got %d", notFoundError.Code)
	}

	if values := splitCSV("1000, 1001 ,"); len(values) != 2 || values[1] != "1001" {
		t.Fatalf("unexpected split csv %+v", values)
	}
	jsonRequest := httptest.NewRequest(http.MethodPost, "/admin/server/asterisk/extensions", strings.NewReader(`{"number":"1000","display_name":"Alice","technology":"pjsip"}`))
	jsonRequest.Header.Set("Content-Type", "application/json")
	if request, err := readExtensionRequest(jsonRequest); err != nil || request.DisplayName != "Alice" {
		t.Fatalf("unexpected extension request parse %v / %+v", err, request)
	}
	badJSONRequest := httptest.NewRequest(http.MethodPost, "/admin/server/asterisk/routes", strings.NewReader("{"))
	badJSONRequest.Header.Set("Content-Type", "application/json")
	if _, err := readRouteRequest(badJSONRequest); err == nil {
		t.Fatalf("expected bad route request parse error")
	}
	formRequest := httptest.NewRequest(http.MethodPost, "/admin/server/asterisk/queues", strings.NewReader("name=Support&strategy=ringall&member_extension_numbers=1000,1001"))
	formRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if request, err := readQueueRequest(formRequest); err != nil || len(request.MemberExtensionNumbers) != 2 {
		t.Fatalf("unexpected queue request parse %v / %+v", err, request)
	}
	formExtension := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("number=1000&display_name=Alice&technology=pjsip&endpoint=alice&voicemail_enabled=true"))
	formExtension.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if request, err := readExtensionRequest(formExtension); err != nil || !request.VoicemailEnabled {
		t.Fatalf("unexpected extension form parse %v / %+v", err, request)
	}
	formTrunk := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("name=Primary&technology=pjsip&endpoint=sip.example&active=true"))
	formTrunk.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if request, err := readTrunkRequest(formTrunk); err != nil || request.Endpoint != "sip.example" {
		t.Fatalf("unexpected trunk form parse %v / %+v", err, request)
	}
	formRoute := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("name=Main&direction=outbound&match=_NXXNXXXXXX&destination=trunk:1"))
	formRoute.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if request, err := readRouteRequest(formRoute); err != nil || request.Match != "_NXXNXXXXXX" {
		t.Fatalf("unexpected route form parse %v / %+v", err, request)
	}
	jsonQueue := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Support","strategy":"ringall","member_extension_numbers":["1000","1001"]}`))
	jsonQueue.Header.Set("Content-Type", "application/json")
	if request, err := readQueueRequest(jsonQueue); err != nil || len(request.MemberExtensionNumbers) != 2 {
		t.Fatalf("unexpected queue json parse %v / %+v", err, request)
	}
	for _, requestFactory := range []func() (*http.Request, func(*http.Request) error){
		func() (*http.Request, func(*http.Request) error) {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{"))
			request.Header.Set("Content-Type", "application/json")
			return request, func(r *http.Request) error { _, err := readExtensionRequest(r); return err }
		},
		func() (*http.Request, func(*http.Request) error) {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{"))
			request.Header.Set("Content-Type", "application/json")
			return request, func(r *http.Request) error { _, err := readTrunkRequest(r); return err }
		},
		func() (*http.Request, func(*http.Request) error) {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{"))
			request.Header.Set("Content-Type", "application/json")
			return request, func(r *http.Request) error { _, err := readConferenceRequest(r); return err }
		},
		func() (*http.Request, func(*http.Request) error) {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{"))
			request.Header.Set("Content-Type", "application/json")
			return request, func(r *http.Request) error { _, err := readIVRRequest(r); return err }
		},
		func() (*http.Request, func(*http.Request) error) {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{"))
			request.Header.Set("Content-Type", "application/json")
			return request, func(r *http.Request) error { _, err := readPromptAssignmentRequest(r); return err }
		},
		func() (*http.Request, func(*http.Request) error) {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{"))
			request.Header.Set("Content-Type", "application/json")
			return request, func(r *http.Request) error { _, err := readProvisioningProfileRequest(r); return err }
		},
		func() (*http.Request, func(*http.Request) error) {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{"))
			request.Header.Set("Content-Type", "application/json")
			return request, func(r *http.Request) error { _, err := readQueueRequest(r); return err }
		},
	} {
		request, read := requestFactory()
		if err := read(request); err == nil {
			t.Fatalf("expected bad json parse error")
		}
	}
	for _, read := range []func(*http.Request) error{
		func(r *http.Request) error { _, err := readExtensionRequest(r); return err },
		func(r *http.Request) error { _, err := readTrunkRequest(r); return err },
		func(r *http.Request) error { _, err := readRouteRequest(r); return err },
		func(r *http.Request) error { _, err := readQueueRequest(r); return err },
		func(r *http.Request) error { _, err := readConferenceRequest(r); return err },
		func(r *http.Request) error { _, err := readIVRRequest(r); return err },
		func(r *http.Request) error { _, err := readPromptAssignmentRequest(r); return err },
		func(r *http.Request) error { _, err := readProvisioningProfileRequest(r); return err },
	} {
		request := httptest.NewRequest(http.MethodPost, "/", nil)
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.Body = errReadCloser{}
		if err := read(request); err == nil {
			t.Fatalf("expected form parse error")
		}
	}
}

func TestPBXAdminAndAPIHandlers(t *testing.T) {
	authService := newTestAuthService(t)
	adminCookie := SessionCookieConfig{Name: "admin_session", Path: "/admin", HTTPOnly: true, Secure: "auto", SameSite: http.SameSiteLaxMode}

	loginForm := httptest.NewRequest(http.MethodPost, "/admin", strings.NewReader("username=root-admin&password=correct+horse+battery+staple"))
	loginForm.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginResponse := httptest.NewRecorder()
	NewAdminHandler("/admin", authService, testDomainService(), testAsteriskService(), testPBXService(), adminCookie).ServeHTTP(loginResponse, loginForm)
	adminSessionCookie := loginResponse.Result().Cookies()[0]

	adminHandlerValue := NewAdminHandler("/admin", authService, testDomainService(), testAsteriskService(), testPBXService(), adminCookie).(AdminHandler)
	createExtensionRequest := httptest.NewRequest(http.MethodPost, "/admin/server/asterisk/extensions", strings.NewReader(`{"number":"2000","display_name":"Bob","technology":"pjsip","endpoint":"bob"}`))
	createExtensionRequest.Header.Set("Content-Type", "application/json")
	createExtensionRequest.Header.Set("Accept", "application/json")
	createExtensionRequest.AddCookie(adminSessionCookie)
	createExtensionResponse := httptest.NewRecorder()
	adminHandlerValue.ServeHTTP(createExtensionResponse, createExtensionRequest)
	if createExtensionResponse.Code != http.StatusCreated || !strings.Contains(createExtensionResponse.Body.String(), "\"resource\":\"extensions\"") {
		t.Fatalf("unexpected pbx create extension response %d %q", createExtensionResponse.Code, createExtensionResponse.Body.String())
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/admin/server/asterisk/extensions", nil)
	listRequest.AddCookie(adminSessionCookie)
	listResponse := httptest.NewRecorder()
	adminHandlerValue.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK || !strings.Contains(listResponse.Body.String(), "PBX resource: extensions") {
		t.Fatalf("unexpected pbx extension list response %d %q", listResponse.Code, listResponse.Body.String())
	}

	detailRequest := httptest.NewRequest(http.MethodGet, "/admin/server/asterisk/extensions/1", nil)
	detailRequest.Header.Set("Accept", "application/json")
	detailRequest.AddCookie(adminSessionCookie)
	detailResponse := httptest.NewRecorder()
	adminHandlerValue.ServeHTTP(detailResponse, detailRequest)
	if detailResponse.Code != http.StatusOK || !strings.Contains(detailResponse.Body.String(), "\"display_name\":\"Alice\"") {
		t.Fatalf("unexpected pbx extension detail response %d %q", detailResponse.Code, detailResponse.Body.String())
	}

	previewRequest := httptest.NewRequest(http.MethodGet, "/admin/server/asterisk/apply-preview", nil)
	previewRequest.AddCookie(adminSessionCookie)
	previewResponse := httptest.NewRecorder()
	adminHandlerValue.ServeHTTP(previewResponse, previewRequest)
	if previewResponse.Code != http.StatusOK || !strings.Contains(previewResponse.Body.String(), "apply-preview") {
		t.Fatalf("unexpected pbx preview response %d %q", previewResponse.Code, previewResponse.Body.String())
	}

	badCreateRequest := httptest.NewRequest(http.MethodPost, "/admin/server/asterisk/extensions", strings.NewReader("{"))
	badCreateRequest.Header.Set("Content-Type", "application/json")
	badCreateRequest.AddCookie(adminSessionCookie)
	badCreateResponse := httptest.NewRecorder()
	adminHandlerValue.ServeHTTP(badCreateResponse, badCreateRequest)
	if badCreateResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected bad pbx create response, got %d", badCreateResponse.Code)
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/admin/server/asterisk/extensions/1", nil)
	deleteRequest.AddCookie(adminSessionCookie)
	deleteResponse := httptest.NewRecorder()
	adminHandlerValue.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusOK || !strings.Contains(deleteResponse.Body.String(), "\"status\":\"deleted\"") {
		t.Fatalf("unexpected pbx delete response %d %q", deleteResponse.Code, deleteResponse.Body.String())
	}

	unauthorizedResponse := httptest.NewRecorder()
	adminHandlerValue.handleAsteriskSurface(unauthorizedResponse, httptest.NewRequest(http.MethodGet, "/admin/server/asterisk/extensions", nil), "server/asterisk/extensions")
	if unauthorizedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected pbx admin unauthorized response, got %d", unauthorizedResponse.Code)
	}

	apiTokenOwner, adminToken, tokenError := authService.AuthenticateAdminAPI(context.Background(), "root-admin", "correct horse battery staple")
	if tokenError != nil || apiTokenOwner.Username != "root-admin" {
		t.Fatalf("expected admin api token, got %v / %+v", tokenError, apiTokenOwner)
	}
	apiHandlerValue := NewAPIAdminHandler("/api/v1/admin/server/asterisk", authService, testDomainService(), testAsteriskService(), testPBXService()).(APIAdminHandler)
	createTrunkRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/server/asterisk/trunks", strings.NewReader(`{"name":"Primary Carrier","technology":"pjsip","endpoint":"sip.provider.example"}`))
	createTrunkRequest.Header.Set("Content-Type", "application/json")
	createTrunkRequest.Header.Set("Authorization", "Bearer "+adminToken.Value)
	createTrunkResponse := httptest.NewRecorder()
	apiHandlerValue.ServeHTTP(createTrunkResponse, createTrunkRequest)
	if createTrunkResponse.Code != http.StatusCreated || !strings.Contains(createTrunkResponse.Body.String(), "\"resource\":\"trunks\"") {
		t.Fatalf("unexpected api pbx create trunk response %d %q", createTrunkResponse.Code, createTrunkResponse.Body.String())
	}

	apiPreviewRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/server/asterisk/apply-preview", nil)
	apiPreviewRequest.Header.Set("Authorization", "Bearer "+adminToken.Value)
	apiPreviewResponse := httptest.NewRecorder()
	apiHandlerValue.ServeHTTP(apiPreviewResponse, apiPreviewRequest)
	if apiPreviewResponse.Code != http.StatusOK || !strings.Contains(apiPreviewResponse.Body.String(), "\"summaries\"") {
		t.Fatalf("unexpected api pbx preview response %d %q", apiPreviewResponse.Code, apiPreviewResponse.Body.String())
	}

	apiDeleteRequest := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/server/asterisk/trunks/1", nil)
	apiDeleteRequest.Header.Set("Authorization", "Bearer "+adminToken.Value)
	apiDeleteResponse := httptest.NewRecorder()
	apiHandlerValue.ServeHTTP(apiDeleteResponse, apiDeleteRequest)
	if apiDeleteResponse.Code != http.StatusOK {
		t.Fatalf("unexpected api pbx delete response %d %q", apiDeleteResponse.Code, apiDeleteResponse.Body.String())
	}

	apiBadCreateRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/server/asterisk/trunks", strings.NewReader("{"))
	apiBadCreateRequest.Header.Set("Content-Type", "application/json")
	apiBadCreateRequest.Header.Set("Authorization", "Bearer "+adminToken.Value)
	apiBadCreateResponse := httptest.NewRecorder()
	apiHandlerValue.ServeHTTP(apiBadCreateResponse, apiBadCreateRequest)
	if apiBadCreateResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected api bad create response, got %d", apiBadCreateResponse.Code)
	}
}

func TestPBXDirectHelpersAndBranches(t *testing.T) {
	adminHandler := AdminHandler{pbxService: testPBXService()}
	apiHandler := APIAdminHandler{pbxService: testPBXService()}
	failingPBXService := service.NewPBXService(failingPBXStateStore{err: errors.New("lookup failed")}, failingAsteriskLookupStore{err: errors.New("lookup failed")})
	failingHandler := AdminHandler{pbxService: failingPBXService}
	failingAPIHandler := APIAdminHandler{pbxService: failingPBXService}

	for _, resource := range []struct {
		name string
		body string
		path string
	}{
		{"extensions", `{"number":"3000","display_name":"Carol","technology":"pjsip","endpoint":"carol"}`, "/admin/server/asterisk/extensions"},
		{"trunks", `{"name":"Primary Carrier","technology":"pjsip","endpoint":"sip.provider.example"}`, "/admin/server/asterisk/trunks"},
		{"routes", `{"name":"Main","direction":"outbound","destination":"trunk:1"}`, "/admin/server/asterisk/routes"},
		{"queues", `{"name":"Support","strategy":"ringall","member_extension_numbers":["1000"]}`, "/admin/server/asterisk/queues"},
		{"conferences", `{"name":"Daily","access_code":"7000","recording_enabled":true}`, "/admin/server/asterisk/conferences"},
		{"ivrs", `{"name":"Main","root_prompt":"welcome","default_destination":"queue:1","timeout_seconds":5}`, "/admin/server/asterisk/ivrs"},
		{"prompt-assignments", `{"name":"Greeting","prompt_name":"welcome","target_kind":"ivr","target_ref":"1"}`, "/admin/server/asterisk/prompt-assignments"},
		{"provisioning-profiles", `{"name":"Desk","technology":"pjsip","template":"yealink"}`, "/admin/server/asterisk/provisioning-profiles"},
	} {
		request := httptest.NewRequest(http.MethodPost, resource.path, strings.NewReader(resource.body))
		request.Header.Set("Content-Type", "application/json")
		if value, err := adminHandler.createPBXResource(request, resource.name); err != nil || value == nil {
			t.Fatalf("expected created %s, got %v / %+v", resource.name, err, value)
		}
	}
	if value, err := adminHandler.createPBXResource(httptest.NewRequest(http.MethodPost, "/admin/server/asterisk/unknown", nil), "unknown"); err == nil || value != nil {
		t.Fatalf("expected unknown create resource error, got %v / %+v", err, value)
	}
	for _, resource := range []string{"routes", "queues", "conferences", "ivrs", "prompt-assignments", "provisioning-profiles"} {
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{"))
		request.Header.Set("Content-Type", "application/json")
		if value, err := createPBXResource(request, adminHandler.pbxService, resource); err == nil || value != nil {
			t.Fatalf("expected invalid create for %s, got %v / %+v", resource, err, value)
		}
	}

	for _, resource := range []struct {
		name  string
		route pbxRoute
	}{
		{"extension", pbxRoute{resource: "extensions", id: 1, hasID: true}},
		{"trunk", pbxRoute{resource: "trunks", id: 1, hasID: true}},
		{"route", pbxRoute{resource: "routes", id: 1, hasID: true}},
		{"queue", pbxRoute{resource: "queues", id: 1, hasID: true}},
		{"conference", pbxRoute{resource: "conferences", id: 1, hasID: true}},
		{"ivr", pbxRoute{resource: "ivrs", id: 1, hasID: true}},
		{"prompt assignment", pbxRoute{resource: "prompt-assignments", id: 1, hasID: true}},
		{"provisioning profile", pbxRoute{resource: "provisioning-profiles", id: 1, hasID: true}},
	} {
		if _, err := adminHandler.lookupPBXResource(httptest.NewRequest(http.MethodGet, "/", nil), resource.route); err != nil {
			t.Fatalf("expected lookup %s success, got %v", resource.name, err)
		}
	}
	for _, route := range []pbxRoute{
		{resource: "extensions"},
		{resource: "trunks"},
		{resource: "routes"},
		{resource: "queues"},
		{resource: "conferences"},
		{resource: "ivrs"},
		{resource: "prompt-assignments"},
		{resource: "provisioning-profiles"},
	} {
		if _, err := lookupPBXResource(context.Background(), adminHandler.pbxService, route); err != nil {
			t.Fatalf("expected global lookup success for %s, got %v", route.resource, err)
		}
	}
	if _, err := apiHandler.lookupPBXResource(httptest.NewRequest(http.MethodGet, "/", nil), pbxRoute{resource: "unknown"}); err == nil {
		t.Fatalf("expected api lookup unknown resource failure")
	}

	for _, resource := range []struct {
		route pbxRoute
	}{
		{pbxRoute{resource: "extensions", id: 1, hasID: true}},
		{pbxRoute{resource: "trunks", id: 1, hasID: true}},
		{pbxRoute{resource: "routes", id: 1, hasID: true}},
		{pbxRoute{resource: "queues", id: 1, hasID: true}},
		{pbxRoute{resource: "conferences", id: 1, hasID: true}},
		{pbxRoute{resource: "ivrs", id: 1, hasID: true}},
		{pbxRoute{resource: "prompt-assignments", id: 1, hasID: true}},
		{pbxRoute{resource: "provisioning-profiles", id: 1, hasID: true}},
	} {
		if err := adminHandler.deletePBXResource(httptest.NewRequest(http.MethodDelete, "/", nil), resource.route); err != nil {
			t.Fatalf("expected pbx delete success, got %v", err)
		}
	}
	if err := apiHandler.deletePBXResource(httptest.NewRequest(http.MethodDelete, "/", nil), pbxRoute{resource: "unknown", id: 1, hasID: true}); err == nil {
		t.Fatalf("expected api delete unknown resource failure")
	}
	if err := deletePBXResource(httptest.NewRequest(http.MethodDelete, "/", nil), adminHandler.pbxService, pbxRoute{resource: "unknown", id: 1, hasID: true}); err == nil {
		t.Fatalf("expected global delete unknown resource failure")
	}

	methodNotAllowedResponse := httptest.NewRecorder()
	adminHandler.handlePBXEntitySurface(methodNotAllowedResponse, httptest.NewRequest(http.MethodPost, "/", nil), pbxRoute{resource: "extensions", id: 1, hasID: true}, "root-admin", "session")
	if methodNotAllowedResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected pbx entity post-with-id rejection, got %d", methodNotAllowedResponse.Code)
	}
	deleteCollectionResponse := httptest.NewRecorder()
	adminHandler.handlePBXEntitySurface(deleteCollectionResponse, httptest.NewRequest(http.MethodDelete, "/", nil), pbxRoute{resource: "extensions"}, "root-admin", "session")
	if deleteCollectionResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected pbx entity delete-collection rejection, got %d", deleteCollectionResponse.Code)
	}
	apiMethodNotAllowedResponse := httptest.NewRecorder()
	apiHandler.handlePBXEntitySurface(apiMethodNotAllowedResponse, httptest.NewRequest(http.MethodPatch, "/", nil), pbxRoute{resource: "extensions"}, "root-admin", "adm_aaaa")
	if apiMethodNotAllowedResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected api pbx entity patch rejection, got %d", apiMethodNotAllowedResponse.Code)
	}
	adminGetEntityResponse := httptest.NewRecorder()
	adminHandler.handlePBXEntitySurface(adminGetEntityResponse, httptest.NewRequest(http.MethodGet, "/", nil), pbxRoute{resource: "extensions"}, "root-admin", "session")
	if adminGetEntityResponse.Code != http.StatusOK {
		t.Fatalf("expected admin pbx entity get success, got %d", adminGetEntityResponse.Code)
	}
	adminGetEntityErrorResponse := httptest.NewRecorder()
	failingHandler.handlePBXEntitySurface(adminGetEntityErrorResponse, httptest.NewRequest(http.MethodGet, "/", nil), pbxRoute{resource: "extensions"}, "root-admin", "session")
	if adminGetEntityErrorResponse.Code != http.StatusInternalServerError {
		t.Fatalf("expected admin pbx entity get error, got %d", adminGetEntityErrorResponse.Code)
	}
	apiGetEntityResponse := httptest.NewRecorder()
	apiHandler.handlePBXEntitySurface(apiGetEntityResponse, httptest.NewRequest(http.MethodGet, "/", nil), pbxRoute{resource: "extensions"}, "root-admin", "adm_aaaa")
	if apiGetEntityResponse.Code != http.StatusOK {
		t.Fatalf("expected api pbx entity get success, got %d", apiGetEntityResponse.Code)
	}
	apiGetEntityErrorResponse := httptest.NewRecorder()
	failingAPIHandler.handlePBXEntitySurface(apiGetEntityErrorResponse, httptest.NewRequest(http.MethodGet, "/", nil), pbxRoute{resource: "extensions"}, "root-admin", "adm_aaaa")
	if apiGetEntityErrorResponse.Code != http.StatusInternalServerError {
		t.Fatalf("expected api pbx entity get error, got %d", apiGetEntityErrorResponse.Code)
	}
	adminPostTextResponse := httptest.NewRecorder()
	adminPostTextRequest := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"number":"4000","display_name":"Dave","technology":"pjsip"}`))
	adminPostTextRequest.Header.Set("Content-Type", "application/json")
	adminHandler.handlePBXEntitySurface(adminPostTextResponse, adminPostTextRequest, pbxRoute{resource: "extensions"}, "root-admin", "session")
	if adminPostTextResponse.Code != http.StatusOK {
		t.Fatalf("expected admin pbx text create success, got %d", adminPostTextResponse.Code)
	}
	adminDeleteErrorResponse := httptest.NewRecorder()
	failingHandler.handlePBXEntitySurface(adminDeleteErrorResponse, httptest.NewRequest(http.MethodDelete, "/", nil), pbxRoute{resource: "extensions", id: 1, hasID: true}, "root-admin", "session")
	if adminDeleteErrorResponse.Code != http.StatusInternalServerError {
		t.Fatalf("expected admin pbx delete error, got %d", adminDeleteErrorResponse.Code)
	}
	adminPatchResponse := httptest.NewRecorder()
	adminHandler.handlePBXEntitySurface(adminPatchResponse, httptest.NewRequest(http.MethodPatch, "/", nil), pbxRoute{resource: "extensions"}, "root-admin", "session")
	if adminPatchResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected admin pbx patch rejection, got %d", adminPatchResponse.Code)
	}
	apiPostWithIDResponse := httptest.NewRecorder()
	apiHandler.handlePBXEntitySurface(apiPostWithIDResponse, httptest.NewRequest(http.MethodPost, "/", nil), pbxRoute{resource: "extensions", id: 1, hasID: true}, "root-admin", "adm_aaaa")
	if apiPostWithIDResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected api pbx post-with-id rejection, got %d", apiPostWithIDResponse.Code)
	}
	apiDeleteCollectionResponse := httptest.NewRecorder()
	apiHandler.handlePBXEntitySurface(apiDeleteCollectionResponse, httptest.NewRequest(http.MethodDelete, "/", nil), pbxRoute{resource: "extensions"}, "root-admin", "adm_aaaa")
	if apiDeleteCollectionResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected api pbx delete collection rejection, got %d", apiDeleteCollectionResponse.Code)
	}
	apiDeleteErrorResponse := httptest.NewRecorder()
	failingAPIHandler.handlePBXEntitySurface(apiDeleteErrorResponse, httptest.NewRequest(http.MethodDelete, "/", nil), pbxRoute{resource: "extensions", id: 1, hasID: true}, "root-admin", "adm_aaaa")
	if apiDeleteErrorResponse.Code != http.StatusInternalServerError {
		t.Fatalf("expected api pbx delete error, got %d", apiDeleteErrorResponse.Code)
	}
	adminPreviewMethodResponse := httptest.NewRecorder()
	adminHandler.handlePBXSurface(adminPreviewMethodResponse, httptest.NewRequest(http.MethodDelete, "/", nil), pbxRoute{resource: "apply-preview"}, "root-admin", "session")
	if adminPreviewMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected admin apply-preview delete rejection, got %d", adminPreviewMethodResponse.Code)
	}
	apiPreviewMethodResponse := httptest.NewRecorder()
	apiHandler.handlePBXSurface(apiPreviewMethodResponse, httptest.NewRequest(http.MethodDelete, "/", nil), pbxRoute{resource: "apply-preview"}, "root-admin", "adm_aaaa", model.TokenScopeGlobal)
	if apiPreviewMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected api apply-preview delete rejection, got %d", apiPreviewMethodResponse.Code)
	}
	adminPreviewJSONResponse := httptest.NewRecorder()
	adminPreviewJSONRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	adminPreviewJSONRequest.Header.Set("Accept", "application/json")
	adminHandler.handlePBXSurface(adminPreviewJSONResponse, adminPreviewJSONRequest, pbxRoute{resource: "apply-preview"}, "root-admin", "session")
	if adminPreviewJSONResponse.Code != http.StatusOK {
		t.Fatalf("expected admin apply-preview json success, got %d", adminPreviewJSONResponse.Code)
	}
	adminPreviewErrorResponse := httptest.NewRecorder()
	failingHandler.handlePBXSurface(adminPreviewErrorResponse, httptest.NewRequest(http.MethodGet, "/", nil), pbxRoute{resource: "apply-preview"}, "root-admin", "session")
	if adminPreviewErrorResponse.Code != http.StatusInternalServerError {
		t.Fatalf("expected admin apply-preview error, got %d", adminPreviewErrorResponse.Code)
	}
	apiPreviewJSONResponse := httptest.NewRecorder()
	apiHandler.handlePBXSurface(apiPreviewJSONResponse, httptest.NewRequest(http.MethodGet, "/", nil), pbxRoute{resource: "apply-preview"}, "root-admin", "adm_aaaa", model.TokenScopeGlobal)
	if apiPreviewJSONResponse.Code != http.StatusOK {
		t.Fatalf("expected api apply-preview get success, got %d", apiPreviewJSONResponse.Code)
	}
	apiPreviewErrorResponse := httptest.NewRecorder()
	failingAPIHandler.handlePBXSurface(apiPreviewErrorResponse, httptest.NewRequest(http.MethodGet, "/", nil), pbxRoute{resource: "apply-preview"}, "root-admin", "adm_aaaa", model.TokenScopeGlobal)
	if apiPreviewErrorResponse.Code != http.StatusInternalServerError {
		t.Fatalf("expected api apply-preview error, got %d", apiPreviewErrorResponse.Code)
	}

	for _, data := range []any{
		[]model.Extension{{ID: 1, Number: "1000", DisplayName: "Alice", Technology: "pjsip"}},
		model.Extension{ID: 1, Number: "1000", DisplayName: "Alice", Technology: "pjsip"},
		[]model.Trunk{{ID: 1, Name: "Primary", Technology: "pjsip"}},
		model.Trunk{ID: 1, Name: "Primary", Technology: "pjsip", Endpoint: "sip.example"},
		[]model.CallRoute{{ID: 1, Name: "Main", Direction: "outbound"}},
		model.CallRoute{ID: 1, Name: "Main", Direction: "outbound", Destination: "trunk:1"},
		[]model.Queue{{ID: 1, Name: "Support"}},
		model.Queue{ID: 1, Name: "Support", Strategy: "ringall"},
		[]model.Conference{{ID: 1, Name: "Daily"}},
		model.Conference{ID: 1, Name: "Daily", AccessCode: "7000"},
		[]model.IVR{{ID: 1, Name: "Main"}},
		model.IVR{ID: 1, Name: "Main", DefaultDestination: "queue:1"},
		[]model.PromptAssignment{{ID: 1, Name: "Greeting", TargetKind: "ivr"}},
		model.PromptAssignment{ID: 1, Name: "Greeting", PromptName: "welcome", TargetKind: "ivr", TargetRef: "1"},
		[]model.ProvisioningProfile{{ID: 1, Name: "Desk", Technology: "pjsip"}},
		model.ProvisioningProfile{ID: 1, Name: "Desk", Technology: "pjsip", Template: "yealink"},
	} {
		response := httptest.NewRecorder()
		writePBXEntityText(response, "root-admin", "detail", data)
		if response.Code != http.StatusOK {
			t.Fatalf("expected pbx text detail write success")
		}
	}

	formConference := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("name=Daily&access_code=7000&recording_enabled=true"))
	formConference.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if request, err := readConferenceRequest(formConference); err != nil || request.AccessCode != "7000" {
		t.Fatalf("unexpected conference form parse %v / %+v", err, request)
	}
	formIVR := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("name=Main&root_prompt=welcome&default_destination=queue:1&timeout_seconds=5"))
	formIVR.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if request, err := readIVRRequest(formIVR); err != nil || request.TimeoutSeconds != 5 {
		t.Fatalf("unexpected ivr form parse %v / %+v", err, request)
	}
	formPrompt := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("name=Greeting&prompt_name=welcome&target_kind=ivr&target_ref=1"))
	formPrompt.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if request, err := readPromptAssignmentRequest(formPrompt); err != nil || request.TargetKind != "ivr" {
		t.Fatalf("unexpected prompt assignment form parse %v / %+v", err, request)
	}
	formProfile := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("name=Desk&technology=pjsip&template=yealink&assigned_extensions=1000,1001"))
	formProfile.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if request, err := readProvisioningProfileRequest(formProfile); err != nil || len(request.AssignedExtensions) != 2 {
		t.Fatalf("unexpected provisioning form parse %v / %+v", err, request)
	}
	jsonConference := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Daily","access_code":"7000","recording_enabled":true}`))
	jsonConference.Header.Set("Content-Type", "application/json")
	if request, err := readConferenceRequest(jsonConference); err != nil || !request.RecordingEnabled {
		t.Fatalf("unexpected conference json parse %v / %+v", err, request)
	}
	jsonIVR := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Main","root_prompt":"welcome","default_destination":"queue:1","timeout_seconds":5}`))
	jsonIVR.Header.Set("Content-Type", "application/json")
	if request, err := readIVRRequest(jsonIVR); err != nil || request.RootPrompt != "welcome" {
		t.Fatalf("unexpected ivr json parse %v / %+v", err, request)
	}
	jsonPrompt := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Greeting","prompt_name":"welcome","target_kind":"ivr","target_ref":"1"}`))
	jsonPrompt.Header.Set("Content-Type", "application/json")
	if request, err := readPromptAssignmentRequest(jsonPrompt); err != nil || request.PromptName != "welcome" {
		t.Fatalf("unexpected prompt assignment json parse %v / %+v", err, request)
	}
	jsonProfile := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Desk","technology":"pjsip","template":"yealink","assigned_extensions":["1000"]}`))
	jsonProfile.Header.Set("Content-Type", "application/json")
	if request, err := readProvisioningProfileRequest(jsonProfile); err != nil || request.Template != "yealink" {
		t.Fatalf("unexpected provisioning json parse %v / %+v", err, request)
	}
	typedInternalError := httptest.NewRecorder()
	writePBXServiceError(typedInternalError, &service.PBXError{Code: "PBX_INTERNAL", Message: "boom"})
	if typedInternalError.Code != http.StatusInternalServerError {
		t.Fatalf("expected typed pbx internal error response, got %d", typedInternalError.Code)
	}
	if values := splitCSV(""); values != nil {
		t.Fatalf("expected empty split csv to return nil, got %+v", values)
	}
}
