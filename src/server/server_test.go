package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/casapps/caspbx/src/config"
	"github.com/casapps/caspbx/src/server/model"
	"github.com/casapps/caspbx/src/server/service"
	"github.com/casapps/caspbx/src/server/store"
)

type failingRuntimeStore struct {
	*store.MemoryStore
}

func (runtimeStore failingRuntimeStore) SaveAsteriskState(context.Context, model.AsteriskState) (model.AsteriskState, error) {
	return model.AsteriskState{}, errors.New("save failed")
}

func TestNewApp(t *testing.T) {
	memoryStore := store.NewMemoryStore()
	passwordHash, hashError := service.HashPassword("correct horse battery staple")
	if hashError != nil {
		t.Fatalf("hash password: %v", hashError)
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
	organization, saveOrgError := memoryStore.SaveOrganization(context.Background(), model.Organization{
		Slug:       "acme",
		Name:       "Acme",
		Visibility: model.OrganizationVisibilityPublic,
		OwnerID:    1,
		CreatedAt:  time.Unix(1_700_000_000, 0),
		UpdatedAt:  time.Unix(1_700_000_000, 0),
	})
	if saveOrgError != nil {
		t.Fatalf("save organization: %v", saveOrgError)
	}
	orgPreferences := model.DefaultOrganizationPreferences()
	orgPreferences.OrgID = organization.ID
	if _, saveError := memoryStore.SaveOrganizationPreferences(context.Background(), orgPreferences); saveError != nil {
		t.Fatalf("save organization preferences: %v", saveError)
	}
	if _, saveError := memoryStore.SaveOrganizationMember(context.Background(), model.OrganizationMember{
		OrgID:     organization.ID,
		UserID:    1,
		Role:      model.OrganizationRoleOwner,
		CreatedAt: time.Unix(1_700_000_100, 0),
	}); saveError != nil {
		t.Fatalf("save organization member: %v", saveError)
	}

	app, appError := NewAppWithStore(DefaultAPIVersion, "admin", "caspbx", "dev", "unknown", "https://example.invalid", config.DefaultConfig().Server, memoryStore)
	if appError != nil {
		t.Fatalf("expected app to build, got %v", appError)
	}

	rootRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	rootResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(rootResponse, rootRequest)
	if rootResponse.Code != http.StatusOK {
		t.Fatalf("expected root response 200, got %d", rootResponse.Code)
	}

	healthRequest := httptest.NewRequest(http.MethodGet, "/health", nil)
	healthRequest.Header.Set("Accept", "application/json")
	healthResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(healthResponse, healthRequest)
	if !strings.Contains(healthResponse.Body.String(), "\"status\":\"ok\"") {
		t.Fatalf("unexpected health response %q", healthResponse.Body.String())
	}

	authRequest := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	authRequest.Header.Set("Accept", "text/plain")
	authResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(authResponse, authRequest)
	if authResponse.Code != http.StatusOK {
		t.Fatalf("expected auth response 200, got %d", authResponse.Code)
	}

	versionRequest := httptest.NewRequest(http.MethodGet, "/version", nil)
	versionResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(versionResponse, versionRequest)
	if !strings.Contains(versionResponse.Body.String(), "caspbx dev (unknown)") {
		t.Fatalf("unexpected version response %q", versionResponse.Body.String())
	}

	orgRequest := httptest.NewRequest(http.MethodGet, "/orgs/acme", nil)
	orgRequest.Header.Set("Accept", "application/json")
	orgResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(orgResponse, orgRequest)
	if orgResponse.Code != http.StatusOK || !strings.Contains(orgResponse.Body.String(), "\"slug\":\"acme\"") {
		t.Fatalf("unexpected org response %d %q", orgResponse.Code, orgResponse.Body.String())
	}

	orgAPIRequest := httptest.NewRequest(http.MethodGet, "/api/v1/orgs/acme", nil)
	orgAPIResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(orgAPIResponse, orgAPIRequest)
	if orgAPIResponse.Code != http.StatusOK || !strings.Contains(orgAPIResponse.Body.String(), "\"name\":\"Acme\"") {
		t.Fatalf("unexpected org api response %d %q", orgAPIResponse.Code, orgAPIResponse.Body.String())
	}

	asteriskAdminRequest := httptest.NewRequest(http.MethodGet, "/admin/server/asterisk/fax", nil)
	asteriskAdminRequest.Header.Set("Accept", "text/plain")
	asteriskAdminResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(asteriskAdminResponse, asteriskAdminRequest)
	if asteriskAdminResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected asterisk admin response %q", asteriskAdminResponse.Body.String())
	}

	if !strings.Contains(app.Summary(), "Health path: /health") {
		t.Fatalf("unexpected app summary %q", app.Summary())
	}
}

func TestNewAppError(t *testing.T) {
	if _, appError := NewApp(DefaultAPIVersion, "api", "caspbx", "dev", "unknown", ""); appError == nil {
		t.Fatalf("expected invalid admin path to fail")
	}
	if _, appError := NewAppWithStore(DefaultAPIVersion, "admin", "caspbx", "dev", "unknown", "https://example.invalid", config.DefaultConfig().Server, failingRuntimeStore{MemoryStore: store.NewMemoryStore()}); appError == nil || appError.Error() != "save failed" {
		t.Fatalf("expected asterisk state save failure, got %v", appError)
	}
}

func TestSameSiteModeAndDuration(t *testing.T) {
	if sameSiteMode("strict") != http.SameSiteStrictMode {
		t.Fatalf("expected strict same-site mode")
	}
	if sameSiteMode("none") != http.SameSiteNoneMode {
		t.Fatalf("expected none same-site mode")
	}
	if sameSiteMode("lax") != http.SameSiteLaxMode {
		t.Fatalf("expected lax same-site mode")
	}
	if timeDurationHours(2) != 2*time.Hour {
		t.Fatalf("expected 2 hour duration")
	}
}

func TestCustomDomainHelpers(t *testing.T) {
	serverConfig := config.DefaultConfig().Server
	serverConfig.Features.CustomDomains.Enabled = true

	constraints := customDomainConstraints(serverConfig)
	if !constraints.Enabled || constraints.MaxDomainsPerUser != serverConfig.Features.CustomDomains.MaxDomainsPerUser {
		t.Fatalf("unexpected custom domain constraints %+v", constraints)
	}
	if hosts := platformHosts("https://pbx.example.com"); len(hosts) != 1 || hosts[0] != "pbx.example.com" {
		t.Fatalf("unexpected platform hosts %+v", hosts)
	}
	if hosts := platformHosts("://bad-url"); hosts != nil {
		t.Fatalf("expected invalid platform host parsing to return nil, got %+v", hosts)
	}
}

func TestAsteriskStateHelper(t *testing.T) {
	serverConfig := config.DefaultConfig().Server
	serverConfig.Asterisk.DetectedVersion = "20.5.1"
	serverConfig.Asterisk.DetectionStatus = "detected"
	serverConfig.Asterisk.HealthStatus = "ready"
	serverConfig.Asterisk.ChannelDrivers = []string{"pjsip"}
	serverConfig.Asterisk.EndpointStacks = []string{"pjsip"}
	serverConfig.Asterisk.Codecs = []string{"ulaw"}
	serverConfig.Asterisk.Capabilities.Fax = true
	serverConfig.Asterisk.Capabilities.BrowserCalling = true
	serverConfig.Asterisk.Subsystems.FaxBackend = "hylafax+"
	serverConfig.Asterisk.Subsystems.MessagingBackend = "xmpp"

	state := asteriskState(serverConfig)
	if state.MinimumSupportedVersion != "12" || state.DetectedVersion != "20.5.1" || len(state.Subsystems) == 0 {
		t.Fatalf("unexpected asterisk state %+v", state)
	}
	if firstNonEmpty("", "value") != "value" || firstNonEmpty("", "") != "" {
		t.Fatalf("unexpected firstNonEmpty helper")
	}
}
