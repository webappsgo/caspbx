package service

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/casapps/caspbx/src/server/model"
	"github.com/casapps/caspbx/src/server/store"
)

type fakeDomainResolver struct {
	ips    map[string][]net.IP
	cnames map[string]string
	errs   map[string]error
}

type failingDomainStore struct {
	domain      model.CustomDomain
	listError   error
	findError   error
	saveError   error
	deleteError error
}

func (domainStore failingDomainStore) SaveCustomDomain(context.Context, model.CustomDomain) (model.CustomDomain, error) {
	return domainStore.domain, domainStore.saveError
}

func (domainStore failingDomainStore) FindCustomDomainByID(context.Context, int64) (model.CustomDomain, error) {
	if domainStore.findError != nil {
		return model.CustomDomain{}, domainStore.findError
	}
	return domainStore.domain, nil
}

func (domainStore failingDomainStore) FindCustomDomainByDomain(context.Context, string) (model.CustomDomain, error) {
	if domainStore.findError != nil {
		return model.CustomDomain{}, domainStore.findError
	}
	return domainStore.domain, nil
}

func (domainStore failingDomainStore) FindDomainByHost(context.Context, string) (model.CustomDomain, error) {
	return domainStore.FindCustomDomainByDomain(context.Background(), "")
}

func (domainStore failingDomainStore) ListCustomDomains(context.Context) ([]model.CustomDomain, error) {
	if domainStore.listError != nil {
		return nil, domainStore.listError
	}
	return []model.CustomDomain{domainStore.domain}, nil
}

func (domainStore failingDomainStore) ListCustomDomainsByOwner(context.Context, model.DomainOwnerType, int64) ([]model.CustomDomain, error) {
	if domainStore.listError != nil {
		return nil, domainStore.listError
	}
	return []model.CustomDomain{domainStore.domain}, nil
}

func (domainStore failingDomainStore) DeleteCustomDomainByID(context.Context, int64) error {
	return domainStore.deleteError
}

func (resolver fakeDomainResolver) LookupIP(host string) ([]net.IP, error) {
	if err := resolver.errs[host]; err != nil {
		return nil, err
	}
	if ips, found := resolver.ips[host]; found {
		return ips, nil
	}
	return nil, errors.New("missing host")
}

func (resolver fakeDomainResolver) LookupCNAME(host string) (string, error) {
	if err := resolver.errs["cname:"+host]; err != nil {
		return "", err
	}
	if cname, found := resolver.cnames[host]; found {
		return cname, nil
	}
	return "", errors.New("missing cname")
}

func TestDomainServiceLifecycleAndHelpers(t *testing.T) {
	memoryStore := store.NewMemoryStore()
	now := time.Unix(1_700_100_000, 0).UTC()
	domainService := NewDomainService(memoryStore, model.DomainConstraints{
		Enabled:           true,
		MaxDomainsPerUser: 1,
		MaxDomainsPerOrg:  2,
		RequireSSL:        true,
		AllowApex:         true,
		AllowSubdomain:    true,
		AllowWildcard:     false,
		VerificationTTL:   24 * time.Hour,
		SSLRenewalDays:    7,
		Reserved:          []string{"localhost", "*.example"},
		BlockedPatterns:   []string{`.*\.gov$`},
	}, []string{"custom.example.com:443"}, []net.IP{net.ParseIP("203.0.113.50")}, WithDomainClock(func() time.Time { return now }))

	if !domainService.Enabled() {
		t.Fatalf("expected domain service to be enabled")
	}
	instructions := domainService.DNSInstructions()
	if instructions.Target != "custom.example.com" || len(instructions.TargetIPs) != 1 {
		t.Fatalf("unexpected dns instructions %+v", instructions)
	}

	userDomain, createError := domainService.CreateUserDomain(context.Background(), 11, "Api.Example.com")
	if createError != nil {
		t.Fatalf("create user domain: %v", createError)
	}
	if userDomain.Domain != "api.example.com" || userDomain.UserID != 11 || userDomain.Status != model.DomainStatusPending {
		t.Fatalf("unexpected user domain %+v", userDomain)
	}

	if _, duplicateError := domainService.CreateUserDomain(context.Background(), 11, "api.example.com"); duplicateError == nil || duplicateError.(*DomainError).Code != "DOMAIN_EXISTS" {
		t.Fatalf("expected duplicate domain error, got %v", duplicateError)
	}
	if _, limitError := domainService.CreateUserDomain(context.Background(), 11, "voice.example.com"); limitError == nil || limitError.(*DomainError).Code != "DOMAIN_LIMIT" {
		t.Fatalf("expected domain limit error, got %v", limitError)
	}
	if _, invalidError := domainService.CreateUserDomain(context.Background(), 11, "api.example"); invalidError == nil || invalidError.(*DomainError).Code != "DOMAIN_RESERVED" {
		t.Fatalf("expected reserved domain error, got %v", invalidError)
	}

	orgDomainA, createError := domainService.CreateOrganizationDomain(context.Background(), 21, "pbx.example.com")
	if createError != nil {
		t.Fatalf("create org domain: %v", createError)
	}
	orgDomainB, createError := domainService.CreateOrganizationDomain(context.Background(), 21, "fax.example.com")
	if createError != nil {
		t.Fatalf("create second org domain: %v", createError)
	}

	userDomains, listError := domainService.ListUserDomains(context.Background(), 11)
	if listError != nil || len(userDomains) != 1 {
		t.Fatalf("expected one user domain, got %v / %d", listError, len(userDomains))
	}
	orgDomains, listError := domainService.ListOrganizationDomains(context.Background(), 21)
	if listError != nil || len(orgDomains) != 2 || orgDomains[0].Domain != "fax.example.com" {
		t.Fatalf("expected sorted org domains, got %v / %+v", listError, orgDomains)
	}
	if allDomains, listError := domainService.ListDomains(context.Background()); listError != nil || len(allDomains) != 3 || allDomains[0].Domain != "api.example.com" {
		t.Fatalf("expected all domain list, got %v / %+v", listError, allDomains)
	}

	if foundDomain, lookupError := domainService.GetUserDomain(context.Background(), 11, "api.example.com"); lookupError != nil || foundDomain.ID != userDomain.ID {
		t.Fatalf("expected user domain lookup, got %v / %+v", lookupError, foundDomain)
	}
	if foundDomain, lookupError := domainService.GetOrganizationDomain(context.Background(), 21, "pbx.example.com"); lookupError != nil || foundDomain.ID != orgDomainA.ID {
		t.Fatalf("expected org domain lookup, got %v / %+v", lookupError, foundDomain)
	}
	if _, lookupError := domainService.GetUserDomain(context.Background(), 11, "pbx.example.com"); lookupError == nil || lookupError.(*DomainError).Code != "DOMAIN_NOT_FOUND" {
		t.Fatalf("expected wrong-owner lookup failure, got %v", lookupError)
	}
	if _, sslError := domainService.configureSSL(context.Background(), model.DomainOwnerTypeUser, 999, "api.example.com", model.SSLChallengeAuto, ""); sslError == nil || sslError.(*DomainError).Code != "DOMAIN_NOT_FOUND" {
		t.Fatalf("expected wrong-owner ssl failure, got %v", sslError)
	}
	if _, verifyError := domainService.verifyOwnedDomain(context.Background(), model.DomainOwnerTypeUser, 999, "api.example.com"); verifyError == nil || verifyError.(*DomainError).Code != "DOMAIN_NOT_FOUND" {
		t.Fatalf("expected wrong-owner verify failure, got %v", verifyError)
	}
	if deleteError := domainService.deleteOwnedDomain(context.Background(), model.DomainOwnerTypeUser, 999, "api.example.com"); deleteError == nil || deleteError.(*DomainError).Code != "DOMAIN_NOT_FOUND" {
		t.Fatalf("expected wrong-owner delete failure, got %v", deleteError)
	}
	if _, lookupError := domainService.GetDomain(context.Background(), "missing.example.com"); lookupError == nil || lookupError.(*DomainError).Code != "DOMAIN_NOT_FOUND" {
		t.Fatalf("expected missing global domain lookup failure, got %v", lookupError)
	}

	suspendedDomain, suspendError := domainService.SuspendDomain(context.Background(), "pbx.example.com", "abuse")
	if suspendError != nil || suspendedDomain.Status != model.DomainStatusSuspended || suspendedDomain.SuspensionReason != "abuse" {
		t.Fatalf("expected suspended domain, got %v / %+v", suspendError, suspendedDomain)
	}
	unsuspendedDomain, unsuspendError := domainService.UnsuspendDomain(context.Background(), "pbx.example.com")
	if unsuspendError != nil || unsuspendedDomain.Status != model.DomainStatusPending || unsuspendedDomain.SuspensionReason != "" {
		t.Fatalf("expected unsuspended pending domain, got %v / %+v", unsuspendError, unsuspendedDomain)
	}

	if deleteError := domainService.DeleteOrganizationDomain(context.Background(), 21, orgDomainB.Domain); deleteError != nil {
		t.Fatalf("delete org domain: %v", deleteError)
	}
	if deleteError := domainService.DeleteUserDomain(context.Background(), 11, userDomain.Domain); deleteError != nil {
		t.Fatalf("delete user domain: %v", deleteError)
	}
	if _, lookupError := memoryStore.FindCustomDomainByDomain(context.Background(), userDomain.Domain); lookupError != store.ErrNotFound {
		t.Fatalf("expected deleted user domain to be removed, got %v", lookupError)
	}
}

func TestDomainServiceVerificationAndSSL(t *testing.T) {
	memoryStore := store.NewMemoryStore()
	now := time.Unix(1_700_200_000, 0).UTC()
	resolver := fakeDomainResolver{
		ips: map[string][]net.IP{
			"match.example.com":       {net.ParseIP("203.0.113.50")},
			"mismatch.example.com":    {net.ParseIP("198.51.100.10")},
			"org-success.example.com": {net.ParseIP("203.0.113.50")},
		},
		cnames: map[string]string{
			"cname.example.com": "custom.example.com.",
		},
		errs: map[string]error{
			"lookup-fail.example.com": errors.New("lookup failed"),
		},
	}
	domainService := NewDomainService(memoryStore, model.DomainConstraints{
		Enabled:           true,
		MaxDomainsPerUser: 5,
		MaxDomainsPerOrg:  5,
		RequireSSL:        true,
		AllowApex:         true,
		AllowSubdomain:    true,
		AllowWildcard:     false,
		VerificationTTL:   24 * time.Hour,
		SSLRenewalDays:    7,
		Reserved:          []string{"localhost"},
		BlockedPatterns:   []string{`.*\.gov$`},
	}, []string{"custom.example.com"}, []net.IP{net.ParseIP("203.0.113.50")}, WithDomainClock(func() time.Time { return now }), WithDomainResolver(resolver), WithDomainChallengeAvailability(false, true))

	if domainError := (*DomainError)(nil).Error(); domainError != "" {
		t.Fatalf("expected nil domain error string, got %q", domainError)
	}

	matchingDomain, _ := domainService.CreateUserDomain(context.Background(), 10, "match.example.com")
	verifyResult, verifyError := domainService.VerifyUserDomain(context.Background(), 10, matchingDomain.Domain)
	if verifyError != nil || verifyResult.Domain.VerificationStatus != model.VerificationStatusVerified || verifyResult.ResolvedTo != "203.0.113.50" {
		t.Fatalf("expected successful verification, got %v / %+v", verifyError, verifyResult)
	}
	sslDomain, sslError := domainService.ConfigureUserSSL(context.Background(), 10, matchingDomain.Domain, model.SSLChallengeAuto, "")
	if sslError != nil || sslDomain.SSLChallenge != model.SSLChallengeHTTP01 || sslDomain.SSLStatus != model.SSLStatusPending {
		t.Fatalf("expected ssl configuration success, got %v / %+v", sslError, sslDomain)
	}
	renewedDomain, renewError := domainService.RenewDomainSSL(context.Background(), matchingDomain.Domain)
	if renewError != nil || renewedDomain.SSLStatus != model.SSLStatusPending {
		t.Fatalf("expected ssl renewal success, got %v / %+v", renewError, renewedDomain)
	}

	cnameDomain, _ := domainService.CreateUserDomain(context.Background(), 12, "cname.example.com")
	cnameResult, cnameError := domainService.VerifyUserDomain(context.Background(), 12, cnameDomain.Domain)
	if cnameError != nil || cnameResult.ResolvedTo != "custom.example.com" {
		t.Fatalf("expected cname verification success, got %v / %+v", cnameError, cnameResult)
	}

	mismatchDomain, _ := domainService.CreateUserDomain(context.Background(), 13, "mismatch.example.com")
	if _, mismatchError := domainService.VerifyUserDomain(context.Background(), 13, mismatchDomain.Domain); mismatchError == nil || mismatchError.(*DomainError).Code != "DNS_MISMATCH" {
		t.Fatalf("expected mismatch verification error, got %v", mismatchError)
	}

	lookupFailDomain, _ := domainService.CreateUserDomain(context.Background(), 14, "lookup-fail.example.com")
	if _, lookupError := domainService.VerifyUserDomain(context.Background(), 14, lookupFailDomain.Domain); lookupError == nil || lookupError.(*DomainError).Code != "DNS_LOOKUP_FAILED" {
		t.Fatalf("expected lookup failure error, got %v", lookupError)
	}

	unverifiedDomain, _ := domainService.CreateOrganizationDomain(context.Background(), 20, "org.example.com")
	if _, sslError := domainService.ConfigureOrganizationSSL(context.Background(), 20, unverifiedDomain.Domain, model.SSLChallengeAuto, ""); sslError == nil || sslError.(*DomainError).Code != "DOMAIN_NOT_VERIFIED" {
		t.Fatalf("expected ssl verify prerequisite error, got %v", sslError)
	}

	orgSuccessDomain, _ := domainService.CreateOrganizationDomain(context.Background(), 20, "org-success.example.com")
	if _, verifyError := domainService.VerifyOrganizationDomain(context.Background(), 20, orgSuccessDomain.Domain); verifyError != nil {
		t.Fatalf("expected org verification success, got %v", verifyError)
	}
	if _, sslError := domainService.ConfigureOrganizationSSL(context.Background(), 20, orgSuccessDomain.Domain, model.SSLChallengeDNS01, ""); sslError == nil || sslError.(*DomainError).Code != "SSL_PROVIDER_INVALID" {
		t.Fatalf("expected dns provider validation error, got %v", sslError)
	}
	orgSSLDomain, sslError := domainService.ConfigureOrganizationSSL(context.Background(), 20, orgSuccessDomain.Domain, model.SSLChallengeDNS01, "cloudflare")
	if sslError != nil || orgSSLDomain.SSLProvider != "cloudflare" || !orgSSLDomain.SSLEnabled {
		t.Fatalf("expected org ssl success, got %v / %+v", sslError, orgSSLDomain)
	}

	if _, renewError := domainService.RenewDomainSSL(context.Background(), lookupFailDomain.Domain); renewError == nil || renewError.(*DomainError).Code != "DOMAIN_NOT_VERIFIED" {
		t.Fatalf("expected missing ssl renewal failure, got %v", renewError)
	}
	if deleteError := domainService.DeleteDomain(context.Background(), orgSuccessDomain.Domain); deleteError != nil {
		t.Fatalf("expected admin delete success, got %v", deleteError)
	}
}

func TestDomainServiceInternalsAndTenantResolver(t *testing.T) {
	memoryStore := store.NewMemoryStore()
	disabledService := NewDomainService(memoryStore, model.DomainConstraints{}, nil, nil)
	if _, createError := disabledService.CreateUserDomain(context.Background(), 1, "api.example.com"); createError == nil || createError.(*DomainError).Code != "DOMAIN_NOT_FOUND" {
		t.Fatalf("expected disabled feature error, got %v", createError)
	}

	domains := []model.CustomDomain{{Domain: "z.example.com"}, {Domain: "a.example.com"}}
	domains = append(domains, model.CustomDomain{Domain: "a.example.com"})
	sortCustomDomains(domains)
	if domains[0].Domain != "a.example.com" {
		t.Fatalf("expected sorted custom domains, got %+v", domains)
	}
	if disabledService.limitForOwner(model.DomainOwnerTypeUser) != 0 || disabledService.limitForOwner(model.DomainOwnerTypeOrg) != 0 {
		t.Fatalf("expected zero limits on disabled service")
	}

	savedDomain, saveError := memoryStore.SaveCustomDomain(context.Background(), model.CustomDomain{
		Domain:             "tenant.example.com",
		Status:             model.DomainStatusActive,
		VerificationStatus: model.VerificationStatusVerified,
		UserID:             77,
	})
	if saveError != nil {
		t.Fatalf("save tenant domain: %v", saveError)
	}
	resolver, resolverError := NewTenantResolverFromStore(context.Background(), []string{"pbx.example.com"}, memoryStore)
	if resolverError != nil {
		t.Fatalf("new tenant resolver from store: %v", resolverError)
	}
	tenantContext, resolveError := resolver.Resolve(savedDomain.Domain)
	if resolveError != nil || tenantContext.UserID != 77 {
		t.Fatalf("expected tenant resolver from store to return user binding, got %v / %+v", resolveError, tenantContext)
	}

	if ips, lookupError := (netDomainResolver{}).LookupIP("localhost"); lookupError != nil || len(ips) == 0 {
		t.Fatalf("expected localhost lookup to succeed, got %v / %+v", lookupError, ips)
	}
	(netDomainResolver{}).LookupCNAME("localhost")
}

func TestDomainServiceErrorAndFallbackBranches(t *testing.T) {
	domainService := NewDomainService(store.NewMemoryStore(), model.DomainConstraints{
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
	}, []string{"", " custom.example.com "}, []net.IP{nil})
	if instructions := domainService.DNSInstructions(); instructions.Target != "custom.example.com" || len(instructions.TargetIPs) != 0 {
		t.Fatalf("unexpected dns instruction fallback %+v", instructions)
	}
	if (&DomainError{Code: "DOMAIN_EXISTS", Message: "exists"}).Error() != "exists" {
		t.Fatalf("expected non-nil domain error string")
	}

	failingStore := failingDomainStore{
		domain:    model.CustomDomain{ID: 1, Domain: "api.example.com", OwnerType: model.DomainOwnerTypeUser, OwnerID: 1},
		listError: errors.New("list failed"),
		findError: errors.New("find failed"),
	}
	failingService := NewDomainService(failingStore, model.DomainConstraints{Enabled: true, AllowApex: true, AllowSubdomain: true, VerificationTTL: time.Hour, SSLRenewalDays: 7}, nil, nil)
	if _, listError := failingService.ListDomains(context.Background()); listError == nil {
		t.Fatalf("expected list domains error")
	}
	if _, createError := failingService.CreateUserDomain(context.Background(), 1, "api.example.com"); createError == nil {
		t.Fatalf("expected create domain error from failing store")
	}
	createLookupFailService := NewDomainService(failingDomainStore{
		findError: errors.New("lookup failed"),
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
	}, nil, nil)
	if _, createError := createLookupFailService.createDomain(context.Background(), model.DomainOwnerTypeUser, 1, "api.example.com"); createError == nil || createError.Error() != "lookup failed" {
		t.Fatalf("expected create lookup failure passthrough, got %v", createError)
	}
	saveCreateService := NewDomainService(failingDomainStore{
		findError: store.ErrNotFound,
		saveError: errors.New("save failed"),
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
	}, nil, nil)
	if _, createError := saveCreateService.createDomain(context.Background(), model.DomainOwnerType("other"), 1, "save.example.com"); createError == nil || createError.Error() != "save failed" {
		t.Fatalf("expected create save failure, got %v", createError)
	}
	validationService := NewDomainService(store.NewMemoryStore(), model.DomainConstraints{
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
	}, nil, nil)
	if _, createError := validationService.createDomain(context.Background(), model.DomainOwnerType("other"), 1, "agency.gov"); createError == nil || createError.(*DomainError).Code != "DOMAIN_RESERVED" {
		t.Fatalf("expected blocked pattern create error, got %v", createError)
	}
	if _, lookupError := failingService.GetDomain(context.Background(), "api.example.com"); lookupError == nil || lookupError.Error() != "find failed" {
		t.Fatalf("expected direct lookup error passthrough, got %v", lookupError)
	}
	if _, lookupError := failingService.getOwnedDomain(context.Background(), model.DomainOwnerTypeUser, 1, "api.example.com"); lookupError == nil || lookupError.Error() != "find failed" {
		t.Fatalf("expected owned lookup passthrough error, got %v", lookupError)
	}
	if _, suspendError := failingService.SuspendDomain(context.Background(), "api.example.com", "reason"); suspendError == nil {
		t.Fatalf("expected suspend lookup error")
	}
	if _, unsuspendError := failingService.UnsuspendDomain(context.Background(), "api.example.com"); unsuspendError == nil {
		t.Fatalf("expected unsuspend lookup error")
	}
	if _, renewError := failingService.RenewDomainSSL(context.Background(), "api.example.com"); renewError == nil {
		t.Fatalf("expected renew lookup error")
	}
	if deleteError := failingService.DeleteDomain(context.Background(), "api.example.com"); deleteError == nil {
		t.Fatalf("expected delete lookup error")
	}
	if _, resolverError := NewTenantResolverFromStore(context.Background(), nil, failingStore); resolverError == nil {
		t.Fatalf("expected tenant resolver store list error")
	}

	verifiedStore := store.NewMemoryStore()
	verifiedService := NewDomainService(verifiedStore, model.DomainConstraints{
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
	}, nil, nil, WithDomainChallengeAvailability(false, false))
	verifiedDomain, _ := verifiedStore.SaveCustomDomain(context.Background(), model.CustomDomain{
		Domain:             "verified.example.com",
		OwnerType:          model.DomainOwnerTypeUser,
		OwnerID:            1,
		Status:             model.DomainStatusSuspended,
		VerificationStatus: model.VerificationStatusVerified,
		SSLEnabled:         true,
	})
	verifiedDomain.SSLChallenge = ""
	if _, saveError := verifiedStore.SaveCustomDomain(context.Background(), verifiedDomain); saveError != nil {
		t.Fatalf("save verified domain: %v", saveError)
	}
	unsuspendedDomain, unsuspendError := verifiedService.UnsuspendDomain(context.Background(), verifiedDomain.Domain)
	if unsuspendError != nil || unsuspendedDomain.Status != model.DomainStatusActive {
		t.Fatalf("expected verified unsuspend to activate domain, got %v / %+v", unsuspendError, unsuspendedDomain)
	}
	renewedDomain, renewError := verifiedService.RenewDomainSSL(context.Background(), verifiedDomain.Domain)
	if renewError != nil || renewedDomain.SSLChallenge != model.SSLChallengeDNS01 {
		t.Fatalf("expected renewal to choose dns challenge fallback, got %v / %+v", renewError, renewedDomain)
	}

	saveFailStore := failingDomainStore{
		domain: model.CustomDomain{
			ID:                 2,
			Domain:             "savefail.example.com",
			OwnerType:          model.DomainOwnerTypeUser,
			OwnerID:            1,
			VerificationStatus: model.VerificationStatusPending,
		},
		saveError: errors.New("save failed"),
	}
	saveFailService := NewDomainService(saveFailStore, model.DomainConstraints{
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
	}, []string{"custom.example.com"}, []net.IP{net.ParseIP("203.0.113.50")}, WithDomainResolver(fakeDomainResolver{
		ips: map[string][]net.IP{"savefail.example.com": {net.ParseIP("203.0.113.50")}},
	}))
	if _, verifyError := saveFailService.verifyDomain(context.Background(), saveFailStore.domain); verifyError == nil || verifyError.Error() != "save failed" {
		t.Fatalf("expected verify save failure, got %v", verifyError)
	}
	lookupFailWithSaveStore := failingDomainStore{
		domain: model.CustomDomain{
			ID:                 3,
			Domain:             "lookupsave.example.com",
			OwnerType:          model.DomainOwnerTypeUser,
			OwnerID:            1,
			VerificationStatus: model.VerificationStatusPending,
		},
		saveError: errors.New("save failed"),
	}
	lookupFailWithSaveService := NewDomainService(lookupFailWithSaveStore, model.DomainConstraints{
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
	}, nil, nil, WithDomainResolver(fakeDomainResolver{
		errs: map[string]error{"lookupsave.example.com": errors.New("lookup failed")},
	}))
	if _, verifyError := lookupFailWithSaveService.verifyDomain(context.Background(), lookupFailWithSaveStore.domain); verifyError == nil || verifyError.Error() != "save failed" {
		t.Fatalf("expected lookup failure save error, got %v", verifyError)
	}
	lookupOnlyFailService := NewDomainService(saveFailStore, model.DomainConstraints{
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
	}, nil, nil, WithDomainResolver(fakeDomainResolver{
		errs: map[string]error{"savefail.example.com": errors.New("lookup failed")},
	}))
	if _, verifyError := lookupOnlyFailService.verifyDomain(context.Background(), saveFailStore.domain); verifyError == nil || verifyError.Error() != "save failed" {
		t.Fatalf("expected lookup-only verify save failure, got %v", verifyError)
	}
	lookupOnlyStore := store.NewMemoryStore()
	lookupOnlyService := NewDomainService(lookupOnlyStore, model.DomainConstraints{
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
	}, nil, nil, WithDomainResolver(fakeDomainResolver{
		errs: map[string]error{"lookuponly.example.com": errors.New("lookup failed")},
	}))
	lookupOnlyDomain, saveError := lookupOnlyStore.SaveCustomDomain(context.Background(), model.CustomDomain{
		Domain:             "lookuponly.example.com",
		OwnerType:          model.DomainOwnerTypeUser,
		OwnerID:            1,
		VerificationStatus: model.VerificationStatusPending,
	})
	if saveError != nil {
		t.Fatalf("save lookup-only domain: %v", saveError)
	}
	verifyResult, verifyError := lookupOnlyService.verifyDomain(context.Background(), lookupOnlyDomain)
	if verifyError == nil || verifyError.(*DomainError).Code != "DNS_LOOKUP_FAILED" || verifyResult.Domain.VerificationError != "DNS_LOOKUP_FAILED" {
		t.Fatalf("expected lookup-only verify failure result, got %v / %+v", verifyError, verifyResult)
	}
	mismatchSaveFailStore := failingDomainStore{
		domain: model.CustomDomain{
			ID:                 4,
			Domain:             "mismatchsave.example.com",
			OwnerType:          model.DomainOwnerTypeUser,
			OwnerID:            1,
			VerificationStatus: model.VerificationStatusPending,
		},
		saveError: errors.New("save failed"),
	}
	mismatchSaveFailService := NewDomainService(mismatchSaveFailStore, model.DomainConstraints{
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
	}, nil, []net.IP{net.ParseIP("203.0.113.50")}, WithDomainResolver(fakeDomainResolver{
		ips: map[string][]net.IP{"mismatchsave.example.com": {net.ParseIP("198.51.100.10")}},
	}))
	if _, verifyError := mismatchSaveFailService.verifyDomain(context.Background(), mismatchSaveFailStore.domain); verifyError == nil || verifyError.Error() != "save failed" {
		t.Fatalf("expected mismatch verify save failure, got %v", verifyError)
	}
}
