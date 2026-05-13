package handler

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/casapps/caspbx/src/server/model"
	"github.com/casapps/caspbx/src/server/service"
	"github.com/casapps/caspbx/src/server/store"
)

type domainTestResolver struct {
	ips    map[string][]net.IP
	cnames map[string]string
	errs   map[string]error
}

type handlerDomainStore struct {
	domain      model.CustomDomain
	listError   error
	findError   error
	deleteError error
}

func (domainStore handlerDomainStore) SaveCustomDomain(context.Context, model.CustomDomain) (model.CustomDomain, error) {
	return domainStore.domain, nil
}

func (domainStore handlerDomainStore) FindCustomDomainByID(context.Context, int64) (model.CustomDomain, error) {
	return domainStore.domain, domainStore.findError
}

func (domainStore handlerDomainStore) FindCustomDomainByDomain(context.Context, string) (model.CustomDomain, error) {
	if domainStore.findError != nil {
		return model.CustomDomain{}, domainStore.findError
	}
	return domainStore.domain, nil
}

func (domainStore handlerDomainStore) FindDomainByHost(context.Context, string) (model.CustomDomain, error) {
	return domainStore.FindCustomDomainByDomain(context.Background(), "")
}

func (domainStore handlerDomainStore) ListCustomDomains(context.Context) ([]model.CustomDomain, error) {
	if domainStore.listError != nil {
		return nil, domainStore.listError
	}
	return []model.CustomDomain{domainStore.domain}, nil
}

func (domainStore handlerDomainStore) ListCustomDomainsByOwner(context.Context, model.DomainOwnerType, int64) ([]model.CustomDomain, error) {
	if domainStore.listError != nil {
		return nil, domainStore.listError
	}
	return []model.CustomDomain{domainStore.domain}, nil
}

func (domainStore handlerDomainStore) DeleteCustomDomainByID(context.Context, int64) error {
	return domainStore.deleteError
}

func (resolver domainTestResolver) LookupIP(host string) ([]net.IP, error) {
	if err := resolver.errs[host]; err != nil {
		return nil, err
	}
	return resolver.ips[host], nil
}

func (resolver domainTestResolver) LookupCNAME(host string) (string, error) {
	if err := resolver.errs["cname:"+host]; err != nil {
		return "", err
	}
	return resolver.cnames[host], nil
}

func TestDomainHelpers(t *testing.T) {
	requestBody, parseError := decodeDomainRequest(httptest.NewRequest(http.MethodPost, "/users/domains", strings.NewReader(`{"domain":"api.example.com"}`)))
	if parseError != nil || requestBody.Domain != "api.example.com" {
		t.Fatalf("expected domain request decode, got %v / %+v", parseError, requestBody)
	}
	if _, parseError := decodeDomainRequest(httptest.NewRequest(http.MethodPost, "/users/domains", strings.NewReader("{"))); parseError == nil {
		t.Fatalf("expected invalid domain request decode error")
	}
	if domainName, action, found := domainRouteParts("domains/api.example.com/verify", "domains"); !found || domainName != "api.example.com" || action != "verify" {
		t.Fatalf("unexpected domain route parts %q %q %t", domainName, action, found)
	}
	if _, _, found := domainRouteParts("profile", "domains"); found {
		t.Fatalf("expected non-domain route lookup to fail")
	}

	instructions := service.DomainDNSInstructions{Target: "custom.example.com", TargetIPs: []string{"203.0.113.50"}, Instructions: "point dns"}
	response := domainResponse(model.CustomDomain{
		ID:                 10,
		Domain:             "api.example.com",
		OwnerType:          model.DomainOwnerTypeUser,
		OwnerID:            11,
		Status:             model.DomainStatusActive,
		VerificationStatus: model.VerificationStatusVerified,
		SSLStatus:          model.SSLStatusActive,
	}, instructions)
	if response["domain"] != "api.example.com" {
		t.Fatalf("unexpected domain response %+v", response)
	}

	listResponse := httptest.NewRecorder()
	writeDomainTextList(listResponse, []model.CustomDomain{{Domain: "api.example.com", Status: model.DomainStatusPending, VerificationStatus: model.VerificationStatusPending, SSLStatus: model.SSLStatusNone}}, instructions)
	if !strings.Contains(listResponse.Body.String(), "api.example.com") {
		t.Fatalf("expected domain list text response, got %q", listResponse.Body.String())
	}
	detailResponse := httptest.NewRecorder()
	writeDomainTextDetail(detailResponse, model.CustomDomain{Domain: "api.example.com", Status: model.DomainStatusActive, VerificationStatus: model.VerificationStatusVerified, SSLStatus: model.SSLStatusActive}, instructions)
	if !strings.Contains(detailResponse.Body.String(), "Domain: api.example.com") {
		t.Fatalf("expected domain detail text response, got %q", detailResponse.Body.String())
	}

	internalErrorResponse := httptest.NewRecorder()
	writeDomainServiceError(internalErrorResponse, errors.New("boom"))
	if internalErrorResponse.Code != http.StatusInternalServerError {
		t.Fatalf("expected internal error response, got %d", internalErrorResponse.Code)
	}
	domainErrorResponse := httptest.NewRecorder()
	writeDomainServiceError(domainErrorResponse, &service.DomainError{Code: "DOMAIN_EXISTS", Message: "exists"})
	if domainErrorResponse.Code != http.StatusConflict {
		t.Fatalf("expected domain exists conflict response, got %d", domainErrorResponse.Code)
	}
	if domainErrorStatus("DOMAIN_NOT_FOUND") != http.StatusNotFound || domainErrorStatus("DOMAIN_SUSPENDED") != http.StatusForbidden || domainErrorStatus("unknown") != http.StatusInternalServerError {
		t.Fatalf("unexpected domain error status mapping")
	}

	domains := domainCollection([]model.CustomDomain{{Domain: "api.example.com"}}, instructions)
	if len(domains) != 1 {
		t.Fatalf("expected domain collection conversion")
	}
}

func TestDomainHandlers(t *testing.T) {
	memoryStore, authService := newTestRuntimeStore(t)
	now := time.Unix(1_700_300_000, 0).UTC()
	domainService := service.NewDomainService(memoryStore, model.DomainConstraints{
		Enabled:           true,
		MaxDomainsPerUser: 5,
		MaxDomainsPerOrg:  5,
		RequireSSL:        true,
		AllowApex:         true,
		AllowSubdomain:    true,
		AllowWildcard:     false,
		VerificationTTL:   24 * time.Hour,
		SSLRenewalDays:    7,
		Reserved:          []string{"localhost", "*.local", "*.test", "*.example", "*.invalid"},
		BlockedPatterns:   []string{`.*\.(gov|mil|edu)$`},
	}, []string{"custom.example.com"}, []net.IP{net.ParseIP("203.0.113.50")}, service.WithDomainClock(func() time.Time { return now }), service.WithDomainResolver(domainTestResolver{
		ips: map[string][]net.IP{
			"user.example.com":    {net.ParseIP("203.0.113.50")},
			"apiuser.example.com": {net.ParseIP("203.0.113.50")},
			"org.example.com":     {net.ParseIP("203.0.113.50")},
		},
	}), service.WithDomainChallengeAvailability(false, true))
	userCookie := SessionCookieConfig{Name: "user_session", Path: "/", HTTPOnly: true, Secure: "auto", SameSite: http.SameSiteLaxMode}
	adminCookie := SessionCookieConfig{Name: "admin_session", Path: "/admin", HTTPOnly: true, Secure: "auto", SameSite: http.SameSiteLaxMode}

	userHandler := NewUserHandler("/users", authService, domainService, userCookie)
	apiUserHandler := NewAPIUserHandler("/api/v1/users", authService, domainService)
	orgHandler := NewOrgHandler("/orgs", authService, domainService, memoryStore, userCookie)
	apiOrgHandler := NewAPIOrgHandler("/api/v1/orgs", authService, domainService, memoryStore)
	adminHandler := NewAdminHandler("/admin", authService, domainService, testAsteriskService(), testPBXService(), adminCookie)
	apiAdminHandler := NewAPIAdminHandler("/api/v1/admin", authService, domainService, testAsteriskService(), testPBXService())

	_, aliceSession, loginError := authService.AuthenticateUser(context.Background(), "alice", "correct horse battery staple", "127.0.0.1", "curl/8.0")
	if loginError != nil {
		t.Fatalf("authenticate alice: %v", loginError)
	}
	_, bobSession, loginError := authService.AuthenticateUser(context.Background(), "bob", "correct horse battery staple", "127.0.0.1", "curl/8.0")
	if loginError != nil {
		t.Fatalf("authenticate bob: %v", loginError)
	}
	_, aliceToken, tokenError := authService.AuthenticateUserAPI(context.Background(), "alice", "correct horse battery staple")
	if tokenError != nil {
		t.Fatalf("authenticate alice api: %v", tokenError)
	}
	_, adminSession, loginError := authService.AuthenticateAdmin(context.Background(), "root-admin", "correct horse battery staple", "127.0.0.1", "curl/8.0")
	if loginError != nil {
		t.Fatalf("authenticate admin: %v", loginError)
	}
	_, adminToken, tokenError := authService.AuthenticateAdminAPI(context.Background(), "root-admin", "correct horse battery staple")
	if tokenError != nil {
		t.Fatalf("authenticate admin api: %v", tokenError)
	}

	createUserDomainRequest := httptest.NewRequest(http.MethodPost, "/users/domains", strings.NewReader(`{"domain":"user.example.com"}`))
	createUserDomainRequest.AddCookie(&http.Cookie{Name: "user_session", Value: aliceSession.Token})
	createUserDomainRequest.Header.Set("Content-Type", "application/json")
	createUserDomainResponse := httptest.NewRecorder()
	userHandler.ServeHTTP(createUserDomainResponse, createUserDomainRequest)
	if createUserDomainResponse.Code != http.StatusCreated {
		t.Fatalf("expected user domain create status 201, got %d", createUserDomainResponse.Code)
	}

	userListRequest := httptest.NewRequest(http.MethodGet, "/users/domains", nil)
	userListRequest.AddCookie(&http.Cookie{Name: "user_session", Value: aliceSession.Token})
	userListRequest.Header.Set("Accept", "text/plain")
	userListResponse := httptest.NewRecorder()
	userHandler.ServeHTTP(userListResponse, userListRequest)
	if !strings.Contains(userListResponse.Body.String(), "user.example.com") {
		t.Fatalf("expected user domain list body, got %q", userListResponse.Body.String())
	}

	verifyUserDomainRequest := httptest.NewRequest(http.MethodPost, "/users/domains/user.example.com/verify", nil)
	verifyUserDomainRequest.AddCookie(&http.Cookie{Name: "user_session", Value: aliceSession.Token})
	verifyUserDomainResponse := httptest.NewRecorder()
	userHandler.ServeHTTP(verifyUserDomainResponse, verifyUserDomainRequest)
	if !strings.Contains(verifyUserDomainResponse.Body.String(), "\"verification_status\":\"verified\"") {
		t.Fatalf("unexpected user verify response %q", verifyUserDomainResponse.Body.String())
	}

	configureUserSSLRequest := httptest.NewRequest(http.MethodPost, "/users/domains/user.example.com/ssl", strings.NewReader(`{"challenge":"auto"}`))
	configureUserSSLRequest.AddCookie(&http.Cookie{Name: "user_session", Value: aliceSession.Token})
	configureUserSSLRequest.Header.Set("Content-Type", "application/json")
	configureUserSSLResponse := httptest.NewRecorder()
	userHandler.ServeHTTP(configureUserSSLResponse, configureUserSSLRequest)
	if !strings.Contains(configureUserSSLResponse.Body.String(), "\"ssl_challenge\":\"http-01\"") {
		t.Fatalf("unexpected user ssl response %q", configureUserSSLResponse.Body.String())
	}

	userDetailRequest := httptest.NewRequest(http.MethodGet, "/users/domains/user.example.com", nil)
	userDetailRequest.AddCookie(&http.Cookie{Name: "user_session", Value: aliceSession.Token})
	userDetailResponse := httptest.NewRecorder()
	userHandler.ServeHTTP(userDetailResponse, userDetailRequest)
	if !strings.Contains(userDetailResponse.Body.String(), "Domain: user.example.com") {
		t.Fatalf("unexpected user domain detail %q", userDetailResponse.Body.String())
	}

	badCreateRequest := httptest.NewRequest(http.MethodPost, "/users/domains", strings.NewReader("{"))
	badCreateRequest.AddCookie(&http.Cookie{Name: "user_session", Value: aliceSession.Token})
	badCreateRequest.Header.Set("Content-Type", "application/json")
	badCreateResponse := httptest.NewRecorder()
	userHandler.ServeHTTP(badCreateResponse, badCreateRequest)
	if badCreateResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected bad create request status 400, got %d", badCreateResponse.Code)
	}

	deleteUserDomainRequest := httptest.NewRequest(http.MethodDelete, "/users/domains/user.example.com", nil)
	deleteUserDomainRequest.AddCookie(&http.Cookie{Name: "user_session", Value: aliceSession.Token})
	deleteUserDomainResponse := httptest.NewRecorder()
	userHandler.ServeHTTP(deleteUserDomainResponse, deleteUserDomainRequest)
	if deleteUserDomainResponse.Code != http.StatusOK {
		t.Fatalf("expected user domain delete status 200, got %d", deleteUserDomainResponse.Code)
	}

	disabledHandler := NewUserHandler("/users", authService, service.NewDomainService(memoryStore, model.DomainConstraints{}, nil, nil), userCookie)
	disabledRequest := httptest.NewRequest(http.MethodGet, "/users/domains", nil)
	disabledRequest.AddCookie(&http.Cookie{Name: "user_session", Value: aliceSession.Token})
	disabledResponse := httptest.NewRecorder()
	disabledHandler.ServeHTTP(disabledResponse, disabledRequest)
	if disabledResponse.Code != http.StatusNotFound {
		t.Fatalf("expected disabled domain feature response 404, got %d", disabledResponse.Code)
	}

	apiCreateRequest := httptest.NewRequest(http.MethodPost, "/api/v1/users/domains", strings.NewReader(`{"domain":"apiuser.example.com"}`))
	apiCreateRequest.Header.Set("Authorization", "Bearer "+aliceToken.Value)
	apiCreateRequest.Header.Set("Content-Type", "application/json")
	apiCreateResponse := httptest.NewRecorder()
	apiUserHandler.ServeHTTP(apiCreateResponse, apiCreateRequest)
	if apiCreateResponse.Code != http.StatusCreated {
		t.Fatalf("expected api user domain create status 201, got %d", apiCreateResponse.Code)
	}

	noMemberOrgRequest := httptest.NewRequest(http.MethodGet, "/orgs/acme/domains", nil)
	noMemberOrgResponse := httptest.NewRecorder()
	orgHandler.ServeHTTP(noMemberOrgResponse, noMemberOrgRequest)
	if noMemberOrgResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected anonymous org domain response 401, got %d", noMemberOrgResponse.Code)
	}

	forbiddenOrgRequest := httptest.NewRequest(http.MethodGet, "/orgs/acme/domains", nil)
	forbiddenOrgRequest.AddCookie(&http.Cookie{Name: "user_session", Value: bobSession.Token})
	forbiddenOrgResponse := httptest.NewRecorder()
	orgHandler.ServeHTTP(forbiddenOrgResponse, forbiddenOrgRequest)
	if forbiddenOrgResponse.Code != http.StatusForbidden {
		t.Fatalf("expected org member domain response 403, got %d", forbiddenOrgResponse.Code)
	}

	createOrgDomainRequest := httptest.NewRequest(http.MethodPost, "/orgs/acme/domains", strings.NewReader(`{"domain":"org.example.com"}`))
	createOrgDomainRequest.AddCookie(&http.Cookie{Name: "user_session", Value: aliceSession.Token})
	createOrgDomainRequest.Header.Set("Content-Type", "application/json")
	createOrgDomainResponse := httptest.NewRecorder()
	orgHandler.ServeHTTP(createOrgDomainResponse, createOrgDomainRequest)
	if createOrgDomainResponse.Code != http.StatusCreated {
		t.Fatalf("expected org domain create status 201, got %d", createOrgDomainResponse.Code)
	}

	verifyOrgDomainRequest := httptest.NewRequest(http.MethodPost, "/api/v1/orgs/acme/domains/org.example.com/verify", nil)
	verifyOrgDomainRequest.Header.Set("Authorization", "Bearer "+aliceToken.Value)
	verifyOrgDomainResponse := httptest.NewRecorder()
	apiOrgHandler.ServeHTTP(verifyOrgDomainResponse, verifyOrgDomainRequest)
	if !strings.Contains(verifyOrgDomainResponse.Body.String(), "\"resolved_to\":\"203.0.113.50\"") {
		t.Fatalf("unexpected org verify response %q", verifyOrgDomainResponse.Body.String())
	}

	configureOrgSSLRequest := httptest.NewRequest(http.MethodPost, "/orgs/acme/domains/org.example.com/ssl", strings.NewReader(`{"provider":"cloudflare","challenge":"dns-01"}`))
	configureOrgSSLRequest.AddCookie(&http.Cookie{Name: "user_session", Value: aliceSession.Token})
	configureOrgSSLRequest.Header.Set("Content-Type", "application/json")
	configureOrgSSLResponse := httptest.NewRecorder()
	orgHandler.ServeHTTP(configureOrgSSLResponse, configureOrgSSLRequest)
	if !strings.Contains(configureOrgSSLResponse.Body.String(), "\"ssl_provider\":\"cloudflare\"") {
		t.Fatalf("unexpected org ssl response %q", configureOrgSSLResponse.Body.String())
	}

	badOrgActionRequest := httptest.NewRequest(http.MethodGet, "/orgs/acme/domains/org.example.com/bad", nil)
	badOrgActionRequest.AddCookie(&http.Cookie{Name: "user_session", Value: aliceSession.Token})
	badOrgActionResponse := httptest.NewRecorder()
	orgHandler.ServeHTTP(badOrgActionResponse, badOrgActionRequest)
	if badOrgActionResponse.Code != http.StatusNotFound {
		t.Fatalf("expected unknown org domain action status 404, got %d", badOrgActionResponse.Code)
	}

	adminUnauthorizedRequest := httptest.NewRequest(http.MethodGet, "/admin/server/domains", nil)
	adminUnauthorizedResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(adminUnauthorizedResponse, adminUnauthorizedRequest)
	if adminUnauthorizedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected admin unauthorized domains response 401, got %d", adminUnauthorizedResponse.Code)
	}

	adminListRequest := httptest.NewRequest(http.MethodGet, "/admin/server/domains", nil)
	adminListRequest.AddCookie(&http.Cookie{Name: "admin_session", Value: adminSession.Token})
	adminListResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(adminListResponse, adminListRequest)
	if !strings.Contains(adminListResponse.Body.String(), "\"domain\":\"org.example.com\"") {
		t.Fatalf("unexpected admin list response %q", adminListResponse.Body.String())
	}

	suspendRequest := httptest.NewRequest(http.MethodPost, "/admin/server/domains/org.example.com/suspend", strings.NewReader(`{"reason":"abuse"}`))
	suspendRequest.AddCookie(&http.Cookie{Name: "admin_session", Value: adminSession.Token})
	suspendRequest.Header.Set("Content-Type", "application/json")
	suspendResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(suspendResponse, suspendRequest)
	if !strings.Contains(suspendResponse.Body.String(), "\"suspension_reason\":\"abuse\"") {
		t.Fatalf("unexpected admin suspend response %q", suspendResponse.Body.String())
	}

	unsuspendRequest := httptest.NewRequest(http.MethodPost, "/admin/server/domains/org.example.com/unsuspend", nil)
	unsuspendRequest.AddCookie(&http.Cookie{Name: "admin_session", Value: adminSession.Token})
	unsuspendResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(unsuspendResponse, unsuspendRequest)
	if !strings.Contains(unsuspendResponse.Body.String(), "\"status\":\"active\"") {
		t.Fatalf("unexpected admin unsuspend response %q", unsuspendResponse.Body.String())
	}

	renewRequest := httptest.NewRequest(http.MethodPost, "/admin/server/domains/org.example.com/ssl/renew", nil)
	renewRequest.AddCookie(&http.Cookie{Name: "admin_session", Value: adminSession.Token})
	renewResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(renewResponse, renewRequest)
	if !strings.Contains(renewResponse.Body.String(), "\"ssl_status\":\"pending\"") {
		t.Fatalf("unexpected admin renew response %q", renewResponse.Body.String())
	}

	apiAdminListRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/server/domains", nil)
	apiAdminListRequest.Header.Set("Authorization", "Bearer "+adminToken.Value)
	apiAdminListResponse := httptest.NewRecorder()
	apiAdminHandler.ServeHTTP(apiAdminListResponse, apiAdminListRequest)
	if apiAdminListResponse.Code != http.StatusOK {
		t.Fatalf("expected api admin domains status 200, got %d", apiAdminListResponse.Code)
	}

	deleteOrgDomainRequest := httptest.NewRequest(http.MethodDelete, "/admin/server/domains/org.example.com", nil)
	deleteOrgDomainRequest.AddCookie(&http.Cookie{Name: "admin_session", Value: adminSession.Token})
	deleteOrgDomainResponse := httptest.NewRecorder()
	adminHandler.ServeHTTP(deleteOrgDomainResponse, deleteOrgDomainRequest)
	if deleteOrgDomainResponse.Code != http.StatusOK {
		t.Fatalf("expected admin domain delete status 200, got %d", deleteOrgDomainResponse.Code)
	}
}

func TestDomainHandlerAdditionalBranches(t *testing.T) {
	baseDomainService := service.NewDomainService(handlerDomainStore{
		domain: model.CustomDomain{
			ID:                 1,
			Domain:             "api.example.com",
			OwnerType:          model.DomainOwnerTypeUser,
			OwnerID:            1,
			OrganizationID:     1,
			VerificationStatus: model.VerificationStatusVerified,
			Status:             model.DomainStatusActive,
			SSLEnabled:         true,
		},
	}, model.DomainConstraints{
		Enabled:           true,
		MaxDomainsPerUser: 5,
		MaxDomainsPerOrg:  5,
		RequireSSL:        true,
		AllowApex:         true,
		AllowSubdomain:    true,
		AllowWildcard:     false,
		VerificationTTL:   time.Hour,
		SSLRenewalDays:    7,
		Reserved:          []string{"localhost"},
		BlockedPatterns:   []string{`.*\.gov$`},
	}, []string{"custom.example.com"}, nil)
	failingDomainService := service.NewDomainService(handlerDomainStore{
		domain:    model.CustomDomain{ID: 1, Domain: "api.example.com", OwnerType: model.DomainOwnerTypeUser, OwnerID: 1},
		listError: errors.New("list failed"),
		findError: errors.New("find failed"),
	}, model.DomainConstraints{
		Enabled:           true,
		MaxDomainsPerUser: 5,
		MaxDomainsPerOrg:  5,
		RequireSSL:        true,
		AllowApex:         true,
		AllowSubdomain:    true,
		AllowWildcard:     false,
		VerificationTTL:   time.Hour,
		SSLRenewalDays:    7,
		Reserved:          []string{"localhost"},
		BlockedPatterns:   []string{`.*\.gov$`},
	}, []string{"custom.example.com"}, nil)
	orgDomainService := service.NewDomainService(handlerDomainStore{
		domain: model.CustomDomain{
			ID:                 2,
			Domain:             "api.example.com",
			OwnerType:          model.DomainOwnerTypeOrg,
			OwnerID:            1,
			OrganizationID:     1,
			VerificationStatus: model.VerificationStatusVerified,
			Status:             model.DomainStatusActive,
			SSLEnabled:         true,
		},
	}, model.DomainConstraints{
		Enabled:           true,
		MaxDomainsPerUser: 5,
		MaxDomainsPerOrg:  5,
		RequireSSL:        true,
		AllowApex:         true,
		AllowSubdomain:    true,
		AllowWildcard:     false,
		VerificationTTL:   time.Hour,
		SSLRenewalDays:    7,
		Reserved:          []string{"localhost"},
		BlockedPatterns:   []string{`.*\.gov$`},
	}, []string{"custom.example.com"}, nil)
	userHandlerValue := UserHandler{routePrefix: "/users", domainService: baseDomainService}
	orgHandlerValue := OrgHandler{routePrefix: "/orgs", domainService: orgDomainService}

	nilBodyRequest := httptest.NewRequest(http.MethodPost, "/users/domains", nil)
	nilBodyRequest.Body = nil
	if decoded, decodeError := decodeDomainRequest(nilBodyRequest); decodeError != nil || decoded.Domain != "" {
		t.Fatalf("expected nil body decode, got %v / %+v", decodeError, decoded)
	}
	if _, _, found := domainRouteParts("domains//verify", "domains"); found {
		t.Fatalf("expected empty domain segment lookup to fail")
	}
	if domainErrorStatus("DOMAIN_INVALID") != http.StatusBadRequest {
		t.Fatalf("expected invalid domain status mapping")
	}

	jsonListResponse := httptest.NewRecorder()
	userHandlerValue.handleDomainSurface(jsonListResponse, httptest.NewRequest(http.MethodGet, "/users/domains", nil), model.User{ID: 1})
	if jsonListResponse.Code != http.StatusOK {
		t.Fatalf("expected json user list response, got %d", jsonListResponse.Code)
	}
	userListJSONRequest := httptest.NewRequest(http.MethodGet, "/users/domains", nil)
	userListJSONRequest.Header.Set("Accept", "application/json")
	userListJSONResponse := httptest.NewRecorder()
	userHandlerValue.handleDomainSurface(userListJSONResponse, userListJSONRequest, model.User{ID: 1})
	if userListJSONResponse.Code != http.StatusOK {
		t.Fatalf("expected explicit json user list response, got %d", userListJSONResponse.Code)
	}
	userListMethodResponse := httptest.NewRecorder()
	userHandlerValue.handleDomainSurface(userListMethodResponse, httptest.NewRequest(http.MethodDelete, "/users/domains", nil), model.User{ID: 1})
	if userListMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected user list method rejection, got %d", userListMethodResponse.Code)
	}
	userNotFoundResponse := httptest.NewRecorder()
	userHandlerValue.handleDomainSurface(userNotFoundResponse, httptest.NewRequest(http.MethodGet, "/users/profile", nil), model.User{ID: 1})
	if userNotFoundResponse.Code != http.StatusNotFound {
		t.Fatalf("expected non-domain user route 404, got %d", userNotFoundResponse.Code)
	}
	userListErrorResponse := httptest.NewRecorder()
	UserHandler{routePrefix: "/users", domainService: failingDomainService}.handleDomainSurface(userListErrorResponse, httptest.NewRequest(http.MethodGet, "/users/domains", nil), model.User{ID: 1})
	if userListErrorResponse.Code != http.StatusInternalServerError {
		t.Fatalf("expected user list error response 500, got %d", userListErrorResponse.Code)
	}
	userCreateErrorResponse := httptest.NewRecorder()
	UserHandler{routePrefix: "/users", domainService: baseDomainService}.handleDomainSurface(userCreateErrorResponse, httptest.NewRequest(http.MethodPost, "/users/domains", strings.NewReader(`{"domain":"localhost"}`)), model.User{ID: 1})
	if userCreateErrorResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected user create validation error, got %d", userCreateErrorResponse.Code)
	}
	userDetailJSONRequest := httptest.NewRequest(http.MethodGet, "/users/domains/api.example.com", nil)
	userDetailJSONRequest.Header.Set("Accept", "application/json")
	userDetailJSONResponse := httptest.NewRecorder()
	userHandlerValue.handleDomainSurface(userDetailJSONResponse, userDetailJSONRequest, model.User{ID: 1})
	if userDetailJSONResponse.Code != http.StatusOK {
		t.Fatalf("expected user detail json response, got %d", userDetailJSONResponse.Code)
	}
	userDetailErrorResponse := httptest.NewRecorder()
	UserHandler{routePrefix: "/users", domainService: failingDomainService}.handleDomainSurface(userDetailErrorResponse, httptest.NewRequest(http.MethodGet, "/users/domains/api.example.com", nil), model.User{ID: 1})
	if userDetailErrorResponse.Code != http.StatusInternalServerError {
		t.Fatalf("expected user detail lookup error, got %d", userDetailErrorResponse.Code)
	}
	userDetailMethodResponse := httptest.NewRecorder()
	userHandlerValue.handleDomainSurface(userDetailMethodResponse, httptest.NewRequest(http.MethodPut, "/users/domains/api.example.com", nil), model.User{ID: 1})
	if userDetailMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected user detail method rejection, got %d", userDetailMethodResponse.Code)
	}
	userDeleteErrorResponse := httptest.NewRecorder()
	UserHandler{routePrefix: "/users", domainService: failingDomainService}.handleDomainSurface(userDeleteErrorResponse, httptest.NewRequest(http.MethodDelete, "/users/domains/api.example.com", nil), model.User{ID: 1})
	if userDeleteErrorResponse.Code != http.StatusInternalServerError {
		t.Fatalf("expected user delete error response 500, got %d", userDeleteErrorResponse.Code)
	}
	userVerifyMethodResponse := httptest.NewRecorder()
	userHandlerValue.handleDomainSurface(userVerifyMethodResponse, httptest.NewRequest(http.MethodGet, "/users/domains/api.example.com/verify", nil), model.User{ID: 1})
	if userVerifyMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected user verify method rejection, got %d", userVerifyMethodResponse.Code)
	}
	userVerifyErrorResponse := httptest.NewRecorder()
	UserHandler{routePrefix: "/users", domainService: failingDomainService}.handleDomainSurface(userVerifyErrorResponse, httptest.NewRequest(http.MethodPost, "/users/domains/api.example.com/verify", nil), model.User{ID: 1})
	if userVerifyErrorResponse.Code != http.StatusInternalServerError {
		t.Fatalf("expected user verify error response 500, got %d", userVerifyErrorResponse.Code)
	}
	userSSLParseResponse := httptest.NewRecorder()
	userHandlerValue.handleDomainSurface(userSSLParseResponse, httptest.NewRequest(http.MethodPost, "/users/domains/api.example.com/ssl", strings.NewReader("{")), model.User{ID: 1})
	if userSSLParseResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected user ssl parse failure, got %d", userSSLParseResponse.Code)
	}
	userSSLErrorResponse := httptest.NewRecorder()
	UserHandler{routePrefix: "/users", domainService: service.NewDomainService(handlerDomainStore{
		domain: model.CustomDomain{Domain: "pending.example.com", OwnerType: model.DomainOwnerTypeUser, OwnerID: 1, VerificationStatus: model.VerificationStatusPending},
	}, model.DomainConstraints{Enabled: true, AllowApex: true, AllowSubdomain: true, VerificationTTL: time.Hour, SSLRenewalDays: 7}, nil, nil)}.handleDomainSurface(userSSLErrorResponse, httptest.NewRequest(http.MethodPost, "/users/domains/pending.example.com/ssl", strings.NewReader(`{}`)), model.User{ID: 1})
	if userSSLErrorResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected user ssl validation failure, got %d", userSSLErrorResponse.Code)
	}
	userSSLMethodResponse := httptest.NewRecorder()
	userHandlerValue.handleDomainSurface(userSSLMethodResponse, httptest.NewRequest(http.MethodGet, "/users/domains/api.example.com/ssl", nil), model.User{ID: 1})
	if userSSLMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected user ssl method rejection, got %d", userSSLMethodResponse.Code)
	}
	userUnknownActionResponse := httptest.NewRecorder()
	userHandlerValue.handleDomainSurface(userUnknownActionResponse, httptest.NewRequest(http.MethodGet, "/users/domains/api.example.com/unknown", nil), model.User{ID: 1})
	if userUnknownActionResponse.Code != http.StatusNotFound {
		t.Fatalf("expected user unknown domain action 404, got %d", userUnknownActionResponse.Code)
	}

	orgDisabledResponse := httptest.NewRecorder()
	OrgHandler{routePrefix: "/orgs", domainService: service.NewDomainService(store.NewMemoryStore(), model.DomainConstraints{}, nil, nil)}.handleDomainSurface(orgDisabledResponse, httptest.NewRequest(http.MethodGet, "/orgs/acme/domains", nil), orgAccessContext{}, "domains")
	if orgDisabledResponse.Code != http.StatusNotFound {
		t.Fatalf("expected disabled org domain response 404, got %d", orgDisabledResponse.Code)
	}
	orgNotFoundResponse := httptest.NewRecorder()
	orgHandlerValue.handleDomainSurface(orgNotFoundResponse, httptest.NewRequest(http.MethodGet, "/orgs/acme/profile", nil), orgAccessContext{hasMember: true, member: model.OrganizationMember{Role: model.OrganizationRoleOwner}, organization: model.Organization{ID: 1}}, "profile")
	if orgNotFoundResponse.Code != http.StatusNotFound {
		t.Fatalf("expected org non-domain route 404, got %d", orgNotFoundResponse.Code)
	}
	orgListErrorResponse := httptest.NewRecorder()
	OrgHandler{routePrefix: "/orgs", domainService: failingDomainService}.handleDomainSurface(orgListErrorResponse, httptest.NewRequest(http.MethodGet, "/orgs/acme/domains", nil), orgAccessContext{hasMember: true, member: model.OrganizationMember{Role: model.OrganizationRoleOwner}, organization: model.Organization{ID: 1}}, "domains")
	if orgListErrorResponse.Code != http.StatusInternalServerError {
		t.Fatalf("expected org list error response 500, got %d", orgListErrorResponse.Code)
	}
	orgListMethodResponse := httptest.NewRecorder()
	orgHandlerValue.handleDomainSurface(orgListMethodResponse, httptest.NewRequest(http.MethodDelete, "/orgs/acme/domains", nil), orgAccessContext{hasMember: true, member: model.OrganizationMember{Role: model.OrganizationRoleOwner}, organization: model.Organization{ID: 1}}, "domains")
	if orgListMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected org list method rejection, got %d", orgListMethodResponse.Code)
	}
	orgListResponse := httptest.NewRecorder()
	orgHandlerValue.handleDomainSurface(orgListResponse, httptest.NewRequest(http.MethodGet, "/orgs/acme/domains", nil), orgAccessContext{hasMember: true, member: model.OrganizationMember{Role: model.OrganizationRoleOwner}, organization: model.Organization{ID: 1}}, "domains")
	if orgListResponse.Code != http.StatusOK {
		t.Fatalf("expected org list success, got %d", orgListResponse.Code)
	}
	orgCreateParseResponse := httptest.NewRecorder()
	orgHandlerValue.handleDomainSurface(orgCreateParseResponse, httptest.NewRequest(http.MethodPost, "/orgs/acme/domains", strings.NewReader("{")), orgAccessContext{hasMember: true, member: model.OrganizationMember{Role: model.OrganizationRoleOwner}, organization: model.Organization{ID: 1}}, "domains")
	if orgCreateParseResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected org create parse failure, got %d", orgCreateParseResponse.Code)
	}
	orgCreateErrorResponse := httptest.NewRecorder()
	OrgHandler{routePrefix: "/orgs", domainService: failingDomainService}.handleDomainSurface(orgCreateErrorResponse, httptest.NewRequest(http.MethodPost, "/orgs/acme/domains", strings.NewReader(`{"domain":"api.example.com"}`)), orgAccessContext{hasMember: true, member: model.OrganizationMember{Role: model.OrganizationRoleOwner}, organization: model.Organization{ID: 1}}, "domains")
	if orgCreateErrorResponse.Code != http.StatusInternalServerError {
		t.Fatalf("expected org create error response 500, got %d", orgCreateErrorResponse.Code)
	}
	orgDetailResponse := httptest.NewRecorder()
	orgHandlerValue.handleDomainSurface(orgDetailResponse, httptest.NewRequest(http.MethodGet, "/orgs/acme/domains/api.example.com", nil), orgAccessContext{hasMember: true, member: model.OrganizationMember{Role: model.OrganizationRoleOwner}, organization: model.Organization{ID: 1}}, "domains/api.example.com")
	if orgDetailResponse.Code != http.StatusOK {
		t.Fatalf("expected org detail success, got %d", orgDetailResponse.Code)
	}
	orgDetailErrorResponse := httptest.NewRecorder()
	OrgHandler{routePrefix: "/orgs", domainService: failingDomainService}.handleDomainSurface(orgDetailErrorResponse, httptest.NewRequest(http.MethodGet, "/orgs/acme/domains/api.example.com", nil), orgAccessContext{hasMember: true, member: model.OrganizationMember{Role: model.OrganizationRoleOwner}, organization: model.Organization{ID: 1}}, "domains/api.example.com")
	if orgDetailErrorResponse.Code != http.StatusInternalServerError {
		t.Fatalf("expected org detail error response 500, got %d", orgDetailErrorResponse.Code)
	}
	orgMethodResponse := httptest.NewRecorder()
	orgHandlerValue.handleDomainSurface(orgMethodResponse, httptest.NewRequest(http.MethodPut, "/orgs/acme/domains/api.example.com", nil), orgAccessContext{hasMember: true, member: model.OrganizationMember{Role: model.OrganizationRoleOwner}, organization: model.Organization{ID: 1}}, "domains/api.example.com")
	if orgMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected org domain method rejection, got %d", orgMethodResponse.Code)
	}
	orgDeleteResponse := httptest.NewRecorder()
	orgHandlerValue.handleDomainSurface(orgDeleteResponse, httptest.NewRequest(http.MethodDelete, "/orgs/acme/domains/api.example.com", nil), orgAccessContext{hasMember: true, member: model.OrganizationMember{Role: model.OrganizationRoleOwner}, organization: model.Organization{ID: 1}}, "domains/api.example.com")
	if orgDeleteResponse.Code != http.StatusOK {
		t.Fatalf("expected org delete success, got %d", orgDeleteResponse.Code)
	}
	orgDeleteErrorResponse := httptest.NewRecorder()
	OrgHandler{routePrefix: "/orgs", domainService: failingDomainService}.handleDomainSurface(orgDeleteErrorResponse, httptest.NewRequest(http.MethodDelete, "/orgs/acme/domains/api.example.com", nil), orgAccessContext{hasMember: true, member: model.OrganizationMember{Role: model.OrganizationRoleOwner}, organization: model.Organization{ID: 1}}, "domains/api.example.com")
	if orgDeleteErrorResponse.Code != http.StatusInternalServerError {
		t.Fatalf("expected org delete error response 500, got %d", orgDeleteErrorResponse.Code)
	}
	orgVerifyMethodResponse := httptest.NewRecorder()
	orgHandlerValue.handleDomainSurface(orgVerifyMethodResponse, httptest.NewRequest(http.MethodGet, "/orgs/acme/domains/api.example.com/verify", nil), orgAccessContext{hasMember: true, member: model.OrganizationMember{Role: model.OrganizationRoleOwner}, organization: model.Organization{ID: 1}}, "domains/api.example.com/verify")
	if orgVerifyMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected org verify method rejection, got %d", orgVerifyMethodResponse.Code)
	}
	orgVerifyErrorResponse := httptest.NewRecorder()
	OrgHandler{routePrefix: "/orgs", domainService: failingDomainService}.handleDomainSurface(orgVerifyErrorResponse, httptest.NewRequest(http.MethodPost, "/orgs/acme/domains/api.example.com/verify", nil), orgAccessContext{hasMember: true, member: model.OrganizationMember{Role: model.OrganizationRoleOwner}, organization: model.Organization{ID: 1}}, "domains/api.example.com/verify")
	if orgVerifyErrorResponse.Code != http.StatusInternalServerError {
		t.Fatalf("expected org verify error response 500, got %d", orgVerifyErrorResponse.Code)
	}
	orgSSLParseResponse := httptest.NewRecorder()
	orgHandlerValue.handleDomainSurface(orgSSLParseResponse, httptest.NewRequest(http.MethodPost, "/orgs/acme/domains/api.example.com/ssl", strings.NewReader("{")), orgAccessContext{hasMember: true, member: model.OrganizationMember{Role: model.OrganizationRoleOwner}, organization: model.Organization{ID: 1}}, "domains/api.example.com/ssl")
	if orgSSLParseResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected org ssl parse failure, got %d", orgSSLParseResponse.Code)
	}
	orgSSLMethodResponse := httptest.NewRecorder()
	orgHandlerValue.handleDomainSurface(orgSSLMethodResponse, httptest.NewRequest(http.MethodGet, "/orgs/acme/domains/api.example.com/ssl", nil), orgAccessContext{hasMember: true, member: model.OrganizationMember{Role: model.OrganizationRoleOwner}, organization: model.Organization{ID: 1}}, "domains/api.example.com/ssl")
	if orgSSLMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected org ssl method rejection, got %d", orgSSLMethodResponse.Code)
	}
	orgSSLErrorResponse := httptest.NewRecorder()
	OrgHandler{routePrefix: "/orgs", domainService: failingDomainService}.handleDomainSurface(orgSSLErrorResponse, httptest.NewRequest(http.MethodPost, "/orgs/acme/domains/api.example.com/ssl", strings.NewReader(`{}`)), orgAccessContext{hasMember: true, member: model.OrganizationMember{Role: model.OrganizationRoleOwner}, organization: model.Organization{ID: 1}}, "domains/api.example.com/ssl")
	if orgSSLErrorResponse.Code != http.StatusInternalServerError {
		t.Fatalf("expected org ssl error response 500, got %d", orgSSLErrorResponse.Code)
	}

	adminDisabledResponse := httptest.NewRecorder()
	AdminHandler{domainService: service.NewDomainService(store.NewMemoryStore(), model.DomainConstraints{}, nil, nil)}.handleDomainSurface(adminDisabledResponse, httptest.NewRequest(http.MethodGet, "/admin/server/domains", nil), "server/domains")
	if adminDisabledResponse.Code != http.StatusNotFound {
		t.Fatalf("expected disabled admin domain response 404, got %d", adminDisabledResponse.Code)
	}
	adminWriteNotFoundResponse := httptest.NewRecorder()
	writeAdminDomainSurface(adminWriteNotFoundResponse, httptest.NewRequest(http.MethodGet, "/admin/server/users", nil), "server/users", baseDomainService)
	if adminWriteNotFoundResponse.Code != http.StatusNotFound {
		t.Fatalf("expected non-domain admin route 404, got %d", adminWriteNotFoundResponse.Code)
	}
	adminListMethodResponse := httptest.NewRecorder()
	writeAdminDomainSurface(adminListMethodResponse, httptest.NewRequest(http.MethodPost, "/admin/server/domains", nil), "server/domains", baseDomainService)
	if adminListMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected admin list method rejection, got %d", adminListMethodResponse.Code)
	}
	adminListErrorResponse := httptest.NewRecorder()
	writeAdminDomainSurface(adminListErrorResponse, httptest.NewRequest(http.MethodGet, "/admin/server/domains", nil), "server/domains", failingDomainService)
	if adminListErrorResponse.Code != http.StatusInternalServerError {
		t.Fatalf("expected admin list error response 500, got %d", adminListErrorResponse.Code)
	}
	adminDetailResponse := httptest.NewRecorder()
	writeAdminDomainSurface(adminDetailResponse, httptest.NewRequest(http.MethodGet, "/admin/server/domains/api.example.com", nil), "server/domains/api.example.com", baseDomainService)
	if adminDetailResponse.Code != http.StatusOK {
		t.Fatalf("expected admin detail success, got %d", adminDetailResponse.Code)
	}
	adminDetailErrorResponse := httptest.NewRecorder()
	writeAdminDomainSurface(adminDetailErrorResponse, httptest.NewRequest(http.MethodGet, "/admin/server/domains/api.example.com", nil), "server/domains/api.example.com", failingDomainService)
	if adminDetailErrorResponse.Code != http.StatusInternalServerError {
		t.Fatalf("expected admin detail error response 500, got %d", adminDetailErrorResponse.Code)
	}
	adminDeleteErrorResponse := httptest.NewRecorder()
	writeAdminDomainSurface(adminDeleteErrorResponse, httptest.NewRequest(http.MethodDelete, "/admin/server/domains/api.example.com", nil), "server/domains/api.example.com", failingDomainService)
	if adminDeleteErrorResponse.Code != http.StatusInternalServerError {
		t.Fatalf("expected admin delete error response 500, got %d", adminDeleteErrorResponse.Code)
	}
	adminDeleteResponse := httptest.NewRecorder()
	writeAdminDomainSurface(adminDeleteResponse, httptest.NewRequest(http.MethodDelete, "/admin/server/domains/api.example.com", nil), "server/domains/api.example.com", baseDomainService)
	if adminDeleteResponse.Code != http.StatusOK {
		t.Fatalf("expected admin delete success, got %d", adminDeleteResponse.Code)
	}
	adminDetailMethodResponse := httptest.NewRecorder()
	writeAdminDomainSurface(adminDetailMethodResponse, httptest.NewRequest(http.MethodPut, "/admin/server/domains/api.example.com", nil), "server/domains/api.example.com", baseDomainService)
	if adminDetailMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected admin detail method rejection, got %d", adminDetailMethodResponse.Code)
	}
	adminSuspendParseResponse := httptest.NewRecorder()
	writeAdminDomainSurface(adminSuspendParseResponse, httptest.NewRequest(http.MethodPost, "/admin/server/domains/api.example.com/suspend", strings.NewReader("{")), "server/domains/api.example.com/suspend", baseDomainService)
	if adminSuspendParseResponse.Code != http.StatusBadRequest {
		t.Fatalf("expected admin suspend parse failure, got %d", adminSuspendParseResponse.Code)
	}
	adminSuspendErrorResponse := httptest.NewRecorder()
	writeAdminDomainSurface(adminSuspendErrorResponse, httptest.NewRequest(http.MethodPost, "/admin/server/domains/api.example.com/suspend", strings.NewReader(`{"reason":"abuse"}`)), "server/domains/api.example.com/suspend", failingDomainService)
	if adminSuspendErrorResponse.Code != http.StatusInternalServerError {
		t.Fatalf("expected admin suspend error response 500, got %d", adminSuspendErrorResponse.Code)
	}
	adminSuspendSuccessResponse := httptest.NewRecorder()
	writeAdminDomainSurface(adminSuspendSuccessResponse, httptest.NewRequest(http.MethodPost, "/admin/server/domains/api.example.com/suspend", strings.NewReader(`{"reason":"abuse"}`)), "server/domains/api.example.com/suspend", baseDomainService)
	if adminSuspendSuccessResponse.Code != http.StatusOK {
		t.Fatalf("expected admin suspend success, got %d", adminSuspendSuccessResponse.Code)
	}
	adminSuspendMethodResponse := httptest.NewRecorder()
	writeAdminDomainSurface(adminSuspendMethodResponse, httptest.NewRequest(http.MethodGet, "/admin/server/domains/api.example.com/suspend", nil), "server/domains/api.example.com/suspend", baseDomainService)
	if adminSuspendMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected admin suspend method rejection, got %d", adminSuspendMethodResponse.Code)
	}
	adminUnsuspendResponse := httptest.NewRecorder()
	writeAdminDomainSurface(adminUnsuspendResponse, httptest.NewRequest(http.MethodPost, "/admin/server/domains/api.example.com/unsuspend", nil), "server/domains/api.example.com/unsuspend", baseDomainService)
	if adminUnsuspendResponse.Code != http.StatusOK {
		t.Fatalf("expected admin unsuspend success, got %d", adminUnsuspendResponse.Code)
	}
	adminUnsuspendMethodResponse := httptest.NewRecorder()
	writeAdminDomainSurface(adminUnsuspendMethodResponse, httptest.NewRequest(http.MethodGet, "/admin/server/domains/api.example.com/unsuspend", nil), "server/domains/api.example.com/unsuspend", baseDomainService)
	if adminUnsuspendMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected admin unsuspend method rejection, got %d", adminUnsuspendMethodResponse.Code)
	}
	adminUnsuspendErrorResponse := httptest.NewRecorder()
	writeAdminDomainSurface(adminUnsuspendErrorResponse, httptest.NewRequest(http.MethodPost, "/admin/server/domains/api.example.com/unsuspend", nil), "server/domains/api.example.com/unsuspend", failingDomainService)
	if adminUnsuspendErrorResponse.Code != http.StatusInternalServerError {
		t.Fatalf("expected admin unsuspend error response 500, got %d", adminUnsuspendErrorResponse.Code)
	}
	adminRenewResponse := httptest.NewRecorder()
	writeAdminDomainSurface(adminRenewResponse, httptest.NewRequest(http.MethodPost, "/admin/server/domains/api.example.com/ssl/renew", nil), "server/domains/api.example.com/ssl/renew", baseDomainService)
	if adminRenewResponse.Code != http.StatusOK {
		t.Fatalf("expected admin renew success, got %d", adminRenewResponse.Code)
	}
	adminRenewMethodResponse := httptest.NewRecorder()
	writeAdminDomainSurface(adminRenewMethodResponse, httptest.NewRequest(http.MethodGet, "/admin/server/domains/api.example.com/ssl/renew", nil), "server/domains/api.example.com/ssl/renew", baseDomainService)
	if adminRenewMethodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected admin renew method rejection, got %d", adminRenewMethodResponse.Code)
	}
	adminRenewErrorResponse := httptest.NewRecorder()
	writeAdminDomainSurface(adminRenewErrorResponse, httptest.NewRequest(http.MethodPost, "/admin/server/domains/api.example.com/ssl/renew", nil), "server/domains/api.example.com/ssl/renew", failingDomainService)
	if adminRenewErrorResponse.Code != http.StatusInternalServerError {
		t.Fatalf("expected admin renew error response 500, got %d", adminRenewErrorResponse.Code)
	}
	adminUnknownActionResponse := httptest.NewRecorder()
	writeAdminDomainSurface(adminUnknownActionResponse, httptest.NewRequest(http.MethodGet, "/admin/server/domains/api.example.com/bad", nil), "server/domains/api.example.com/bad", baseDomainService)
	if adminUnknownActionResponse.Code != http.StatusNotFound {
		t.Fatalf("expected admin unknown action 404, got %d", adminUnknownActionResponse.Code)
	}
	apiAdminDisabledResponse := httptest.NewRecorder()
	APIAdminHandler{domainService: service.NewDomainService(store.NewMemoryStore(), model.DomainConstraints{}, nil, nil)}.handleDomainSurface(apiAdminDisabledResponse, httptest.NewRequest(http.MethodGet, "/api/v1/admin/server/domains", nil), "server/domains")
	if apiAdminDisabledResponse.Code != http.StatusNotFound {
		t.Fatalf("expected api admin disabled response 404, got %d", apiAdminDisabledResponse.Code)
	}
	apiAdminUnauthorizedResponse := httptest.NewRecorder()
	APIAdminHandler{domainService: baseDomainService}.handleDomainSurface(apiAdminUnauthorizedResponse, httptest.NewRequest(http.MethodGet, "/api/v1/admin/server/domains", nil), "server/domains")
	if apiAdminUnauthorizedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected api admin unauthorized response 401, got %d", apiAdminUnauthorizedResponse.Code)
	}
}
