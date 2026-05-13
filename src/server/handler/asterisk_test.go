package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/casapps/caspbx/src/server/model"
	"github.com/casapps/caspbx/src/server/service"
)

type failingAsteriskStateStore struct {
	err error
}

func (asteriskStore failingAsteriskStateStore) SaveAsteriskState(context.Context, model.AsteriskState) (model.AsteriskState, error) {
	return model.AsteriskState{}, asteriskStore.err
}

func (asteriskStore failingAsteriskStateStore) GetAsteriskState(context.Context) (model.AsteriskState, error) {
	return model.AsteriskState{}, asteriskStore.err
}

func TestAsteriskHelpers(t *testing.T) {
	if key, found := normalizeAsteriskRoute("/admin", "server/asterisk/fax"); !found || key != "fax" {
		t.Fatalf("unexpected normalized admin asterisk route %t / %q", found, key)
	}
	if key, found := normalizeAsteriskRoute("/admin/server/asterisk", "fax"); !found || key != "fax" {
		t.Fatalf("unexpected normalized exact asterisk route %t / %q", found, key)
	}
	if _, found := normalizeAsteriskRoute("/admin", "server/settings"); found {
		t.Fatalf("expected non-asterisk route normalization to fail")
	}

	textResponse := httptest.NewRecorder()
	writeAsteriskTextSurface(textResponse, service.AsteriskSurfaceView{
		Surface:         model.AsteriskSurface{Key: "fax", Label: "Fax"},
		Summary:         "Fax summary",
		DetectionStatus: "detected",
		HealthStatus:    model.AsteriskHealthReady,
		Items:           []service.AsteriskSurfaceItem{{Label: "Fax backend", Status: "ready", Value: "hylafax+", Detail: "healthy"}},
		AvailableSurfaces: []model.AsteriskSurface{
			{Key: "overview", Label: "Overview"},
			{Key: "fax", Label: "Fax"},
		},
	}, "root-admin")
	if !strings.Contains(textResponse.Body.String(), "Asterisk surface: Fax") {
		t.Fatalf("unexpected asterisk text output %q", textResponse.Body.String())
	}

	internalErrorResponse := httptest.NewRecorder()
	writeAsteriskServiceError(internalErrorResponse, errors.New("boom"))
	if internalErrorResponse.Code != http.StatusInternalServerError {
		t.Fatalf("expected internal asterisk error response, got %d", internalErrorResponse.Code)
	}
	notFoundErrorResponse := httptest.NewRecorder()
	writeAsteriskServiceError(notFoundErrorResponse, &service.AsteriskError{Code: "ASTERISK_SURFACE_NOT_FOUND", Message: "surface not found"})
	if notFoundErrorResponse.Code != http.StatusNotFound {
		t.Fatalf("expected asterisk not found response, got %d", notFoundErrorResponse.Code)
	}
	typedInternalErrorResponse := httptest.NewRecorder()
	writeAsteriskServiceError(typedInternalErrorResponse, &service.AsteriskError{Code: "ASTERISK_INTERNAL", Message: "internal"})
	if typedInternalErrorResponse.Code != http.StatusInternalServerError {
		t.Fatalf("expected typed internal asterisk error response, got %d", typedInternalErrorResponse.Code)
	}
}

func TestAsteriskAdminHandlerBranches(t *testing.T) {
	authService := newTestAuthService(t)
	adminCookie := SessionCookieConfig{Name: "admin_session", Path: "/admin", HTTPOnly: true, Secure: "auto", SameSite: http.SameSiteLaxMode}

	loginForm := httptest.NewRequest(http.MethodPost, "/admin", strings.NewReader("username=root-admin&password=correct+horse+battery+staple"))
	loginForm.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginResponse := httptest.NewRecorder()
	NewAdminHandler("/admin", authService, testDomainService(), testAsteriskService(), testPBXService(), adminCookie).ServeHTTP(loginResponse, loginForm)
	adminSessionCookie := loginResponse.Result().Cookies()[0]

	adminHandlerValue := NewAdminHandler("/admin", authService, testDomainService(), testAsteriskService(), testPBXService(), adminCookie).(AdminHandler)
	overviewRequest := httptest.NewRequest(http.MethodGet, "/admin/server/asterisk", nil)
	overviewRequest.AddCookie(adminSessionCookie)
	overviewResponse := httptest.NewRecorder()
	adminHandlerValue.ServeHTTP(overviewResponse, overviewRequest)
	if overviewResponse.Code != http.StatusOK || !strings.Contains(overviewResponse.Body.String(), "Asterisk surface: Overview") {
		t.Fatalf("unexpected asterisk overview response %d %q", overviewResponse.Code, overviewResponse.Body.String())
	}

	jsonRequest := httptest.NewRequest(http.MethodGet, "/admin/server/asterisk/media", nil)
	jsonRequest.AddCookie(adminSessionCookie)
	jsonRequest.Header.Set("Accept", "application/json")
	jsonResponse := httptest.NewRecorder()
	adminHandlerValue.ServeHTTP(jsonResponse, jsonRequest)
	if jsonResponse.Code != http.StatusOK || !strings.Contains(jsonResponse.Body.String(), "\"key\":\"media\"") {
		t.Fatalf("unexpected asterisk media response %d %q", jsonResponse.Code, jsonResponse.Body.String())
	}

	methodRequest := httptest.NewRequest(http.MethodPost, "/admin/server/asterisk/fax", nil)
	methodRequest.AddCookie(adminSessionCookie)
	methodResponse := httptest.NewRecorder()
	adminHandlerValue.ServeHTTP(methodResponse, methodRequest)
	if methodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected asterisk method rejection, got %d", methodResponse.Code)
	}

	adminUnauthorizedResponse := httptest.NewRecorder()
	adminHandlerValue.handleAsteriskSurface(adminUnauthorizedResponse, httptest.NewRequest(http.MethodGet, "/admin/server/asterisk/fax", nil), "server/asterisk/fax")
	if adminUnauthorizedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected admin asterisk unauthorized response, got %d", adminUnauthorizedResponse.Code)
	}

	notFoundRouteResponse := httptest.NewRecorder()
	adminHandlerValue.handleAsteriskSurface(notFoundRouteResponse, func() *http.Request {
		request := httptest.NewRequest(http.MethodGet, "/admin/server/settings", nil)
		request.AddCookie(adminSessionCookie)
		return request
	}(), "server/settings")
	if notFoundRouteResponse.Code != http.StatusNotFound {
		t.Fatalf("expected admin asterisk non-route response, got %d", notFoundRouteResponse.Code)
	}

	missingRequest := httptest.NewRequest(http.MethodGet, "/admin/server/asterisk/hardware", nil)
	missingRequest.AddCookie(adminSessionCookie)
	missingResponse := httptest.NewRecorder()
	AdminHandler{
		routePrefix:     "/admin",
		authService:     authService,
		domainService:   testDomainService(),
		asteriskService: service.NewAsteriskService(failingAsteriskStateStore{err: errors.New("lookup failed")}),
		pbxService:      testPBXService(),
		adminCookie:     adminCookie,
	}.handleAsteriskSurface(missingResponse, missingRequest, "server/asterisk/fax")
	if missingResponse.Code != http.StatusInternalServerError {
		t.Fatalf("expected failing asterisk lookup response 500, got %d", missingResponse.Code)
	}

	unavailableResponse := httptest.NewRecorder()
	AdminHandler{
		routePrefix:     "/admin",
		authService:     authService,
		domainService:   testDomainService(),
		asteriskService: service.NewAsteriskService(failingAsteriskStateStore{err: nil}),
		pbxService:      testPBXService(),
		adminCookie:     adminCookie,
	}.handleAsteriskSurface(unavailableResponse, func() *http.Request {
		request := httptest.NewRequest(http.MethodGet, "/admin/server/asterisk/fax", nil)
		request.AddCookie(adminSessionCookie)
		return request
	}(), "server/asterisk/fax")
	if unavailableResponse.Code != http.StatusNotFound {
		t.Fatalf("expected unavailable asterisk surface response 404, got %d", unavailableResponse.Code)
	}

	apiTokenOwner, adminToken, tokenError := authService.AuthenticateAdminAPI(context.Background(), "root-admin", "correct horse battery staple")
	if tokenError != nil || apiTokenOwner.Username != "root-admin" {
		t.Fatalf("expected admin api token, got %v / %+v", tokenError, apiTokenOwner)
	}
	apiHandlerValue := NewAPIAdminHandler("/api/v1/admin/server/asterisk", authService, testDomainService(), testAsteriskService(), testPBXService()).(APIAdminHandler)
	apiRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/server/asterisk/fax", nil)
	apiRequest.Header.Set("Authorization", "Bearer "+adminToken.Value)
	apiResponse := httptest.NewRecorder()
	apiHandlerValue.ServeHTTP(apiResponse, apiRequest)
	if apiResponse.Code != http.StatusOK || !strings.Contains(apiResponse.Body.String(), "\"key\":\"fax\"") {
		t.Fatalf("unexpected api asterisk surface response %d %q", apiResponse.Code, apiResponse.Body.String())
	}

	apiUnauthorizedResponse := httptest.NewRecorder()
	apiHandlerValue.handleAsteriskSurface(apiUnauthorizedResponse, httptest.NewRequest(http.MethodGet, "/api/v1/admin/server/asterisk/fax", nil), "fax")
	if apiUnauthorizedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected api asterisk unauthorized response, got %d", apiUnauthorizedResponse.Code)
	}

	apiNotFoundRouteResponse := httptest.NewRecorder()
	apiRequestForNotFound := httptest.NewRequest(http.MethodGet, "/api/v1/admin/server/settings", nil)
	apiRequestForNotFound.Header.Set("Authorization", "Bearer "+adminToken.Value)
	APIAdminHandler{
		routePrefix:     "/api/v1/admin",
		authService:     authService,
		domainService:   testDomainService(),
		asteriskService: testAsteriskService(),
		pbxService:      testPBXService(),
	}.handleAsteriskSurface(apiNotFoundRouteResponse, apiRequestForNotFound, "server/settings")
	if apiNotFoundRouteResponse.Code != http.StatusNotFound {
		t.Fatalf("expected api asterisk non-route response, got %d", apiNotFoundRouteResponse.Code)
	}

	apiMethodResponse := httptest.NewRecorder()
	apiMethodRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/server/asterisk/fax", nil)
	apiMethodRequest.Header.Set("Authorization", "Bearer "+adminToken.Value)
	apiHandlerValue.ServeHTTP(apiMethodResponse, apiMethodRequest)
	if apiMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected api asterisk method rejection, got %d", apiMethodResponse.Code)
	}

	apiUnavailableResponse := httptest.NewRecorder()
	apiUnavailableRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/server/asterisk/fax", nil)
	apiUnavailableRequest.Header.Set("Authorization", "Bearer "+adminToken.Value)
	APIAdminHandler{
		routePrefix:     "/api/v1/admin/server/asterisk",
		authService:     authService,
		domainService:   testDomainService(),
		asteriskService: service.NewAsteriskService(failingAsteriskStateStore{}),
		pbxService:      testPBXService(),
	}.handleAsteriskSurface(apiUnavailableResponse, apiUnavailableRequest, "fax")
	if apiUnavailableResponse.Code != http.StatusNotFound {
		t.Fatalf("expected api unavailable asterisk surface response, got %d", apiUnavailableResponse.Code)
	}
}

var _ = time.Time{}
