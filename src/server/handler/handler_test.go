package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/casapps/caspbx/src/server/model"
	"github.com/casapps/caspbx/src/server/service"
	"github.com/casapps/caspbx/src/server/store"
)

type errReadCloser struct{}

func (errReadCloser) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (errReadCloser) Close() error             { return nil }

func TestNewRootHandler(t *testing.T) {
	handlerValue := NewRootHandler("caspbx", "https://example.invalid", "/admin", "/api/v1")

	textRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	textResponse := httptest.NewRecorder()
	handlerValue.ServeHTTP(textResponse, textRequest)
	if textResponse.Code != http.StatusOK {
		t.Fatalf("expected root status 200, got %d", textResponse.Code)
	}
	if !strings.Contains(textResponse.Body.String(), "Admin path: /admin") {
		t.Fatalf("unexpected root text body %q", textResponse.Body.String())
	}

	jsonRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	jsonRequest.Header.Set("Accept", "application/json")
	jsonResponse := httptest.NewRecorder()
	handlerValue.ServeHTTP(jsonResponse, jsonRequest)
	if !strings.Contains(jsonResponse.Body.String(), "\"api_base_path\":\"/api/v1\"") {
		t.Fatalf("unexpected root json body %q", jsonResponse.Body.String())
	}

	postRequest := httptest.NewRequest(http.MethodPost, "/", nil)
	postResponse := httptest.NewRecorder()
	handlerValue.ServeHTTP(postResponse, postRequest)
	if postResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected post root status 405, got %d", postResponse.Code)
	}
}

func TestNewHealthHandler(t *testing.T) {
	handlerValue := NewHealthHandler(HealthResponse{
		Status:                "ok",
		Project:               "caspbx",
		Version:               "dev",
		CommitID:              "unknown",
		APIBasePath:           "/api/v1",
		AdminPath:             "/admin",
		AsteriskAdminPath:     "/admin/server/asterisk",
		RuntimeImplementation: "scaffold",
	})

	textRequest := httptest.NewRequest(http.MethodGet, "/health", nil)
	textRequest.Header.Set("Accept", "text/plain")
	textResponse := httptest.NewRecorder()
	handlerValue.ServeHTTP(textResponse, textRequest)
	if !strings.Contains(textResponse.Body.String(), "status=ok") {
		t.Fatalf("unexpected health text body %q", textResponse.Body.String())
	}

	jsonRequest := httptest.NewRequest(http.MethodGet, "/health", nil)
	jsonRequest.Header.Set("Accept", "application/json")
	jsonResponse := httptest.NewRecorder()
	handlerValue.ServeHTTP(jsonResponse, jsonRequest)
	if !strings.Contains(jsonResponse.Body.String(), "\"asterisk_admin_path\":\"/admin/server/asterisk\"") {
		t.Fatalf("unexpected health json body %q", jsonResponse.Body.String())
	}

	headRequest := httptest.NewRequest(http.MethodHead, "/health", nil)
	headResponse := httptest.NewRecorder()
	handlerValue.ServeHTTP(headResponse, headRequest)
	if headResponse.Code != http.StatusOK {
		t.Fatalf("expected head health status 200, got %d", headResponse.Code)
	}

	postRequest := httptest.NewRequest(http.MethodPost, "/health", nil)
	postResponse := httptest.NewRecorder()
	handlerValue.ServeHTTP(postResponse, postRequest)
	if postResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected post health status 405, got %d", postResponse.Code)
	}
}

func TestNewVersionHandler(t *testing.T) {
	handlerValue := NewVersionHandler("caspbx", "1.0.0", "abcdef")

	textRequest := httptest.NewRequest(http.MethodGet, "/version", nil)
	textResponse := httptest.NewRecorder()
	handlerValue.ServeHTTP(textResponse, textRequest)
	if !strings.Contains(textResponse.Body.String(), "caspbx 1.0.0 (abcdef)") {
		t.Fatalf("unexpected version text body %q", textResponse.Body.String())
	}

	jsonRequest := httptest.NewRequest(http.MethodGet, "/version", nil)
	jsonRequest.Header.Set("Accept", "application/json")
	jsonResponse := httptest.NewRecorder()
	handlerValue.ServeHTTP(jsonResponse, jsonRequest)
	if !strings.Contains(jsonResponse.Body.String(), "\"commit\":\"abcdef\"") {
		t.Fatalf("unexpected version json body %q", jsonResponse.Body.String())
	}

	postRequest := httptest.NewRequest(http.MethodPost, "/version", nil)
	postResponse := httptest.NewRecorder()
	handlerValue.ServeHTTP(postResponse, postRequest)
	if postResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected version post to fail, got %d", postResponse.Code)
	}
}

func TestPlaceholderHandlers(t *testing.T) {
	handlers := []struct {
		name         string
		handlerValue http.Handler
		path         string
		expectText   string
	}{
		{name: "placeholder", handlerValue: NewPlaceholderHandler("surface", "/surface"), path: "/surface", expectText: "surface surface is scaffolded at /surface"},
		{name: "api", handlerValue: NewAPIHandler("auth_api", "/api/v1/auth"), path: "/api/v1/auth/login", expectText: "auth_api surface is scaffolded at /api/v1/auth"},
	}

	for _, testCase := range handlers {
		textRequest := httptest.NewRequest(http.MethodGet, testCase.path, nil)
		textRequest.Header.Set("Accept", "text/plain")
		textResponse := httptest.NewRecorder()
		testCase.handlerValue.ServeHTTP(textResponse, textRequest)
		if textResponse.Code != http.StatusNotImplemented {
			t.Fatalf("%s: expected status 501, got %d", testCase.name, textResponse.Code)
		}
		if !strings.Contains(textResponse.Body.String(), testCase.expectText) {
			t.Fatalf("%s: unexpected text body %q", testCase.name, textResponse.Body.String())
		}

		jsonRequest := httptest.NewRequest(http.MethodGet, testCase.path, nil)
		jsonRequest.Header.Set("Accept", "application/json")
		jsonResponse := httptest.NewRecorder()
		testCase.handlerValue.ServeHTTP(jsonResponse, jsonRequest)
		if !strings.Contains(jsonResponse.Body.String(), "\"status\":\"not_implemented\"") {
			t.Fatalf("%s: unexpected json body %q", testCase.name, jsonResponse.Body.String())
		}

		postRequest := httptest.NewRequest(http.MethodPost, testCase.path, nil)
		postResponse := httptest.NewRecorder()
		testCase.handlerValue.ServeHTTP(postResponse, postRequest)
		if postResponse.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s: expected post status 405, got %d", testCase.name, postResponse.Code)
		}
	}
}

func TestAuthUserAndAdminHandlers(t *testing.T) {
	authService := newTestAuthService(t)
	userCookie := SessionCookieConfig{Name: "user_session", Path: "/", HTTPOnly: true, Secure: "auto", SameSite: http.SameSiteLaxMode}
	adminCookie := SessionCookieConfig{Name: "admin_session", Path: "/admin", HTTPOnly: true, Secure: "auto", SameSite: http.SameSiteLaxMode}

	authHandler := NewAuthHandler("/auth", authService, userCookie, model.RegistrationModePrivate)
	userHandler := NewUserHandler("/users", authService, testDomainService(), userCookie)
	adminHandler := NewAdminHandler("/admin", authService, testDomainService(), testAsteriskService(), testPBXService(), adminCookie)

	authOverviewRequest := httptest.NewRequest(http.MethodGet, "/auth", nil)
	authOverviewResponse := httptest.NewRecorder()
	authHandler.ServeHTTP(authOverviewResponse, authOverviewRequest)
	if !strings.Contains(authOverviewResponse.Body.String(), "Registration mode: private") {
		t.Fatalf("unexpected auth overview body %q", authOverviewResponse.Body.String())
	}

	registerRequest := httptest.NewRequest(http.MethodPost, "/auth/register", nil)
	registerResponse := httptest.NewRecorder()
	authHandler.ServeHTTP(registerResponse, registerRequest)
	if registerResponse.Code != http.StatusForbidden {
		t.Fatalf("expected register status 403, got %d", registerResponse.Code)
	}

	badLoginRequest := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader("{"))
	badLoginRequest.Header.Set("Content-Type", "application/json")
	badLoginResponse := httptest.NewRecorder()
	authHandler.ServeHTTP(badLoginResponse, badLoginRequest)
	if badLoginResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected bad login status 400, got %d", badLoginResponse.Code)
	}

	loginForm := url.Values{"identifier": {"alice"}, "password": {"correct horse battery staple"}}
	loginRequest := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(loginForm.Encode()))
	loginRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginResponse := httptest.NewRecorder()
	authHandler.ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d", loginResponse.Code)
	}
	userSessionCookie := loginResponse.Result().Cookies()[0]
	if userSessionCookie.Name != "user_session" {
		t.Fatalf("unexpected user session cookie %+v", userSessionCookie)
	}

	userUnauthorizedRequest := httptest.NewRequest(http.MethodGet, "/users/profile", nil)
	userUnauthorizedResponse := httptest.NewRecorder()
	userHandler.ServeHTTP(userUnauthorizedResponse, userUnauthorizedRequest)
	if userUnauthorizedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized user profile status 401, got %d", userUnauthorizedResponse.Code)
	}

	userProfileRequest := httptest.NewRequest(http.MethodGet, "/users/profile", nil)
	userProfileRequest.AddCookie(userSessionCookie)
	userProfileRequest.Header.Set("Accept", "application/json")
	userProfileResponse := httptest.NewRecorder()
	userHandler.ServeHTTP(userProfileResponse, userProfileRequest)
	if !strings.Contains(userProfileResponse.Body.String(), "\"username\":\"alice\"") {
		t.Fatalf("unexpected user profile body %q", userProfileResponse.Body.String())
	}

	userNotFoundRequest := httptest.NewRequest(http.MethodGet, "/users/missing", nil)
	userNotFoundRequest.AddCookie(userSessionCookie)
	userNotFoundResponse := httptest.NewRecorder()
	userHandler.ServeHTTP(userNotFoundResponse, userNotFoundRequest)
	if userNotFoundResponse.Code != http.StatusNotFound {
		t.Fatalf("expected unknown user route status 404, got %d", userNotFoundResponse.Code)
	}

	userExpiredRequest := httptest.NewRequest(http.MethodGet, "/users/profile", nil)
	userExpiredRequest.AddCookie(&http.Cookie{Name: "user_session", Value: "missing"})
	userExpiredResponse := httptest.NewRecorder()
	userHandler.ServeHTTP(userExpiredResponse, userExpiredRequest)
	if userExpiredResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected expired user profile status 401, got %d", userExpiredResponse.Code)
	}
	if len(userExpiredResponse.Result().Cookies()) == 0 || userExpiredResponse.Result().Cookies()[0].MaxAge != -1 {
		t.Fatalf("expected expired user session cookie to be cleared")
	}

	refreshRequest := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	refreshRequest.AddCookie(userSessionCookie)
	refreshResponse := httptest.NewRecorder()
	authHandler.ServeHTTP(refreshResponse, refreshRequest)
	if refreshResponse.Code != http.StatusOK {
		t.Fatalf("expected refresh status 200, got %d", refreshResponse.Code)
	}

	logoutUnauthorizedRequest := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	logoutUnauthorizedResponse := httptest.NewRecorder()
	authHandler.ServeHTTP(logoutUnauthorizedResponse, logoutUnauthorizedRequest)
	if logoutUnauthorizedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected logout without cookie to fail, got %d", logoutUnauthorizedResponse.Code)
	}

	logoutRequest := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	logoutRequest.AddCookie(userSessionCookie)
	logoutResponse := httptest.NewRecorder()
	authHandler.ServeHTTP(logoutResponse, logoutRequest)
	if logoutResponse.Code != http.StatusOK {
		t.Fatalf("expected logout status 200, got %d", logoutResponse.Code)
	}

	adminUnauthorizedRequest := httptest.NewRequest(http.MethodGet, "/admin", nil)
	adminUnauthorizedResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(adminUnauthorizedResponse, adminUnauthorizedRequest)
	if adminUnauthorizedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected admin auth required status 401, got %d", adminUnauthorizedResponse.Code)
	}

	adminLoginForm := url.Values{"username": {"root-admin"}, "password": {"correct horse battery staple"}}
	adminLoginRequest := httptest.NewRequest(http.MethodPost, "/admin", strings.NewReader(adminLoginForm.Encode()))
	adminLoginRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	adminLoginResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(adminLoginResponse, adminLoginRequest)
	if adminLoginResponse.Code != http.StatusOK {
		t.Fatalf("expected admin login status 200, got %d", adminLoginResponse.Code)
	}
	adminSessionCookie := adminLoginResponse.Result().Cookies()[0]

	adminRootRequest := httptest.NewRequest(http.MethodGet, "/admin", nil)
	adminRootRequest.AddCookie(adminSessionCookie)
	adminRootResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(adminRootResponse, adminRootRequest)
	if !strings.Contains(adminRootResponse.Body.String(), "\"scope\":\"admin\"") {
		t.Fatalf("unexpected admin root body %q", adminRootResponse.Body.String())
	}

	adminProfileRequest := httptest.NewRequest(http.MethodGet, "/admin/profile", nil)
	adminProfileRequest.AddCookie(adminSessionCookie)
	adminProfileResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(adminProfileResponse, adminProfileRequest)
	if !strings.Contains(adminProfileResponse.Body.String(), "Admin: root-admin") {
		t.Fatalf("unexpected admin profile body %q", adminProfileResponse.Body.String())
	}

	adminSurfaceRequest := httptest.NewRequest(http.MethodGet, "/admin/server/asterisk/fax", nil)
	adminSurfaceRequest.AddCookie(adminSessionCookie)
	adminSurfaceResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(adminSurfaceResponse, adminSurfaceRequest)
	if !strings.Contains(adminSurfaceResponse.Body.String(), "Asterisk surface: Fax") {
		t.Fatalf("unexpected admin surface body %q", adminSurfaceResponse.Body.String())
	}
	adminServerSurfaceRequest := httptest.NewRequest(http.MethodGet, "/admin/server", nil)
	adminServerSurfaceRequest.AddCookie(adminSessionCookie)
	adminServerSurfaceResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(adminServerSurfaceResponse, adminServerSurfaceRequest)
	if !strings.Contains(adminServerSurfaceResponse.Body.String(), "Admin surface: server") {
		t.Fatalf("unexpected admin server surface body %q", adminServerSurfaceResponse.Body.String())
	}

	adminProtectedMethodRequest := httptest.NewRequest(http.MethodPost, "/admin/profile", nil)
	adminProtectedMethodRequest.AddCookie(adminSessionCookie)
	adminProtectedMethodResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(adminProtectedMethodResponse, adminProtectedMethodRequest)
	if adminProtectedMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected admin profile post status 405, got %d", adminProtectedMethodResponse.Code)
	}

	adminMissingRouteRequest := httptest.NewRequest(http.MethodGet, "/admin/missing", nil)
	adminMissingRouteRequest.AddCookie(adminSessionCookie)
	adminMissingRouteResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(adminMissingRouteResponse, adminMissingRouteRequest)
	if adminMissingRouteResponse.Code != http.StatusNotFound {
		t.Fatalf("expected missing admin route status 404, got %d", adminMissingRouteResponse.Code)
	}
}

func TestAPIAuthUserAndAdminHandlers(t *testing.T) {
	authService := newTestAuthService(t)
	authAPIHandler := NewAPIAuthHandler("/api/v1/auth", authService, model.RegistrationModePrivate)
	userAPIHandler := NewAPIUserHandler("/api/v1/users", authService, testDomainService())
	adminAPIHandler := NewAPIAdminHandler("/api/v1/admin", authService, testDomainService(), testAsteriskService(), testPBXService())
	asteriskAdminAPIHandler := NewAPIAdminHandler("/api/v1/admin/server/asterisk", authService, testDomainService(), testAsteriskService(), testPBXService())

	overviewRequest := httptest.NewRequest(http.MethodGet, "/api/v1/auth", nil)
	overviewResponse := httptest.NewRecorder()
	authAPIHandler.ServeHTTP(overviewResponse, overviewRequest)
	if !strings.Contains(overviewResponse.Body.String(), "\"scope\":\"auth_api\"") {
		t.Fatalf("unexpected auth api overview body %q", overviewResponse.Body.String())
	}

	loginInfoRequest := httptest.NewRequest(http.MethodGet, "/api/v1/auth/login", nil)
	loginInfoResponse := httptest.NewRecorder()
	authAPIHandler.ServeHTTP(loginInfoResponse, loginInfoRequest)
	if !strings.Contains(loginInfoResponse.Body.String(), "\"auth_model\":\"Bearer usr_ token\"") {
		t.Fatalf("unexpected auth api login info %q", loginInfoResponse.Body.String())
	}

	loginForm := url.Values{"identifier": {"alice"}, "password": {"correct horse battery staple"}}
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginForm.Encode()))
	loginRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginResponse := httptest.NewRecorder()
	authAPIHandler.ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusOK || !strings.Contains(loginResponse.Body.String(), "\"token_type\":\"Bearer\"") {
		t.Fatalf("unexpected auth api login response %d %q", loginResponse.Code, loginResponse.Body.String())
	}
	userBearer := strings.Split(strings.Split(loginResponse.Body.String(), "\"token\":\"")[1], "\"")[0]

	userProfileRequest := httptest.NewRequest(http.MethodGet, "/api/v1/users/profile", nil)
	userProfileRequest.Header.Set("Authorization", "Bearer "+userBearer)
	userProfileResponse := httptest.NewRecorder()
	userAPIHandler.ServeHTTP(userProfileResponse, userProfileRequest)
	if !strings.Contains(userProfileResponse.Body.String(), "\"username\":\"alice\"") {
		t.Fatalf("unexpected user api profile body %q", userProfileResponse.Body.String())
	}

	refreshRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	refreshRequest.Header.Set("Authorization", "Bearer "+userBearer)
	refreshResponse := httptest.NewRecorder()
	authAPIHandler.ServeHTTP(refreshResponse, refreshRequest)
	if refreshResponse.Code != http.StatusOK || !strings.Contains(refreshResponse.Body.String(), "\"status\":\"refreshed\"") {
		t.Fatalf("unexpected auth api refresh response %d %q", refreshResponse.Code, refreshResponse.Body.String())
	}
	refreshedBearer := strings.Split(strings.Split(refreshResponse.Body.String(), "\"token\":\"")[1], "\"")[0]

	staleUserRequest := httptest.NewRequest(http.MethodGet, "/api/v1/users/profile", nil)
	staleUserRequest.Header.Set("Authorization", "Bearer "+userBearer)
	staleUserResponse := httptest.NewRecorder()
	userAPIHandler.ServeHTTP(staleUserResponse, staleUserRequest)
	if staleUserResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected stale user token to fail, got %d", staleUserResponse.Code)
	}

	logoutRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutRequest.Header.Set("Authorization", "Bearer "+refreshedBearer)
	logoutResponse := httptest.NewRecorder()
	authAPIHandler.ServeHTTP(logoutResponse, logoutRequest)
	if logoutResponse.Code != http.StatusOK {
		t.Fatalf("expected auth api logout status 200, got %d", logoutResponse.Code)
	}

	adminAccount, adminBearerToken, adminTokenError := authService.AuthenticateAdminAPI(context.Background(), "root-admin", "correct horse battery staple")
	if adminTokenError != nil || adminAccount.Username != "root-admin" {
		t.Fatalf("expected admin api token, got %v / %+v", adminTokenError, adminAccount)
	}

	adminRootRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin", nil)
	adminRootRequest.Header.Set("Authorization", "Bearer "+adminBearerToken.Value)
	adminRootResponse := httptest.NewRecorder()
	adminAPIHandler.ServeHTTP(adminRootResponse, adminRootRequest)
	if !strings.Contains(adminRootResponse.Body.String(), "\"scope\":\"admin_api\"") {
		t.Fatalf("unexpected admin api root body %q", adminRootResponse.Body.String())
	}

	adminProfileRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/profile", nil)
	adminProfileRequest.Header.Set("Authorization", "Bearer "+adminBearerToken.Value)
	adminProfileResponse := httptest.NewRecorder()
	adminAPIHandler.ServeHTTP(adminProfileResponse, adminProfileRequest)
	if !strings.Contains(adminProfileResponse.Body.String(), "\"role\":\"admin\"") {
		t.Fatalf("unexpected admin api profile body %q", adminProfileResponse.Body.String())
	}

	adminSurfaceRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/server/security/auth", nil)
	adminSurfaceRequest.Header.Set("Authorization", "Bearer "+adminBearerToken.Value)
	adminSurfaceResponse := httptest.NewRecorder()
	adminAPIHandler.ServeHTTP(adminSurfaceResponse, adminSurfaceRequest)
	if !strings.Contains(adminSurfaceResponse.Body.String(), "\"surface\":\"server/security/auth\"") {
		t.Fatalf("unexpected admin api surface body %q", adminSurfaceResponse.Body.String())
	}
	adminServerRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/server", nil)
	adminServerRequest.Header.Set("Authorization", "Bearer "+adminBearerToken.Value)
	adminServerResponse := httptest.NewRecorder()
	adminAPIHandler.ServeHTTP(adminServerResponse, adminServerRequest)
	if !strings.Contains(adminServerResponse.Body.String(), "\"surface\":\"server\"") {
		t.Fatalf("unexpected admin api server body %q", adminServerResponse.Body.String())
	}

	asteriskSurfaceRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/server/asterisk/fax", nil)
	asteriskSurfaceRequest.Header.Set("Authorization", "Bearer "+adminBearerToken.Value)
	asteriskSurfaceResponse := httptest.NewRecorder()
	asteriskAdminAPIHandler.ServeHTTP(asteriskSurfaceResponse, asteriskSurfaceRequest)
	if !strings.Contains(asteriskSurfaceResponse.Body.String(), "\"key\":\"fax\"") {
		t.Fatalf("unexpected asterisk admin api body %q", asteriskSurfaceResponse.Body.String())
	}
}

func TestOrgHandlers(t *testing.T) {
	memoryStore, authService := newTestRuntimeStore(t)
	userCookie := SessionCookieConfig{Name: "user_session", Path: "/", HTTPOnly: true, Secure: "auto", SameSite: http.SameSiteLaxMode}

	orgHandler := NewOrgHandler("/orgs", authService, newEnabledDomainService(memoryStore), memoryStore, userCookie)
	orgAPIHandler := NewAPIOrgHandler("/api/v1/orgs", authService, newEnabledDomainService(memoryStore), memoryStore)

	_, aliceSession, loginError := authService.AuthenticateUser(context.Background(), "alice", "correct horse battery staple", "127.0.0.1", "curl/8.0")
	if loginError != nil {
		t.Fatalf("authenticate user session: %v", loginError)
	}
	_, aliceToken, tokenError := authService.AuthenticateUserAPI(context.Background(), "alice", "correct horse battery staple")
	if tokenError != nil {
		t.Fatalf("authenticate user api token: %v", tokenError)
	}

	publicOrgRequest := httptest.NewRequest(http.MethodGet, "/orgs/acme", nil)
	publicOrgResponse := httptest.NewRecorder()
	orgHandler.ServeHTTP(publicOrgResponse, publicOrgRequest)
	if publicOrgResponse.Code != http.StatusOK || !strings.Contains(publicOrgResponse.Body.String(), "Organization: Acme") {
		t.Fatalf("unexpected public org response %d %q", publicOrgResponse.Code, publicOrgResponse.Body.String())
	}

	publicMembersRequest := httptest.NewRequest(http.MethodGet, "/orgs/acme/members", nil)
	publicMembersRequest.Header.Set("Accept", "application/json")
	publicMembersResponse := httptest.NewRecorder()
	orgHandler.ServeHTTP(publicMembersResponse, publicMembersRequest)
	if !strings.Contains(publicMembersResponse.Body.String(), "\"username\":\"alice\"") || strings.Contains(publicMembersResponse.Body.String(), "\"username\":\"bob\"") {
		t.Fatalf("unexpected public org members response %q", publicMembersResponse.Body.String())
	}

	memberMembersRequest := httptest.NewRequest(http.MethodGet, "/orgs/acme/members", nil)
	memberMembersRequest.Header.Set("Accept", "application/json")
	memberMembersRequest.AddCookie(userCookie.Build(memberMembersRequest, aliceSession.Token, aliceSession.Session.ExpiresAt))
	memberMembersResponse := httptest.NewRecorder()
	orgHandler.ServeHTTP(memberMembersResponse, memberMembersRequest)
	if !strings.Contains(memberMembersResponse.Body.String(), "\"username\":\"bob\"") {
		t.Fatalf("expected member-scoped org members response to include private member, got %q", memberMembersResponse.Body.String())
	}

	privateOrgRequest := httptest.NewRequest(http.MethodGet, "/orgs/secret", nil)
	privateOrgResponse := httptest.NewRecorder()
	orgHandler.ServeHTTP(privateOrgResponse, privateOrgRequest)
	if privateOrgResponse.Code != http.StatusNotFound {
		t.Fatalf("expected hidden private org response, got %d", privateOrgResponse.Code)
	}

	privateSettingsRequest := httptest.NewRequest(http.MethodGet, "/orgs/secret/settings", nil)
	privateSettingsRequest.AddCookie(userCookie.Build(privateSettingsRequest, aliceSession.Token, aliceSession.Session.ExpiresAt))
	privateSettingsResponse := httptest.NewRecorder()
	orgHandler.ServeHTTP(privateSettingsResponse, privateSettingsRequest)
	if privateSettingsResponse.Code != http.StatusOK || !strings.Contains(privateSettingsResponse.Body.String(), "\"can_delete_org\":true") {
		t.Fatalf("unexpected org settings response %d %q", privateSettingsResponse.Code, privateSettingsResponse.Body.String())
	}

	publicMembersAPIRequest := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/acme/members", nil)
	publicMembersAPIResponse := httptest.NewRecorder()
	orgAPIHandler.ServeHTTP(publicMembersAPIResponse, publicMembersAPIRequest)
	if publicMembersAPIResponse.Code != http.StatusOK || strings.Contains(publicMembersAPIResponse.Body.String(), "\"username\":\"bob\"") {
		t.Fatalf("unexpected public org api members response %d %q", publicMembersAPIResponse.Code, publicMembersAPIResponse.Body.String())
	}

	privateAPIRequest := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/secret", nil)
	privateAPIResponse := httptest.NewRecorder()
	orgAPIHandler.ServeHTTP(privateAPIResponse, privateAPIRequest)
	if privateAPIResponse.Code != http.StatusNotFound {
		t.Fatalf("expected hidden private org api response, got %d", privateAPIResponse.Code)
	}

	privateSettingsAPIRequest := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/secret/settings", nil)
	privateSettingsAPIRequest.Header.Set("Authorization", "Bearer "+aliceToken.Value)
	privateSettingsAPIResponse := httptest.NewRecorder()
	orgAPIHandler.ServeHTTP(privateSettingsAPIResponse, privateSettingsAPIRequest)
	if privateSettingsAPIResponse.Code != http.StatusOK || !strings.Contains(privateSettingsAPIResponse.Body.String(), "\"allow_invites\":true") {
		t.Fatalf("unexpected private org api settings response %d %q", privateSettingsAPIResponse.Code, privateSettingsAPIResponse.Body.String())
	}

	orgTokenValue := "org_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if _, saveTokenError := memoryStore.SaveToken(context.Background(), model.Token{
		OwnerType:   model.TokenOwnerOrg,
		OwnerID:     2,
		Name:        "default",
		TokenHash:   service.HashToken(orgTokenValue),
		TokenPrefix: "org_aaaa",
		Scope:       model.TokenScopeGlobal,
		ExpiresAt:   time.Unix(2_000_000_000, 0),
	}); saveTokenError != nil {
		t.Fatalf("save org token: %v", saveTokenError)
	}

	orgTokenSettingsRequest := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/secret/settings", nil)
	orgTokenSettingsRequest.Header.Set("Authorization", "Bearer "+orgTokenValue)
	orgTokenSettingsResponse := httptest.NewRecorder()
	orgAPIHandler.ServeHTTP(orgTokenSettingsResponse, orgTokenSettingsRequest)
	if orgTokenSettingsResponse.Code != http.StatusOK || !strings.Contains(orgTokenSettingsResponse.Body.String(), "\"can_manage_members\":true") {
		t.Fatalf("unexpected org-token settings response %d %q", orgTokenSettingsResponse.Code, orgTokenSettingsResponse.Body.String())
	}

	publicMemberDetailRequest := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/acme/members/1", nil)
	publicMemberDetailResponse := httptest.NewRecorder()
	orgAPIHandler.ServeHTTP(publicMemberDetailResponse, publicMemberDetailRequest)
	if publicMemberDetailResponse.Code != http.StatusOK || !strings.Contains(publicMemberDetailResponse.Body.String(), "\"username\":\"alice\"") {
		t.Fatalf("unexpected org member detail response %d %q", publicMemberDetailResponse.Code, publicMemberDetailResponse.Body.String())
	}
}

func TestAPIHandlerAdditionalBranches(t *testing.T) {
	authService := newTestAuthService(t)
	authAPIHandler := NewAPIAuthHandler("/api/v1/auth", authService, model.RegistrationModePrivate)
	userAPIHandler := NewAPIUserHandler("/api/v1/users", authService, testDomainService())
	adminAPIHandler := NewAPIAdminHandler("/api/v1/admin", authService, testDomainService(), testAsteriskService(), testPBXService())

	badLoginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader("{"))
	badLoginRequest.Header.Set("Content-Type", "application/json")
	badLoginResponse := httptest.NewRecorder()
	authAPIHandler.ServeHTTP(badLoginResponse, badLoginRequest)
	if badLoginResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected api bad login status 400, got %d", badLoginResponse.Code)
	}

	invalidLoginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(url.Values{"identifier": {"alice"}, "password": {"wrong"}}.Encode()))
	invalidLoginRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	invalidLoginResponse := httptest.NewRecorder()
	authAPIHandler.ServeHTTP(invalidLoginResponse, invalidLoginRequest)
	if invalidLoginResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected api invalid login status 401, got %d", invalidLoginResponse.Code)
	}

	loginMethodRequest := httptest.NewRequest(http.MethodPut, "/api/v1/auth/login", nil)
	loginMethodResponse := httptest.NewRecorder()
	authAPIHandler.ServeHTTP(loginMethodResponse, loginMethodRequest)
	if loginMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected api login method status 405, got %d", loginMethodResponse.Code)
	}

	registerInfoRequest := httptest.NewRequest(http.MethodGet, "/api/v1/auth/register", nil)
	registerInfoResponse := httptest.NewRecorder()
	authAPIHandler.ServeHTTP(registerInfoResponse, registerInfoRequest)
	if !strings.Contains(registerInfoResponse.Body.String(), "\"registration_mode\":\"private\"") {
		t.Fatalf("unexpected api register info %q", registerInfoResponse.Body.String())
	}
	registerRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", nil)
	registerResponse := httptest.NewRecorder()
	authAPIHandler.ServeHTTP(registerResponse, registerRequest)
	if registerResponse.Code != http.StatusForbidden {
		t.Fatalf("expected api private register status 403, got %d", registerResponse.Code)
	}
	registerMethodRequest := httptest.NewRequest(http.MethodPut, "/api/v1/auth/register", nil)
	registerMethodResponse := httptest.NewRecorder()
	authAPIHandler.ServeHTTP(registerMethodResponse, registerMethodRequest)
	if registerMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected api register method status 405, got %d", registerMethodResponse.Code)
	}

	publicAPIAuthHandler := NewAPIAuthHandler("/api/v1/auth", authService, model.RegistrationModePublic)
	publicRegisterRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", nil)
	publicRegisterResponse := httptest.NewRecorder()
	publicAPIAuthHandler.ServeHTTP(publicRegisterResponse, publicRegisterRequest)
	if publicRegisterResponse.Code != http.StatusNotImplemented {
		t.Fatalf("expected api public register not implemented, got %d", publicRegisterResponse.Code)
	}
	trailingOverviewRequest := httptest.NewRequest(http.MethodGet, "/api/v1/auth/", nil)
	trailingOverviewResponse := httptest.NewRecorder()
	authAPIHandler.ServeHTTP(trailingOverviewResponse, trailingOverviewRequest)
	if trailingOverviewResponse.Code != http.StatusOK {
		t.Fatalf("expected api auth trailing overview 200, got %d", trailingOverviewResponse.Code)
	}
	overviewMethodRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth", nil)
	overviewMethodResponse := httptest.NewRecorder()
	authAPIHandler.ServeHTTP(overviewMethodResponse, overviewMethodRequest)
	if overviewMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected api auth overview method status 405, got %d", overviewMethodResponse.Code)
	}

	logoutUnauthorizedRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutUnauthorizedResponse := httptest.NewRecorder()
	authAPIHandler.ServeHTTP(logoutUnauthorizedResponse, logoutUnauthorizedRequest)
	if logoutUnauthorizedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected api logout auth failure, got %d", logoutUnauthorizedResponse.Code)
	}
	logoutInvalidTokenRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutInvalidTokenRequest.Header.Set("Authorization", "Bearer usr_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	logoutInvalidTokenResponse := httptest.NewRecorder()
	authAPIHandler.ServeHTTP(logoutInvalidTokenResponse, logoutInvalidTokenRequest)
	if logoutInvalidTokenResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected api logout invalid token failure, got %d", logoutInvalidTokenResponse.Code)
	}
	logoutMethodRequest := httptest.NewRequest(http.MethodGet, "/api/v1/auth/logout", nil)
	logoutMethodResponse := httptest.NewRecorder()
	authAPIHandler.ServeHTTP(logoutMethodResponse, logoutMethodRequest)
	if logoutMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected api logout method status 405, got %d", logoutMethodResponse.Code)
	}
	refreshMethodRequest := httptest.NewRequest(http.MethodGet, "/api/v1/auth/refresh", nil)
	refreshMethodResponse := httptest.NewRecorder()
	authAPIHandler.ServeHTTP(refreshMethodResponse, refreshMethodRequest)
	if refreshMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected api refresh method status 405, got %d", refreshMethodResponse.Code)
	}
	refreshInvalidTokenRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	refreshInvalidTokenRequest.Header.Set("Authorization", "Bearer usr_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	refreshInvalidTokenResponse := httptest.NewRecorder()
	authAPIHandler.ServeHTTP(refreshInvalidTokenResponse, refreshInvalidTokenRequest)
	if refreshInvalidTokenResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected api refresh invalid token failure, got %d", refreshInvalidTokenResponse.Code)
	}

	for _, path := range []string{"/api/v1/auth/2fa", "/api/v1/auth/passkey/challenge", "/api/v1/auth/passkey/verify", "/api/v1/auth/password/forgot", "/api/v1/auth/password/reset", "/api/v1/auth/username/forgot", "/api/v1/auth/recovery/use", "/api/v1/auth/verify", "/api/v1/auth/ldap", "/api/v1/auth/invite/user/token", "/api/v1/auth/invite/server/token", "/api/v1/auth/oidc/github"} {
		request := httptest.NewRequest(http.MethodPost, path, nil)
		if strings.Contains(path, "/oidc/") {
			request = httptest.NewRequest(http.MethodGet, path, nil)
		}
		response := httptest.NewRecorder()
		authAPIHandler.ServeHTTP(response, request)
		if response.Code != http.StatusNotImplemented {
			t.Fatalf("expected api not implemented for %s, got %d", path, response.Code)
		}
	}
	authUnknownRequest := httptest.NewRequest(http.MethodGet, "/api/v1/auth/missing", nil)
	authUnknownResponse := httptest.NewRecorder()
	authAPIHandler.ServeHTTP(authUnknownResponse, authUnknownRequest)
	if authUnknownResponse.Code != http.StatusNotFound {
		t.Fatalf("expected unknown api auth route 404, got %d", authUnknownResponse.Code)
	}

	userUnauthorizedRequest := httptest.NewRequest(http.MethodGet, "/api/v1/users/profile", nil)
	userUnauthorizedResponse := httptest.NewRecorder()
	userAPIHandler.ServeHTTP(userUnauthorizedResponse, userUnauthorizedRequest)
	if userUnauthorizedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized user api status 401, got %d", userUnauthorizedResponse.Code)
	}
	userMethodRequest := httptest.NewRequest(http.MethodPost, "/api/v1/users/profile", nil)
	userMethodResponse := httptest.NewRecorder()
	userAPIHandler.ServeHTTP(userMethodResponse, userMethodRequest)
	if userMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected user api method status 405, got %d", userMethodResponse.Code)
	}

	_, userTokenRecord, userTokenError := authService.AuthenticateUserAPI(context.Background(), "alice", "correct horse battery staple")
	if userTokenError != nil {
		t.Fatalf("authenticate api user token: %v", userTokenError)
	}
	userMissingRouteRequest := httptest.NewRequest(http.MethodGet, "/api/v1/users/missing", nil)
	userMissingRouteRequest.Header.Set("Authorization", "Bearer "+userTokenRecord.Value)
	userMissingRouteResponse := httptest.NewRecorder()
	userAPIHandler.ServeHTTP(userMissingRouteResponse, userMissingRouteRequest)
	if userMissingRouteResponse.Code != http.StatusNotFound {
		t.Fatalf("expected unknown user api route 404, got %d", userMissingRouteResponse.Code)
	}

	adminUnauthorizedRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin", nil)
	adminUnauthorizedResponse := httptest.NewRecorder()
	adminAPIHandler.ServeHTTP(adminUnauthorizedResponse, adminUnauthorizedRequest)
	if adminUnauthorizedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized admin api status 401, got %d", adminUnauthorizedResponse.Code)
	}
	adminUnauthorizedProfileRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/profile", nil)
	adminUnauthorizedProfileResponse := httptest.NewRecorder()
	adminAPIHandler.ServeHTTP(adminUnauthorizedProfileResponse, adminUnauthorizedProfileRequest)
	if adminUnauthorizedProfileResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized admin api profile status 401, got %d", adminUnauthorizedProfileResponse.Code)
	}
	adminMethodRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin", nil)
	adminMethodResponse := httptest.NewRecorder()
	adminAPIHandler.ServeHTTP(adminMethodResponse, adminMethodRequest)
	if adminMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected admin api method status 405, got %d", adminMethodResponse.Code)
	}
	adminProfileMethodRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/profile", nil)
	adminProfileMethodResponse := httptest.NewRecorder()
	adminAPIHandler.ServeHTTP(adminProfileMethodResponse, adminProfileMethodRequest)
	if adminProfileMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected admin api profile method status 405, got %d", adminProfileMethodResponse.Code)
	}
	adminUnknownRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/missing", nil)
	adminUnknownResponse := httptest.NewRecorder()
	adminAPIHandler.ServeHTTP(adminUnknownResponse, adminUnknownRequest)
	if adminUnknownResponse.Code != http.StatusNotFound {
		t.Fatalf("expected unknown admin api route 404, got %d", adminUnknownResponse.Code)
	}

	_, adminTokenRecord, adminTokenError := authService.AuthenticateAdminAPI(context.Background(), "root-admin", "correct horse battery staple")
	if adminTokenError != nil {
		t.Fatalf("authenticate api admin token: %v", adminTokenError)
	}
	adminProfileRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/profile/security", nil)
	adminProfileRequest.Header.Set("Authorization", "Bearer "+adminTokenRecord.Value)
	adminProfileResponse := httptest.NewRecorder()
	adminAPIHandler.ServeHTTP(adminProfileResponse, adminProfileRequest)
	if !strings.Contains(adminProfileResponse.Body.String(), "\"account_email\":\"root@example.com\"") {
		t.Fatalf("unexpected admin api profile security body %q", adminProfileResponse.Body.String())
	}
	adminProtectedRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/server/users", nil)
	adminProtectedRequest.Header.Set("Authorization", "Bearer "+adminTokenRecord.Value)
	adminProtectedResponse := httptest.NewRecorder()
	adminAPIHandler.ServeHTTP(adminProtectedResponse, adminProtectedRequest)
	if !strings.Contains(adminProtectedResponse.Body.String(), "\"surface\":\"server/users\"") {
		t.Fatalf("unexpected admin api users surface body %q", adminProtectedResponse.Body.String())
	}
	adminAsteriskExactRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/server/asterisk/fax", nil)
	adminAsteriskExactRequest.Header.Set("Authorization", "Bearer "+adminTokenRecord.Value)
	adminAsteriskExactResponse := httptest.NewRecorder()
	adminAPIHandler.ServeHTTP(adminAsteriskExactResponse, adminAsteriskExactRequest)
	if !strings.Contains(adminAsteriskExactResponse.Body.String(), "\"key\":\"fax\"") {
		t.Fatalf("unexpected admin api exact asterisk surface body %q", adminAsteriskExactResponse.Body.String())
	}
	adminProtectedMethodRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/server/users", nil)
	adminProtectedMethodRequest.Header.Set("Authorization", "Bearer "+adminTokenRecord.Value)
	adminProtectedMethodResponse := httptest.NewRecorder()
	adminAPIHandler.ServeHTTP(adminProtectedMethodResponse, adminProtectedMethodRequest)
	if adminProtectedMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected admin api protected method status 405, got %d", adminProtectedMethodResponse.Code)
	}
}

type handlerFailingStore struct {
	user       model.User
	admin      model.Admin
	session    model.Session
	token      model.Token
	userError  error
	adminError error
	tokenError error
}

func (authStore handlerFailingStore) SaveAdmin(context.Context, model.Admin) (model.Admin, error) {
	return model.Admin{}, nil
}

func (authStore handlerFailingStore) FindAdminByUsername(context.Context, string) (model.Admin, error) {
	if authStore.adminError != nil {
		return model.Admin{}, authStore.adminError
	}
	return authStore.admin, nil
}

func (authStore handlerFailingStore) FindAdminByID(context.Context, int64) (model.Admin, error) {
	if authStore.adminError != nil {
		return model.Admin{}, authStore.adminError
	}
	return authStore.admin, nil
}

func (authStore handlerFailingStore) SaveUser(context.Context, model.User) (model.User, error) {
	return model.User{}, nil
}

func (authStore handlerFailingStore) FindUserByUsername(context.Context, string) (model.User, error) {
	if authStore.userError != nil {
		return model.User{}, authStore.userError
	}
	return authStore.user, nil
}

func (authStore handlerFailingStore) FindUserByEmail(context.Context, string) (model.User, error) {
	if authStore.userError != nil {
		return model.User{}, authStore.userError
	}
	return authStore.user, nil
}

func (authStore handlerFailingStore) FindUserByID(context.Context, int64) (model.User, error) {
	if authStore.userError != nil {
		return model.User{}, authStore.userError
	}
	return authStore.user, nil
}

func (authStore handlerFailingStore) SaveSession(context.Context, model.Session) (model.Session, error) {
	return authStore.session, nil
}

func (authStore handlerFailingStore) FindSessionByTokenHash(context.Context, model.SessionKind, string) (model.Session, error) {
	return authStore.session, nil
}

func (authStore handlerFailingStore) DeleteSessionByTokenHash(context.Context, model.SessionKind, string) error {
	return nil
}

func (authStore handlerFailingStore) SaveToken(_ context.Context, token model.Token) (model.Token, error) {
	if authStore.token.TokenHash == "" && authStore.token.ID == 0 {
		return token, nil
	}
	return authStore.token, nil
}

func (authStore handlerFailingStore) FindTokenByHash(context.Context, model.TokenOwnerType, string) (model.Token, error) {
	if authStore.tokenError != nil {
		return model.Token{}, authStore.tokenError
	}
	return authStore.token, nil
}

func (authStore handlerFailingStore) DeleteTokenByHash(context.Context, model.TokenOwnerType, string) error {
	return authStore.tokenError
}

type failingOrganizationStore struct {
	organization  model.Organization
	preferences   model.OrganizationPreferences
	member        model.OrganizationMember
	members       []model.OrganizationMember
	orgError      error
	prefError     error
	savePrefError error
	memberError   error
	listError     error
}

func (orgStore failingOrganizationStore) SaveOrganization(context.Context, model.Organization) (model.Organization, error) {
	return orgStore.organization, orgStore.orgError
}

func (orgStore failingOrganizationStore) FindOrganizationBySlug(context.Context, string) (model.Organization, error) {
	if orgStore.orgError != nil {
		return model.Organization{}, orgStore.orgError
	}
	return orgStore.organization, nil
}

func (orgStore failingOrganizationStore) FindOrganizationByID(context.Context, int64) (model.Organization, error) {
	if orgStore.orgError != nil {
		return model.Organization{}, orgStore.orgError
	}
	return orgStore.organization, nil
}

func (orgStore failingOrganizationStore) SaveOrganizationPreferences(context.Context, model.OrganizationPreferences) (model.OrganizationPreferences, error) {
	if orgStore.savePrefError != nil {
		return model.OrganizationPreferences{}, orgStore.savePrefError
	}
	if orgStore.preferences.OrgID == 0 {
		orgStore.preferences = model.DefaultOrganizationPreferences()
		orgStore.preferences.OrgID = orgStore.organization.ID
	}
	return orgStore.preferences, nil
}

func (orgStore failingOrganizationStore) FindOrganizationPreferencesByOrgID(context.Context, int64) (model.OrganizationPreferences, error) {
	if orgStore.prefError != nil {
		return model.OrganizationPreferences{}, orgStore.prefError
	}
	return orgStore.preferences, nil
}

func (orgStore failingOrganizationStore) SaveOrganizationMember(context.Context, model.OrganizationMember) (model.OrganizationMember, error) {
	return orgStore.member, orgStore.memberError
}

func (orgStore failingOrganizationStore) FindOrganizationMember(context.Context, int64) (model.OrganizationMember, error) {
	if orgStore.memberError != nil {
		return model.OrganizationMember{}, orgStore.memberError
	}
	return orgStore.member, nil
}

func (orgStore failingOrganizationStore) FindOrganizationMemberByUserID(context.Context, int64, int64) (model.OrganizationMember, error) {
	if orgStore.memberError != nil {
		return model.OrganizationMember{}, orgStore.memberError
	}
	return orgStore.member, nil
}

func (orgStore failingOrganizationStore) ListOrganizationMembers(context.Context, int64) ([]model.OrganizationMember, error) {
	if orgStore.listError != nil {
		return nil, orgStore.listError
	}
	return orgStore.members, nil
}

func TestOrgHandlerAdditionalBranches(t *testing.T) {
	memoryStore, authService := newTestRuntimeStore(t)
	userCookie := SessionCookieConfig{Name: "user_session", Path: "/", HTTPOnly: true, Secure: "auto", SameSite: http.SameSiteLaxMode}

	orgHandler := NewOrgHandler("/orgs", authService, newEnabledDomainService(memoryStore), memoryStore, userCookie).(OrgHandler)
	orgAPIHandler := NewAPIOrgHandler("/api/v1/orgs", authService, newEnabledDomainService(memoryStore), memoryStore).(APIOrgHandler)

	_, bobSession, bobLoginError := authService.AuthenticateUser(context.Background(), "bob", "correct horse battery staple", "127.0.0.1", "curl/8.0")
	if bobLoginError != nil {
		t.Fatalf("authenticate bob session: %v", bobLoginError)
	}

	methodRequest := httptest.NewRequest(http.MethodPost, "/orgs/acme", nil)
	methodResponse := httptest.NewRecorder()
	orgHandler.ServeHTTP(methodResponse, methodRequest)
	if methodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected org handler method rejection, got %d", methodResponse.Code)
	}

	rootRequest := httptest.NewRequest(http.MethodGet, "/orgs", nil)
	rootResponse := httptest.NewRecorder()
	orgHandler.ServeHTTP(rootResponse, rootRequest)
	if rootResponse.Code != http.StatusNotFound {
		t.Fatalf("expected org root 404, got %d", rootResponse.Code)
	}

	forbiddenSettingsRequest := httptest.NewRequest(http.MethodGet, "/orgs/acme/settings", nil)
	forbiddenSettingsRequest.AddCookie(userCookie.Build(forbiddenSettingsRequest, bobSession.Token, bobSession.Session.ExpiresAt))
	forbiddenSettingsResponse := httptest.NewRecorder()
	orgHandler.ServeHTTP(forbiddenSettingsResponse, forbiddenSettingsRequest)
	if forbiddenSettingsResponse.Code != http.StatusForbidden {
		t.Fatalf("expected member settings forbidden, got %d", forbiddenSettingsResponse.Code)
	}

	unauthorizedSettingsRequest := httptest.NewRequest(http.MethodGet, "/orgs/acme/settings", nil)
	unauthorizedSettingsResponse := httptest.NewRecorder()
	orgHandler.ServeHTTP(unauthorizedSettingsResponse, unauthorizedSettingsRequest)
	if unauthorizedSettingsResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated public org settings 401, got %d", unauthorizedSettingsResponse.Code)
	}

	textMembersRequest := httptest.NewRequest(http.MethodGet, "/orgs/acme/members", nil)
	textMembersResponse := httptest.NewRecorder()
	orgHandler.ServeHTTP(textMembersResponse, textMembersRequest)
	if textMembersResponse.Code != http.StatusOK || !strings.Contains(textMembersResponse.Body.String(), "Organization Members: acme") {
		t.Fatalf("unexpected text org members response %d %q", textMembersResponse.Code, textMembersResponse.Body.String())
	}

	unknownOrgRequest := httptest.NewRequest(http.MethodGet, "/orgs/missing", nil)
	unknownOrgResponse := httptest.NewRecorder()
	orgHandler.ServeHTTP(unknownOrgResponse, unknownOrgRequest)
	if unknownOrgResponse.Code != http.StatusNotFound {
		t.Fatalf("expected missing org 404, got %d", unknownOrgResponse.Code)
	}
	unknownOrgPathRequest := httptest.NewRequest(http.MethodGet, "/orgs/acme/unknown", nil)
	unknownOrgPathResponse := httptest.NewRecorder()
	orgHandler.ServeHTTP(unknownOrgPathResponse, unknownOrgPathRequest)
	if unknownOrgPathResponse.Code != http.StatusNotFound {
		t.Fatalf("expected unknown org path 404, got %d", unknownOrgPathResponse.Code)
	}

	jsonOrgRequest := httptest.NewRequest(http.MethodGet, "/orgs/acme", nil)
	jsonOrgRequest.Header.Set("Accept", "application/json")
	jsonOrgResponse := httptest.NewRecorder()
	orgHandler.writeOrgProfile(jsonOrgResponse, jsonOrgRequest, orgAccessContext{
		organization: model.Organization{Slug: "acme", Name: "Acme", Visibility: model.OrganizationVisibilityPublic},
		preferences:  model.DefaultOrganizationPreferences(),
	})
	if !strings.Contains(jsonOrgResponse.Body.String(), "\"slug\":\"acme\"") {
		t.Fatalf("unexpected direct org profile response %q", jsonOrgResponse.Body.String())
	}

	failingMembersRequest := httptest.NewRequest(http.MethodGet, "/orgs/acme/members", nil)
	failingMembersResponse := httptest.NewRecorder()
	NewOrgHandler("/orgs", authService, testDomainService(), failingOrganizationStore{
		organization: model.Organization{ID: 1, Slug: "acme", Visibility: model.OrganizationVisibilityPublic},
		preferences:  model.DefaultOrganizationPreferences(),
		listError:    errors.New("list failed"),
	}, userCookie).ServeHTTP(failingMembersResponse, failingMembersRequest)
	if failingMembersResponse.Code != http.StatusNotFound {
		t.Fatalf("expected failing members response 404, got %d", failingMembersResponse.Code)
	}

	apiMethodRequest := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/acme", nil)
	apiMethodResponse := httptest.NewRecorder()
	orgAPIHandler.ServeHTTP(apiMethodResponse, apiMethodRequest)
	if apiMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected api org method rejection, got %d", apiMethodResponse.Code)
	}

	apiRootRequest := httptest.NewRequest(http.MethodGet, "/api/v1/orgs", nil)
	apiRootResponse := httptest.NewRecorder()
	orgAPIHandler.ServeHTTP(apiRootResponse, apiRootRequest)
	if apiRootResponse.Code != http.StatusNotFound {
		t.Fatalf("expected api org root 404, got %d", apiRootResponse.Code)
	}

	publicWithBadTokenRequest := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/acme", nil)
	publicWithBadTokenRequest.Header.Set("Authorization", "Bearer invalid")
	publicWithBadTokenResponse := httptest.NewRecorder()
	orgAPIHandler.ServeHTTP(publicWithBadTokenResponse, publicWithBadTokenRequest)
	if publicWithBadTokenResponse.Code != http.StatusOK {
		t.Fatalf("expected public org api to ignore invalid auth, got %d", publicWithBadTokenResponse.Code)
	}

	hiddenMemberRequest := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/acme/members/2", nil)
	hiddenMemberResponse := httptest.NewRecorder()
	orgAPIHandler.ServeHTTP(hiddenMemberResponse, hiddenMemberRequest)
	if hiddenMemberResponse.Code != http.StatusNotFound {
		t.Fatalf("expected hidden member detail 404, got %d", hiddenMemberResponse.Code)
	}
	unauthorizedPublicSettingsRequest := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/acme/settings", nil)
	unauthorizedPublicSettingsResponse := httptest.NewRecorder()
	orgAPIHandler.ServeHTTP(unauthorizedPublicSettingsResponse, unauthorizedPublicSettingsRequest)
	if unauthorizedPublicSettingsResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated public org settings api 401, got %d", unauthorizedPublicSettingsResponse.Code)
	}
	unknownAPIPathRequest := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/acme/unknown", nil)
	unknownAPIPathResponse := httptest.NewRecorder()
	orgAPIHandler.ServeHTTP(unknownAPIPathResponse, unknownAPIPathRequest)
	if unknownAPIPathResponse.Code != http.StatusNotFound {
		t.Fatalf("expected unknown api org path 404, got %d", unknownAPIPathResponse.Code)
	}

	invalidMemberIDRequest := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/acme/members/not-a-number", nil)
	invalidMemberIDResponse := httptest.NewRecorder()
	orgAPIHandler.ServeHTTP(invalidMemberIDResponse, invalidMemberIDRequest)
	if invalidMemberIDResponse.Code != http.StatusNotFound {
		t.Fatalf("expected invalid member id 404, got %d", invalidMemberIDResponse.Code)
	}

	wrongOrgTokenValue := "org_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if _, saveTokenError := memoryStore.SaveToken(context.Background(), model.Token{
		OwnerType:   model.TokenOwnerOrg,
		OwnerID:     999,
		Name:        "default",
		TokenHash:   service.HashToken(wrongOrgTokenValue),
		TokenPrefix: "org_bbbb",
		Scope:       model.TokenScopeGlobal,
		ExpiresAt:   time.Unix(2_000_000_000, 0),
	}); saveTokenError != nil {
		t.Fatalf("save wrong org token: %v", saveTokenError)
	}
	wrongOrgTokenRequest := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/secret/settings", nil)
	wrongOrgTokenRequest.Header.Set("Authorization", "Bearer "+wrongOrgTokenValue)
	wrongOrgTokenResponse := httptest.NewRecorder()
	orgAPIHandler.ServeHTTP(wrongOrgTokenResponse, wrongOrgTokenRequest)
	if wrongOrgTokenResponse.Code != http.StatusNotFound {
		t.Fatalf("expected wrong org token private org response 404, got %d", wrongOrgTokenResponse.Code)
	}

	_, bobToken, bobTokenError := authService.AuthenticateUserAPI(context.Background(), "bob", "correct horse battery staple")
	if bobTokenError != nil {
		t.Fatalf("authenticate bob token: %v", bobTokenError)
	}
	bobPublicOrgRequest := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/acme/settings", nil)
	bobPublicOrgRequest.Header.Set("Authorization", "Bearer "+bobToken.Value)
	bobPublicOrgResponse := httptest.NewRecorder()
	orgAPIHandler.ServeHTTP(bobPublicOrgResponse, bobPublicOrgRequest)
	if bobPublicOrgResponse.Code != http.StatusForbidden {
		t.Fatalf("expected non-admin public org settings response 403, got %d", bobPublicOrgResponse.Code)
	}
	bobPrivateOrgRequest := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/secret/settings", nil)
	bobPrivateOrgRequest.Header.Set("Authorization", "Bearer "+bobToken.Value)
	bobPrivateOrgResponse := httptest.NewRecorder()
	orgAPIHandler.ServeHTTP(bobPrivateOrgResponse, bobPrivateOrgRequest)
	if bobPrivateOrgResponse.Code != http.StatusNotFound {
		t.Fatalf("expected non-member private org api response 404, got %d", bobPrivateOrgResponse.Code)
	}

	if !writeOrgAccessError(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/orgs/acme", nil), orgAccessAllowed) {
		t.Fatalf("expected allowed access to return true")
	}
	for _, accessState := range []orgAccessState{orgAccessUnauthorized, orgAccessForbidden, orgAccessNotFound} {
		response := httptest.NewRecorder()
		if writeOrgAccessError(response, httptest.NewRequest(http.MethodGet, "/orgs/acme", nil), accessState) {
			t.Fatalf("expected non-allowed access state %v to return false", accessState)
		}
	}

	if _, _, found := orgRouteParts("/orgs", "/orgs"); found {
		t.Fatalf("expected empty org route parts to be rejected")
	}
	if slug, rest, found := orgRouteParts("/orgs", "/orgs/acme/members"); !found || slug != "acme" || rest != "members" {
		t.Fatalf("unexpected org route parts %q %q %t", slug, rest, found)
	}

	if profile := orgProfileResponse(orgAccessContext{
		organization: model.Organization{Slug: "acme", Name: "Acme", Visibility: model.OrganizationVisibilityPublic},
		preferences:  model.DefaultOrganizationPreferences(),
	}); profile["name"] != "Acme" {
		t.Fatalf("unexpected org profile helper response %+v", profile)
	}

	defaultOrgStore := failingOrganizationStore{
		organization: model.Organization{ID: 11, Slug: "defaults", Visibility: model.OrganizationVisibilityPublic},
		prefError:    store.ErrNotFound,
	}
	_, defaultPreferences, loadError := loadOrganizationContext(context.Background(), defaultOrgStore, "defaults")
	if loadError != nil || defaultPreferences.OrgID != 11 {
		t.Fatalf("expected default org preferences, got %v / %+v", loadError, defaultPreferences)
	}
	if _, _, loadError = loadOrganizationContext(context.Background(), failingOrganizationStore{
		organization: model.Organization{ID: 12, Slug: "broken"},
		prefError:    errors.New("pref failed"),
	}, "broken"); loadError == nil {
		t.Fatalf("expected org preference load failure")
	}
	if _, _, loadError = loadOrganizationContext(context.Background(), failingOrganizationStore{
		organization:  model.Organization{ID: 13, Slug: "broken-save"},
		prefError:     store.ErrNotFound,
		savePrefError: errors.New("save failed"),
	}, "broken-save"); loadError == nil {
		t.Fatalf("expected org preference save failure")
	}

	if _, membersError := buildOrganizationMemberResponses(context.Background(), failingOrganizationStore{
		organization: model.Organization{ID: 1, Slug: "acme"},
		listError:    errors.New("list failed"),
	}, authService, orgAccessContext{organization: model.Organization{ID: 1, Slug: "acme"}}, false); membersError == nil {
		t.Fatalf("expected member list failure")
	}
	memberProfiles, membersError := buildOrganizationMemberResponses(context.Background(), failingOrganizationStore{
		organization: model.Organization{ID: 1, Slug: "acme"},
		members:      []model.OrganizationMember{{ID: 1, OrgID: 1, UserID: 1, Role: model.OrganizationRoleOwner}},
	}, service.NewAuthService(handlerFailingStore{userError: errors.New("missing user")}, service.DefaultSessionConfig()), orgAccessContext{organization: model.Organization{ID: 1, Slug: "acme"}}, true)
	if membersError != nil || len(memberProfiles) != 0 {
		t.Fatalf("expected skipped missing user member profiles, got %v / %+v", membersError, memberProfiles)
	}

	invalidSessionRequest := httptest.NewRequest(http.MethodGet, "/orgs/secret", nil)
	invalidSessionRequest.AddCookie(&http.Cookie{Name: "user_session", Value: "missing"})
	if _, found := orgHandler.resolveWebMember(invalidSessionRequest, 2); found {
		t.Fatalf("expected missing web member resolution")
	}
	if _, accessState := orgAPIHandler.resolveAPIOrgAccess(httptest.NewRequest(http.MethodGet, "/api/v1/orgs/missing", nil), "missing"); accessState != orgAccessNotFound {
		t.Fatalf("expected missing api org access state, got %v", accessState)
	}
	nonMemberSessionRequest := httptest.NewRequest(http.MethodGet, "/orgs/secret", nil)
	nonMemberSessionRequest.AddCookie(userCookie.Build(nonMemberSessionRequest, bobSession.Token, bobSession.Session.ExpiresAt))
	if _, found := orgHandler.resolveWebMember(nonMemberSessionRequest, 2); found {
		t.Fatalf("expected non-member web member resolution to fail")
	}
	if _, _, accessState := orgAPIHandler.resolveAPIMember(httptest.NewRequest(http.MethodGet, "/api/v1/orgs/secret", nil), 2); accessState != orgAccessUnauthorized {
		t.Fatalf("expected missing api member auth state, got %v", accessState)
	}
	if _, _, accessState := orgAPIHandler.resolveAPIMember(func() *http.Request {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/secret", nil)
		request.Header.Set("Authorization", "Bearer invalid")
		return request
	}(), 2); accessState != orgAccessUnauthorized {
		t.Fatalf("expected invalid org token auth state, got %v", accessState)
	}
	if _, _, accessState := orgAPIHandler.resolveAPIMember(func() *http.Request {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/secret", nil)
		request.Header.Set("Authorization", "Bearer org_cccccccccccccccccccccccccccccccc")
		return request
	}(), 2); accessState != orgAccessUnauthorized {
		t.Fatalf("expected missing stored org token auth state, got %v", accessState)
	}
	failingAPIMembersResponse := httptest.NewRecorder()
	NewAPIOrgHandler("/api/v1/orgs", authService, testDomainService(), failingOrganizationStore{
		organization: model.Organization{ID: 1, Slug: "acme", Visibility: model.OrganizationVisibilityPublic},
		preferences:  model.DefaultOrganizationPreferences(),
		listError:    errors.New("list failed"),
	}).ServeHTTP(failingAPIMembersResponse, httptest.NewRequest(http.MethodGet, "/api/v1/orgs/acme/members", nil))
	if failingAPIMembersResponse.Code != http.StatusNotFound {
		t.Fatalf("expected failing api members response 404, got %d", failingAPIMembersResponse.Code)
	}
	userLookupFailureResponse := httptest.NewRecorder()
	NewAPIOrgHandler("/api/v1/orgs", service.NewAuthService(handlerFailingStore{userError: errors.New("missing user")}, service.DefaultSessionConfig()), testDomainService(), failingOrganizationStore{
		organization: model.Organization{ID: 1, Slug: "acme", Visibility: model.OrganizationVisibilityPublic},
		preferences:  model.DefaultOrganizationPreferences(),
		member:       model.OrganizationMember{ID: 1, OrgID: 1, UserID: 1, Role: model.OrganizationRoleOwner},
	}).ServeHTTP(userLookupFailureResponse, httptest.NewRequest(http.MethodGet, "/api/v1/orgs/acme/members/1", nil))
	if userLookupFailureResponse.Code != http.StatusNotFound {
		t.Fatalf("expected failing api member detail response 404, got %d", userLookupFailureResponse.Code)
	}
	memberLookupFailureResponse := httptest.NewRecorder()
	NewAPIOrgHandler("/api/v1/orgs", authService, testDomainService(), failingOrganizationStore{
		organization: model.Organization{ID: 1, Slug: "acme", Visibility: model.OrganizationVisibilityPublic},
		preferences:  model.DefaultOrganizationPreferences(),
		memberError:  errors.New("missing member"),
	}).ServeHTTP(memberLookupFailureResponse, httptest.NewRequest(http.MethodGet, "/api/v1/orgs/acme/members/1", nil))
	if memberLookupFailureResponse.Code != http.StatusNotFound {
		t.Fatalf("expected failing api member lookup response 404, got %d", memberLookupFailureResponse.Code)
	}
	if _, _, found := orgRouteParts("/orgs", "/orgs/ /members"); found {
		t.Fatalf("expected blank org slug route parts to be rejected")
	}
}

func TestAuthHandlerAdditionalBranches(t *testing.T) {
	authService := newTestAuthService(t)
	userCookie := SessionCookieConfig{Name: "user_session", Path: "/", HTTPOnly: true, Secure: "auto", SameSite: http.SameSiteLaxMode}
	authHandler := NewAuthHandler("/auth", authService, userCookie, model.RegistrationModePrivate)

	authOverviewJSONRequest := httptest.NewRequest(http.MethodGet, "/auth", nil)
	authOverviewJSONRequest.Header.Set("Accept", "application/json")
	authOverviewJSONResponse := httptest.NewRecorder()
	authHandler.ServeHTTP(authOverviewJSONResponse, authOverviewJSONRequest)
	if !strings.Contains(authOverviewJSONResponse.Body.String(), "\"scope\":\"auth\"") {
		t.Fatalf("unexpected auth overview json %q", authOverviewJSONResponse.Body.String())
	}

	loginInfoRequest := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	loginInfoResponse := httptest.NewRecorder()
	authHandler.ServeHTTP(loginInfoResponse, loginInfoRequest)
	if !strings.Contains(loginInfoResponse.Body.String(), "POST /auth/login") {
		t.Fatalf("unexpected login info body %q", loginInfoResponse.Body.String())
	}
	loginInfoJSONRequest := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	loginInfoJSONRequest.Header.Set("Accept", "application/json")
	loginInfoJSONResponse := httptest.NewRecorder()
	authHandler.ServeHTTP(loginInfoJSONResponse, loginInfoJSONRequest)
	if !strings.Contains(loginInfoJSONResponse.Body.String(), "\"identity\":\"username_or_email\"") {
		t.Fatalf("unexpected login info json %q", loginInfoJSONResponse.Body.String())
	}

	invalidLoginForm := url.Values{"identifier": {"alice"}, "password": {"wrong password"}}
	invalidLoginRequest := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(invalidLoginForm.Encode()))
	invalidLoginRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	invalidLoginResponse := httptest.NewRecorder()
	authHandler.ServeHTTP(invalidLoginResponse, invalidLoginRequest)
	if invalidLoginResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected invalid login status 401, got %d", invalidLoginResponse.Code)
	}

	loginMethodRequest := httptest.NewRequest(http.MethodPut, "/auth/login", nil)
	loginMethodResponse := httptest.NewRecorder()
	authHandler.ServeHTTP(loginMethodResponse, loginMethodRequest)
	if loginMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected login method status 405, got %d", loginMethodResponse.Code)
	}

	publicAuthHandler := NewAuthHandler("/auth", authService, userCookie, model.RegistrationModePublic)
	registerInfoRequest := httptest.NewRequest(http.MethodGet, "/auth/register", nil)
	registerInfoResponse := httptest.NewRecorder()
	publicAuthHandler.ServeHTTP(registerInfoResponse, registerInfoRequest)
	if !strings.Contains(registerInfoResponse.Body.String(), "\"registration_mode\":\"public\"") {
		t.Fatalf("unexpected register info body %q", registerInfoResponse.Body.String())
	}
	publicRegisterRequest := httptest.NewRequest(http.MethodPost, "/auth/register", nil)
	publicRegisterResponse := httptest.NewRecorder()
	publicAuthHandler.ServeHTTP(publicRegisterResponse, publicRegisterRequest)
	if publicRegisterResponse.Code != http.StatusNotImplemented {
		t.Fatalf("expected public register status 501, got %d", publicRegisterResponse.Code)
	}
	registerMethodRequest := httptest.NewRequest(http.MethodPut, "/auth/register", nil)
	registerMethodResponse := httptest.NewRecorder()
	authHandler.ServeHTTP(registerMethodResponse, registerMethodRequest)
	if registerMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected register method status 405, got %d", registerMethodResponse.Code)
	}

	refreshUnauthorizedRequest := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	refreshUnauthorizedResponse := httptest.NewRecorder()
	authHandler.ServeHTTP(refreshUnauthorizedResponse, refreshUnauthorizedRequest)
	if refreshUnauthorizedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected refresh without cookie to fail, got %d", refreshUnauthorizedResponse.Code)
	}
	refreshMethodRequest := httptest.NewRequest(http.MethodGet, "/auth/refresh", nil)
	refreshMethodResponse := httptest.NewRecorder()
	authHandler.ServeHTTP(refreshMethodResponse, refreshMethodRequest)
	if refreshMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected refresh method status 405, got %d", refreshMethodResponse.Code)
	}

	logoutInvalidRequest := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	logoutInvalidRequest.AddCookie(&http.Cookie{Name: "user_session", Value: "invalid"})
	logoutInvalidResponse := httptest.NewRecorder()
	authHandler.ServeHTTP(logoutInvalidResponse, logoutInvalidRequest)
	if logoutInvalidResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected invalid logout status 401, got %d", logoutInvalidResponse.Code)
	}
	logoutMethodRequest := httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	logoutMethodResponse := httptest.NewRecorder()
	authHandler.ServeHTTP(logoutMethodResponse, logoutMethodRequest)
	if logoutMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected logout method status 405, got %d", logoutMethodResponse.Code)
	}

	authUnknownRequest := httptest.NewRequest(http.MethodGet, "/auth/missing", nil)
	authUnknownResponse := httptest.NewRecorder()
	authHandler.ServeHTTP(authUnknownResponse, authUnknownRequest)
	if authUnknownResponse.Code != http.StatusNotFound {
		t.Fatalf("expected unknown auth route status 404, got %d", authUnknownResponse.Code)
	}
}

func TestAdminAndUserHandlerAdditionalBranches(t *testing.T) {
	authService := newTestAuthService(t)
	userCookie := SessionCookieConfig{Name: "user_session", Path: "/", HTTPOnly: true, Secure: "auto", SameSite: http.SameSiteLaxMode}
	adminCookie := SessionCookieConfig{Name: "admin_session", Path: "/admin", HTTPOnly: true, Secure: "auto", SameSite: http.SameSiteLaxMode}

	adminHandler := NewAdminHandler("/admin", authService, testDomainService(), testAsteriskService(), testPBXService(), adminCookie)
	adminLoginInvalidRequest := httptest.NewRequest(http.MethodPost, "/admin", strings.NewReader("{"))
	adminLoginInvalidRequest.Header.Set("Content-Type", "application/json")
	adminLoginInvalidResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(adminLoginInvalidResponse, adminLoginInvalidRequest)
	if adminLoginInvalidResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid admin login status 400, got %d", adminLoginInvalidResponse.Code)
	}

	adminRootMethodRequest := httptest.NewRequest(http.MethodPut, "/admin", nil)
	adminRootMethodResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(adminRootMethodResponse, adminRootMethodRequest)
	if adminRootMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected admin root method status 405, got %d", adminRootMethodResponse.Code)
	}

	adminLoginForm := url.Values{"username": {"root-admin"}, "password": {"correct horse battery staple"}}
	adminLoginRequest := httptest.NewRequest(http.MethodPost, "/admin", strings.NewReader(adminLoginForm.Encode()))
	adminLoginRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	adminLoginResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(adminLoginResponse, adminLoginRequest)
	adminSessionCookie := adminLoginResponse.Result().Cookies()[0]

	adminProfileJSONRequest := httptest.NewRequest(http.MethodGet, "/admin/profile", nil)
	adminProfileJSONRequest.AddCookie(adminSessionCookie)
	adminProfileJSONRequest.Header.Set("Accept", "application/json")
	adminProfileJSONResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(adminProfileJSONResponse, adminProfileJSONRequest)
	if !strings.Contains(adminProfileJSONResponse.Body.String(), "\"role\":\"admin\"") {
		t.Fatalf("unexpected admin profile json %q", adminProfileJSONResponse.Body.String())
	}

	adminSurfaceJSONRequest := httptest.NewRequest(http.MethodGet, "/admin/server/settings", nil)
	adminSurfaceJSONRequest.AddCookie(adminSessionCookie)
	adminSurfaceJSONRequest.Header.Set("Accept", "application/json")
	adminSurfaceJSONResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(adminSurfaceJSONResponse, adminSurfaceJSONRequest)
	if !strings.Contains(adminSurfaceJSONResponse.Body.String(), "\"surface\":\"server/settings\"") {
		t.Fatalf("unexpected admin surface json %q", adminSurfaceJSONResponse.Body.String())
	}

	adminProtectedSurfaceMethodRequest := httptest.NewRequest(http.MethodPost, "/admin/server/settings", nil)
	adminProtectedSurfaceMethodRequest.AddCookie(adminSessionCookie)
	adminProtectedSurfaceMethodResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(adminProtectedSurfaceMethodResponse, adminProtectedSurfaceMethodRequest)
	if adminProtectedSurfaceMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected admin surface post status 405, got %d", adminProtectedSurfaceMethodResponse.Code)
	}

	adminLookupFailureService := service.NewAuthService(handlerFailingStore{
		adminError: errors.New("admin missing"),
		session:    model.Session{ID: "s1", Kind: model.SessionKindAdmin, SubjectID: 1, TokenHash: service.HashToken("token")},
	}, service.DefaultSessionConfig())
	adminLookupFailureHandler := NewAdminHandler("/admin", adminLookupFailureService, testDomainService(), testAsteriskService(), testPBXService(), adminCookie)
	adminLookupFailureRequest := httptest.NewRequest(http.MethodGet, "/admin/profile", nil)
	adminLookupFailureRequest.AddCookie(&http.Cookie{Name: "admin_session", Value: "token"})
	adminLookupFailureResponse := httptest.NewRecorder()
	adminLookupFailureHandler.ServeHTTP(adminLookupFailureResponse, adminLookupFailureRequest)
	if adminLookupFailureResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected admin lookup failure status 401, got %d", adminLookupFailureResponse.Code)
	}

	userAuthService := newTestAuthService(t)
	userHandler := NewUserHandler("/users", userAuthService, testDomainService(), userCookie)
	loginForm := url.Values{"identifier": {"alice"}, "password": {"correct horse battery staple"}}
	loginRequest := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(loginForm.Encode()))
	loginRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginResponse := httptest.NewRecorder()
	NewAuthHandler("/auth", userAuthService, userCookie, model.RegistrationModePrivate).ServeHTTP(loginResponse, loginRequest)
	userSessionCookie := loginResponse.Result().Cookies()[0]

	userProfileTextRequest := httptest.NewRequest(http.MethodGet, "/users/profile", nil)
	userProfileTextRequest.AddCookie(userSessionCookie)
	userProfileTextResponse := httptest.NewRecorder()
	userHandler.ServeHTTP(userProfileTextResponse, userProfileTextRequest)
	if !strings.Contains(userProfileTextResponse.Body.String(), "User: alice") {
		t.Fatalf("unexpected user profile text %q", userProfileTextResponse.Body.String())
	}

	userMethodRequest := httptest.NewRequest(http.MethodPost, "/users/profile", nil)
	userMethodResponse := httptest.NewRecorder()
	userHandler.ServeHTTP(userMethodResponse, userMethodRequest)
	if userMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected user profile post status 405, got %d", userMethodResponse.Code)
	}

	userLookupFailureService := service.NewAuthService(handlerFailingStore{
		userError: errors.New("user missing"),
		session:   model.Session{ID: "s2", Kind: model.SessionKindUser, SubjectID: 1, TokenHash: service.HashToken("token")},
	}, service.DefaultSessionConfig())
	userLookupFailureHandler := NewUserHandler("/users", userLookupFailureService, testDomainService(), userCookie)
	userLookupFailureRequest := httptest.NewRequest(http.MethodGet, "/users/profile", nil)
	userLookupFailureRequest.AddCookie(&http.Cookie{Name: "user_session", Value: "token"})
	userLookupFailureResponse := httptest.NewRecorder()
	userLookupFailureHandler.ServeHTTP(userLookupFailureResponse, userLookupFailureRequest)
	if userLookupFailureResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected user lookup failure status 401, got %d", userLookupFailureResponse.Code)
	}
}

func TestDirectHandlerMethodsAndHelpers(t *testing.T) {
	authService := newTestAuthService(t)
	userCookie := SessionCookieConfig{Name: "user_session", Path: "/", HTTPOnly: true, Secure: "auto", SameSite: http.SameSiteLaxMode}
	adminCookie := SessionCookieConfig{Name: "admin_session", Path: "/admin", HTTPOnly: true, Secure: "auto", SameSite: http.SameSiteLaxMode}

	authHandlerValue := NewAuthHandler("/auth", authService, userCookie, model.RegistrationModePrivate).(AuthHandler)
	postOverviewRequest := httptest.NewRequest(http.MethodPost, "/auth", nil)
	postOverviewResponse := httptest.NewRecorder()
	authHandlerValue.writeOverview(postOverviewResponse, postOverviewRequest)
	if postOverviewResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected auth overview post status 405, got %d", postOverviewResponse.Code)
	}

	loginJSONRequest := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"identifier":"alice","password":"correct horse battery staple"}`))
	loginJSONRequest.Header.Set("Content-Type", "application/json")
	loginJSONResponse := httptest.NewRecorder()
	authHandlerValue.handleLogin(loginJSONResponse, loginJSONRequest)
	if loginJSONResponse.Code != http.StatusOK {
		t.Fatalf("expected json login status 200, got %d", loginJSONResponse.Code)
	}
	refreshInvalidRequest := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	refreshInvalidRequest.AddCookie(&http.Cookie{Name: "user_session", Value: "invalid"})
	refreshInvalidResponse := httptest.NewRecorder()
	authHandlerValue.handleRefresh(refreshInvalidResponse, refreshInvalidRequest)
	if len(refreshInvalidResponse.Result().Cookies()) == 0 || refreshInvalidResponse.Result().Cookies()[0].MaxAge != -1 {
		t.Fatalf("expected invalid refresh cookie clear")
	}

	failingFormRequest := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	failingFormRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	failingFormRequest.Body = errReadCloser{}
	if _, parseError := readAuthLoginRequest(failingFormRequest); parseError == nil {
		t.Fatalf("expected form parse failure")
	}

	adminHandlerValue := NewAdminHandler("/admin", authService, testDomainService(), testAsteriskService(), testPBXService(), adminCookie).(AdminHandler)
	adminWrongLoginRequest := httptest.NewRequest(http.MethodPost, "/admin", strings.NewReader(url.Values{"username": {"root-admin"}, "password": {"wrong"}}.Encode()))
	adminWrongLoginRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	adminWrongLoginResponse := httptest.NewRecorder()
	adminHandlerValue.handleRoot(adminWrongLoginResponse, adminWrongLoginRequest)
	if adminWrongLoginResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected wrong admin login status 401, got %d", adminWrongLoginResponse.Code)
	}

	asteriskLoginRequest := httptest.NewRequest(http.MethodPost, "/admin", strings.NewReader(url.Values{"username": {"root-admin"}, "password": {"correct horse battery staple"}}.Encode()))
	asteriskLoginRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	asteriskLoginResponse := httptest.NewRecorder()
	adminHandlerValue.handleRoot(asteriskLoginResponse, asteriskLoginRequest)
	adminSessionCookie := asteriskLoginResponse.Result().Cookies()[0]

	asteriskHandlerValue := NewAdminHandler("/admin/server/asterisk", authService, testDomainService(), testAsteriskService(), testPBXService(), adminCookie).(AdminHandler)
	asteriskRequest := httptest.NewRequest(http.MethodGet, "/admin/server/asterisk/fax", nil)
	asteriskRequest.AddCookie(adminSessionCookie)
	asteriskResponse := httptest.NewRecorder()
	asteriskHandlerValue.ServeHTTP(asteriskResponse, asteriskRequest)
	if !strings.Contains(asteriskResponse.Body.String(), "Asterisk surface: Fax") {
		t.Fatalf("unexpected asterisk direct surface %q", asteriskResponse.Body.String())
	}

	adminResolveFailureHandler := NewAdminHandler("/admin", service.NewAuthService(handlerFailingStore{
		adminError: errors.New("admin missing"),
		session:    model.Session{ID: "s1", Kind: model.SessionKindAdmin, SubjectID: 1, TokenHash: service.HashToken("token"), ExpiresAt: time.Unix(2_000_000_000, 0)},
	}, service.DefaultSessionConfig()), testDomainService(), testAsteriskService(), testPBXService(), adminCookie).(AdminHandler)
	if _, _, ok := adminResolveFailureHandler.resolveAdmin(func() *http.Request {
		request := httptest.NewRequest(http.MethodGet, "/admin/profile", nil)
		request.AddCookie(&http.Cookie{Name: "admin_session", Value: "token"})
		return request
	}()); ok {
		t.Fatalf("expected admin resolve failure")
	}
	adminProtectedUnauthorizedRequest := httptest.NewRequest(http.MethodGet, "/admin/server/settings", nil)
	adminProtectedUnauthorizedResponse := httptest.NewRecorder()
	adminResolveFailureHandler.handleProtectedSurface(adminProtectedUnauthorizedResponse, adminProtectedUnauthorizedRequest, "server/settings")
	if adminProtectedUnauthorizedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized protected admin surface, got %d", adminProtectedUnauthorizedResponse.Code)
	}

	userHandlerValue := NewUserHandler("/users", service.NewAuthService(handlerFailingStore{
		userError: errors.New("user missing"),
		session:   model.Session{ID: "s2", Kind: model.SessionKindUser, SubjectID: 1, TokenHash: service.HashToken("token"), ExpiresAt: time.Unix(2_000_000_000, 0)},
	}, service.DefaultSessionConfig()), testDomainService(), userCookie).(UserHandler)
	userLookupFailureRequest := httptest.NewRequest(http.MethodGet, "/users/profile", nil)
	userLookupFailureRequest.AddCookie(&http.Cookie{Name: "user_session", Value: "token"})
	userLookupFailureResponse := httptest.NewRecorder()
	userHandlerValue.ServeHTTP(userLookupFailureResponse, userLookupFailureRequest)
	if userLookupFailureResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected direct user lookup failure status 401, got %d", userLookupFailureResponse.Code)
	}

	apiAuthHandlerValue := NewAPIAuthHandler("/api/v1/auth", authService, model.RegistrationModePrivate).(APIAuthHandler)
	apiRefreshUnauthorizedRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	apiRefreshUnauthorizedResponse := httptest.NewRecorder()
	apiAuthHandlerValue.handleRefresh(apiRefreshUnauthorizedResponse, apiRefreshUnauthorizedRequest)
	if apiRefreshUnauthorizedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected api refresh auth failure, got %d", apiRefreshUnauthorizedResponse.Code)
	}
	apiNotImplementedRequest := httptest.NewRequest(http.MethodPut, "/api/v1/auth/2fa", nil)
	apiNotImplementedResponse := httptest.NewRecorder()
	writeAPINotImplemented(apiNotImplementedResponse, apiNotImplementedRequest, "/api/v1/auth/2fa")
	if apiNotImplementedResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected api not implemented method failure, got %d", apiNotImplementedResponse.Code)
	}
	if token, tokenError := readBearerToken(func() *http.Request {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/users/profile", nil)
		request.Header.Set("Authorization", "Bearer usr_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		return request
	}()); tokenError != nil || token != "usr_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("expected bearer token read, got %v / %q", tokenError, token)
	}
	if _, tokenError := readBearerToken(func() *http.Request {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/users/profile", nil)
		request.Header.Set("Authorization", "Basic abc123")
		return request
	}()); tokenError == nil {
		t.Fatalf("expected invalid bearer token format error")
	}
	if _, tokenError := readBearerToken(httptest.NewRequest(http.MethodGet, "/api/v1/users/profile", nil)); tokenError == nil {
		t.Fatalf("expected missing bearer token error")
	}
	if formatOptionalTime(time.Time{}) != "" || formatOptionalTime(time.Unix(0, 0)) == "" {
		t.Fatalf("unexpected optional time formatting")
	}

	apiAdminResolveFailureHandler := NewAPIAdminHandler("/api/v1/admin", service.NewAuthService(handlerFailingStore{
		adminError: errors.New("admin missing"),
		token: model.Token{
			ID:          1,
			OwnerType:   model.TokenOwnerAdmin,
			OwnerID:     1,
			TokenHash:   service.HashToken("adm_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
			TokenPrefix: "adm_aaaa",
			ExpiresAt:   time.Unix(2_000_000_000, 0),
		},
	}, service.DefaultSessionConfig()), testDomainService(), testAsteriskService(), testPBXService()).(APIAdminHandler)
	if _, _, ok := apiAdminResolveFailureHandler.resolveAdmin(func() *http.Request {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/profile", nil)
		request.Header.Set("Authorization", "Bearer adm_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		return request
	}()); ok {
		t.Fatalf("expected api admin resolve failure")
	}
	apiAdminTokenResolveFailureHandler := NewAPIAdminHandler("/api/v1/admin", service.NewAuthService(handlerFailingStore{
		tokenError: errors.New("token missing"),
	}, service.DefaultSessionConfig()), testDomainService(), testAsteriskService(), testPBXService()).(APIAdminHandler)
	if _, _, ok := apiAdminTokenResolveFailureHandler.resolveAdmin(func() *http.Request {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/profile", nil)
		request.Header.Set("Authorization", "Bearer adm_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		return request
	}()); ok {
		t.Fatalf("expected api admin token resolve failure")
	}
	apiAdminProtectedUnauthorizedRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/server/settings", nil)
	apiAdminProtectedUnauthorizedResponse := httptest.NewRecorder()
	apiAdminResolveFailureHandler.handleProtectedSurface(apiAdminProtectedUnauthorizedResponse, apiAdminProtectedUnauthorizedRequest, "server/settings")
	if apiAdminProtectedUnauthorizedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized api admin protected surface, got %d", apiAdminProtectedUnauthorizedResponse.Code)
	}

	apiUserResolveFailureHandler := NewAPIUserHandler("/api/v1/users", service.NewAuthService(handlerFailingStore{
		userError: errors.New("user missing"),
		token: model.Token{
			ID:          1,
			OwnerType:   model.TokenOwnerUser,
			OwnerID:     1,
			TokenHash:   service.HashToken("usr_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
			TokenPrefix: "usr_aaaa",
			ExpiresAt:   time.Unix(2_000_000_000, 0),
		},
	}, service.DefaultSessionConfig()), testDomainService()).(APIUserHandler)
	if _, _, ok := apiUserResolveFailureHandler.resolveUser(func() *http.Request {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/users/profile", nil)
		request.Header.Set("Authorization", "Bearer usr_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		return request
	}()); ok {
		t.Fatalf("expected api user resolve failure")
	}
}

func TestPrefersText(t *testing.T) {
	if !prefersText("/health.txt", "", "") {
		t.Fatalf("expected txt route to prefer text")
	}
	if !prefersText("/health", "text/plain", "Mozilla/5.0") {
		t.Fatalf("expected plain accept header to prefer text")
	}
	if prefersText("/health", "application/json", "Mozilla/5.0") {
		t.Fatalf("expected explicit json accept header not to prefer text")
	}
	if !prefersText("/health", "", "curl/8.0") {
		t.Fatalf("expected cli user-agent to prefer text")
	}
	if prefersText("/health", "", "Mozilla/5.0") {
		t.Fatalf("expected browser with no plain preference not to prefer text")
	}
	if !prefersJSON("application/json") {
		t.Fatalf("expected json accept header to prefer json")
	}
	if prefersJSON("text/plain") {
		t.Fatalf("expected non-json accept header not to prefer json")
	}
	if firstNonEmpty() != "" {
		t.Fatalf("expected empty firstNonEmpty result")
	}
	forwardedRequest := httptest.NewRequest(http.MethodGet, "/auth", nil)
	forwardedRequest.Header.Set("X-Forwarded-For", "203.0.113.10, 127.0.0.1")
	if requestIP(forwardedRequest) != "203.0.113.10" {
		t.Fatalf("expected forwarded request ip")
	}
	if joinPath() != "/" {
		t.Fatalf("expected empty joinPath root")
	}
	secureRequest := httptest.NewRequest(http.MethodGet, "/auth", nil)
	if !(SessionCookieConfig{Secure: "always"}).isSecure(secureRequest) {
		t.Fatalf("expected always secure cookie")
	}
	if (SessionCookieConfig{Secure: "never"}).isSecure(secureRequest) {
		t.Fatalf("expected never secure cookie")
	}
}

func newTestAuthService(t *testing.T) service.AuthService {
	t.Helper()

	_, authService := newTestRuntimeStore(t)
	return authService
}

func newTestRuntimeStore(t *testing.T) (*store.MemoryStore, service.AuthService) {
	t.Helper()

	memoryStore := store.NewMemoryStore()
	passwordHash, hashError := service.HashPassword("correct horse battery staple")
	if hashError != nil {
		t.Fatalf("hash password: %v", hashError)
	}

	if _, saveError := memoryStore.SaveUser(context.Background(), model.User{
		Username:     "alice",
		DisplayName:  "Alice Example",
		AccountEmail: "alice@example.com",
		PasswordHash: passwordHash,
		Enabled:      true,
		Visibility:   model.UserVisibilityPublic,
	}); saveError != nil {
		t.Fatalf("save user: %v", saveError)
	}

	if _, saveError := memoryStore.SaveUser(context.Background(), model.User{
		Username:     "bob",
		DisplayName:  "Bob Hidden",
		AccountEmail: "bob@example.com",
		PasswordHash: passwordHash,
		Enabled:      true,
		Visibility:   model.UserVisibilityPrivate,
	}); saveError != nil {
		t.Fatalf("save private user: %v", saveError)
	}

	if _, saveError := memoryStore.SaveAdmin(context.Background(), model.Admin{
		Username:     "root-admin",
		AccountEmail: "root@example.com",
		PasswordHash: passwordHash,
		Enabled:      true,
		Role:         model.AdminRoleAdmin,
	}); saveError != nil {
		t.Fatalf("save admin: %v", saveError)
	}

	acmeOrg, saveOrgError := memoryStore.SaveOrganization(context.Background(), model.Organization{
		Slug:       "acme",
		Name:       "Acme",
		Visibility: model.OrganizationVisibilityPublic,
		OwnerID:    1,
		CreatedAt:  time.Unix(1_700_000_000, 0),
		UpdatedAt:  time.Unix(1_700_000_000, 0),
	})
	if saveOrgError != nil {
		t.Fatalf("save public org: %v", saveOrgError)
	}
	secretOrg, saveOrgError := memoryStore.SaveOrganization(context.Background(), model.Organization{
		Slug:       "secret",
		Name:       "Secret",
		Visibility: model.OrganizationVisibilityPrivate,
		OwnerID:    1,
		CreatedAt:  time.Unix(1_700_000_100, 0),
		UpdatedAt:  time.Unix(1_700_000_100, 0),
	})
	if saveOrgError != nil {
		t.Fatalf("save private org: %v", saveOrgError)
	}

	acmePrefs := model.DefaultOrganizationPreferences()
	acmePrefs.OrgID = acmeOrg.ID
	if _, saveError := memoryStore.SaveOrganizationPreferences(context.Background(), acmePrefs); saveError != nil {
		t.Fatalf("save public org preferences: %v", saveError)
	}
	secretPrefs := model.DefaultOrganizationPreferences()
	secretPrefs.OrgID = secretOrg.ID
	if _, saveError := memoryStore.SaveOrganizationPreferences(context.Background(), secretPrefs); saveError != nil {
		t.Fatalf("save private org preferences: %v", saveError)
	}

	for _, member := range []model.OrganizationMember{
		{OrgID: acmeOrg.ID, UserID: 1, Role: model.OrganizationRoleOwner, CreatedAt: time.Unix(1_700_000_200, 0)},
		{OrgID: acmeOrg.ID, UserID: 2, Role: model.OrganizationRoleMember, CreatedAt: time.Unix(1_700_000_201, 0)},
		{OrgID: secretOrg.ID, UserID: 1, Role: model.OrganizationRoleOwner, CreatedAt: time.Unix(1_700_000_202, 0)},
	} {
		if _, saveError := memoryStore.SaveOrganizationMember(context.Background(), member); saveError != nil {
			t.Fatalf("save org member: %v", saveError)
		}
	}

	return memoryStore, service.NewAuthService(memoryStore, service.DefaultSessionConfig())
}

func testDomainService() service.DomainService {
	return newEnabledDomainService(store.NewMemoryStore())
}

func testAsteriskService() service.AsteriskService {
	memoryStore := store.NewMemoryStore()
	_, _ = memoryStore.SaveAsteriskState(context.Background(), model.AsteriskState{
		MinimumSupportedVersion: "12",
		DetectedVersion:         "20.5.1",
		DetectionStatus:         "detected",
		HealthStatus:            model.AsteriskHealthReady,
		ChannelDrivers:          []string{"pjsip"},
		EndpointStacks:          []string{"pjsip"},
		Codecs:                  []string{"ulaw", "alaw"},
		Capabilities: []model.AsteriskCapability{
			{Key: "recordings", Label: "Recordings", Family: "media", Available: true},
			{Key: "voicemail", Label: "Voicemail", Family: "media", Available: true},
			{Key: "prompts", Label: "Prompts", Family: "media", Available: true},
			{Key: "music_on_hold", Label: "Music on Hold", Family: "media", Available: true},
			{Key: "fax", Label: "Fax", Family: "fax", Available: true},
			{Key: "queues", Label: "Queues", Family: "queue", Available: true},
			{Key: "conferences", Label: "Conferences", Family: "conference", Available: true},
			{Key: "browser_calling", Label: "Browser Calling", Family: "webphone", Available: true},
			{Key: "tls", Label: "TLS", Family: "security", Available: true},
			{Key: "presence", Label: "Presence", Family: "messaging", Available: true},
		},
		Subsystems: []model.AsteriskManagedSubsystem{
			{Key: "fax_backend", Label: "Fax backend", Provider: "hylafax+", Healthy: true},
			{Key: "tts_engine", Label: "TTS engine", Provider: "flite", Healthy: true},
			{Key: "music_on_hold", Label: "Music on hold", Provider: "mixed", Healthy: true},
			{Key: "messaging_backend", Label: "Messaging backend", Provider: "xmpp", Healthy: true},
		},
	})
	return service.NewAsteriskService(memoryStore)
}

func testPBXService() service.PBXService {
	memoryStore := store.NewMemoryStore()
	_, _ = memoryStore.SaveAsteriskState(context.Background(), model.AsteriskState{
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
	})
	pbxService := service.NewPBXService(memoryStore, memoryStore)
	_, _ = pbxService.CreateExtension(context.Background(), model.Extension{Number: "1000", DisplayName: "Alice", Technology: "pjsip", Endpoint: "alice"})
	_, _ = pbxService.CreateQueue(context.Background(), model.Queue{Name: "Support", Strategy: "ringall"})
	return pbxService
}

func newEnabledDomainService(domainStore store.DomainStore) service.DomainService {
	return service.NewDomainService(domainStore, model.DomainConstraints{
		Enabled:           true,
		MaxDomainsPerUser: 5,
		MaxDomainsPerOrg:  20,
		RequireSSL:        true,
		AllowApex:         true,
		AllowSubdomain:    true,
		AllowWildcard:     false,
		VerificationTTL:   24 * time.Hour,
		SSLRenewalDays:    7,
		Reserved:          []string{"localhost", "*.local", "*.test", "*.example", "*.invalid"},
		BlockedPatterns:   []string{`.*\.(gov|mil|edu)$`},
	}, []string{"example.invalid"}, nil)
}
