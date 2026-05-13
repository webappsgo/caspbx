package service

import (
	"context"
	"errors"
	"net"
	"slices"
	"strings"
	"time"

	"github.com/casapps/caspbx/src/server/model"
	"github.com/casapps/caspbx/src/server/store"
)

type DomainResolver interface {
	LookupIP(host string) ([]net.IP, error)
	LookupCNAME(host string) (string, error)
}

type DomainService struct {
	store            store.DomainStore
	constraints      model.DomainConstraints
	platformHosts    []string
	publicIPs        []net.IP
	tlsALPNAvailable bool
	httpAvailable    bool
	resolver         DomainResolver
	now              func() time.Time
}

type DomainServiceOption func(*DomainService)

type DomainDNSInstructions struct {
	Target       string
	TargetIPs    []string
	Instructions string
}

type DomainVerifyResult struct {
	Domain     model.CustomDomain
	ResolvedTo string
}

type DomainError struct {
	Code    string
	Message string
}

func (err *DomainError) Error() string {
	if err == nil {
		return ""
	}
	return err.Message
}

func NewDomainService(domainStore store.DomainStore, constraints model.DomainConstraints, platformHosts []string, publicIPs []net.IP, options ...DomainServiceOption) DomainService {
	normalizedHosts := make([]string, 0, len(platformHosts))
	for _, platformHost := range platformHosts {
		normalizedHost := normalizeHost(platformHost)
		if normalizedHost != "" {
			normalizedHosts = append(normalizedHosts, normalizedHost)
		}
	}

	clonedIPs := make([]net.IP, 0, len(publicIPs))
	for _, publicIP := range publicIPs {
		if publicIP == nil {
			continue
		}
		clonedIPs = append(clonedIPs, slices.Clone(publicIP))
	}

	domainService := DomainService{
		store:            domainStore,
		constraints:      constraints,
		platformHosts:    normalizedHosts,
		publicIPs:        clonedIPs,
		tlsALPNAvailable: true,
		httpAvailable:    true,
		resolver:         netDomainResolver{},
		now:              func() time.Time { return time.Now().UTC() },
	}
	for _, option := range options {
		option(&domainService)
	}
	return domainService
}

func WithDomainResolver(resolver DomainResolver) DomainServiceOption {
	return func(domainService *DomainService) {
		if resolver != nil {
			domainService.resolver = resolver
		}
	}
}

func WithDomainClock(now func() time.Time) DomainServiceOption {
	return func(domainService *DomainService) {
		if now != nil {
			domainService.now = now
		}
	}
}

func WithDomainChallengeAvailability(tlsALPNAvailable bool, httpAvailable bool) DomainServiceOption {
	return func(domainService *DomainService) {
		domainService.tlsALPNAvailable = tlsALPNAvailable
		domainService.httpAvailable = httpAvailable
	}
}

func (service DomainService) Enabled() bool {
	return service.constraints.Enabled
}

func (service DomainService) DNSInstructions() DomainDNSInstructions {
	target := ""
	if len(service.platformHosts) > 0 {
		target = service.platformHosts[0]
	}
	targetIPs := make([]string, 0, len(service.publicIPs))
	for _, publicIP := range service.publicIPs {
		targetIPs = append(targetIPs, publicIP.String())
	}

	instructions := "Add an A/AAAA record pointing to this server."
	if target != "" && len(targetIPs) > 0 {
		instructions = "Add a CNAME record pointing to " + target + ", or A/AAAA records pointing to the IPs above."
	} else if target != "" {
		instructions = "Add a CNAME record pointing to " + target + "."
	}

	return DomainDNSInstructions{
		Target:       target,
		TargetIPs:    targetIPs,
		Instructions: instructions,
	}
}

func (service DomainService) CreateUserDomain(ctx context.Context, userID int64, name string) (model.CustomDomain, error) {
	return service.createDomain(ctx, model.DomainOwnerTypeUser, userID, name)
}

func (service DomainService) CreateOrganizationDomain(ctx context.Context, orgID int64, name string) (model.CustomDomain, error) {
	return service.createDomain(ctx, model.DomainOwnerTypeOrg, orgID, name)
}

func (service DomainService) ListUserDomains(ctx context.Context, userID int64) ([]model.CustomDomain, error) {
	return service.listDomainsByOwner(ctx, model.DomainOwnerTypeUser, userID)
}

func (service DomainService) ListOrganizationDomains(ctx context.Context, orgID int64) ([]model.CustomDomain, error) {
	return service.listDomainsByOwner(ctx, model.DomainOwnerTypeOrg, orgID)
}

func (service DomainService) GetUserDomain(ctx context.Context, userID int64, domainName string) (model.CustomDomain, error) {
	return service.getOwnedDomain(ctx, model.DomainOwnerTypeUser, userID, domainName)
}

func (service DomainService) GetOrganizationDomain(ctx context.Context, orgID int64, domainName string) (model.CustomDomain, error) {
	return service.getOwnedDomain(ctx, model.DomainOwnerTypeOrg, orgID, domainName)
}

func (service DomainService) DeleteUserDomain(ctx context.Context, userID int64, domainName string) error {
	return service.deleteOwnedDomain(ctx, model.DomainOwnerTypeUser, userID, domainName)
}

func (service DomainService) DeleteOrganizationDomain(ctx context.Context, orgID int64, domainName string) error {
	return service.deleteOwnedDomain(ctx, model.DomainOwnerTypeOrg, orgID, domainName)
}

func (service DomainService) ConfigureUserSSL(ctx context.Context, userID int64, domainName string, challenge model.SSLChallenge, provider string) (model.CustomDomain, error) {
	return service.configureSSL(ctx, model.DomainOwnerTypeUser, userID, domainName, challenge, provider)
}

func (service DomainService) ConfigureOrganizationSSL(ctx context.Context, orgID int64, domainName string, challenge model.SSLChallenge, provider string) (model.CustomDomain, error) {
	return service.configureSSL(ctx, model.DomainOwnerTypeOrg, orgID, domainName, challenge, provider)
}

func (service DomainService) VerifyUserDomain(ctx context.Context, userID int64, domainName string) (DomainVerifyResult, error) {
	return service.verifyOwnedDomain(ctx, model.DomainOwnerTypeUser, userID, domainName)
}

func (service DomainService) VerifyOrganizationDomain(ctx context.Context, orgID int64, domainName string) (DomainVerifyResult, error) {
	return service.verifyOwnedDomain(ctx, model.DomainOwnerTypeOrg, orgID, domainName)
}

func (service DomainService) ListDomains(ctx context.Context) ([]model.CustomDomain, error) {
	domains, listError := service.store.ListCustomDomains(ctx)
	if listError != nil {
		return nil, listError
	}
	sortCustomDomains(domains)
	return domains, nil
}

func (service DomainService) GetDomain(ctx context.Context, domainName string) (model.CustomDomain, error) {
	return service.lookupDomain(ctx, domainName)
}

func (service DomainService) SuspendDomain(ctx context.Context, domainName string, reason string) (model.CustomDomain, error) {
	domain, lookupError := service.lookupDomain(ctx, domainName)
	if lookupError != nil {
		return model.CustomDomain{}, lookupError
	}
	domain.Status = model.DomainStatusSuspended
	domain.SuspensionReason = strings.TrimSpace(reason)
	domain.UpdatedAt = service.now()
	return service.store.SaveCustomDomain(ctx, domain)
}

func (service DomainService) UnsuspendDomain(ctx context.Context, domainName string) (model.CustomDomain, error) {
	domain, lookupError := service.lookupDomain(ctx, domainName)
	if lookupError != nil {
		return model.CustomDomain{}, lookupError
	}
	domain.SuspensionReason = ""
	if domain.VerificationStatus == model.VerificationStatusVerified {
		domain.Status = model.DomainStatusActive
	} else {
		domain.Status = model.DomainStatusPending
	}
	domain.UpdatedAt = service.now()
	return service.store.SaveCustomDomain(ctx, domain)
}

func (service DomainService) RenewDomainSSL(ctx context.Context, domainName string) (model.CustomDomain, error) {
	domain, lookupError := service.lookupDomain(ctx, domainName)
	if lookupError != nil {
		return model.CustomDomain{}, lookupError
	}
	if !domain.SSLEnabled {
		return model.CustomDomain{}, &DomainError{Code: "DOMAIN_NOT_VERIFIED", Message: "domain ssl is not configured"}
	}
	domain.SSLStatus = model.SSLStatusPending
	domain.SSLError = ""
	if domain.SSLChallenge == "" {
		domain.SSLChallenge = model.SelectSSLChallenge(domain, service.tlsALPNAvailable, service.httpAvailable)
	}
	domain.UpdatedAt = service.now()
	return service.store.SaveCustomDomain(ctx, domain)
}

func (service DomainService) DeleteDomain(ctx context.Context, domainName string) error {
	domain, lookupError := service.lookupDomain(ctx, domainName)
	if lookupError != nil {
		return lookupError
	}
	return service.store.DeleteCustomDomainByID(ctx, domain.ID)
}

func (service DomainService) createDomain(ctx context.Context, ownerType model.DomainOwnerType, ownerID int64, name string) (model.CustomDomain, error) {
	if !service.constraints.Enabled {
		return model.CustomDomain{}, &DomainError{Code: "DOMAIN_NOT_FOUND", Message: "custom domains are disabled"}
	}
	if validationError := model.ValidateDomainName(name, service.constraints); validationError != nil {
		message := validationError.Error()
		code := "DOMAIN_INVALID"
		if strings.Contains(message, "reserved") || strings.Contains(message, "blocked pattern") {
			code = "DOMAIN_RESERVED"
		}
		return model.CustomDomain{}, &DomainError{Code: code, Message: message}
	}

	existingDomains, listError := service.listDomainsByOwner(ctx, ownerType, ownerID)
	if listError != nil {
		return model.CustomDomain{}, listError
	}
	if _, lookupError := service.store.FindCustomDomainByDomain(ctx, name); lookupError == nil {
		return model.CustomDomain{}, &DomainError{Code: "DOMAIN_EXISTS", Message: "domain already registered"}
	} else if !errors.Is(lookupError, store.ErrNotFound) {
		return model.CustomDomain{}, lookupError
	}
	limit := service.limitForOwner(ownerType)
	if limit > 0 && len(existingDomains) >= limit {
		return model.CustomDomain{}, &DomainError{Code: "DOMAIN_LIMIT", Message: "domain limit reached"}
	}

	now := service.now()
	domain := model.CustomDomain{
		OwnerType:          ownerType,
		OwnerID:            ownerID,
		Domain:             model.NormalizeDomainName(name),
		IsWildcard:         strings.HasPrefix(model.NormalizeDomainName(name), "*."),
		Status:             model.DomainStatusPending,
		VerificationStatus: model.VerificationStatusPending,
		SSLStatus:          model.SSLStatusNone,
		SSLChallenge:       model.SSLChallengeAuto,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	switch ownerType {
	case model.DomainOwnerTypeUser:
		domain.UserID = ownerID
	case model.DomainOwnerTypeOrg:
		domain.OrganizationID = ownerID
	}

	return service.store.SaveCustomDomain(ctx, domain)
}

func (service DomainService) listDomainsByOwner(ctx context.Context, ownerType model.DomainOwnerType, ownerID int64) ([]model.CustomDomain, error) {
	domains, listError := service.store.ListCustomDomainsByOwner(ctx, ownerType, ownerID)
	if listError != nil {
		return nil, listError
	}
	sortCustomDomains(domains)
	return domains, nil
}

func (service DomainService) getOwnedDomain(ctx context.Context, ownerType model.DomainOwnerType, ownerID int64, domainName string) (model.CustomDomain, error) {
	domain, lookupError := service.lookupDomain(ctx, domainName)
	if lookupError != nil {
		return model.CustomDomain{}, lookupError
	}
	if domain.OwnerType != ownerType || domain.OwnerID != ownerID {
		return model.CustomDomain{}, &DomainError{Code: "DOMAIN_NOT_FOUND", Message: "domain not found"}
	}
	return domain, nil
}

func (service DomainService) deleteOwnedDomain(ctx context.Context, ownerType model.DomainOwnerType, ownerID int64, domainName string) error {
	domain, lookupError := service.getOwnedDomain(ctx, ownerType, ownerID, domainName)
	if lookupError != nil {
		return lookupError
	}
	return service.store.DeleteCustomDomainByID(ctx, domain.ID)
}

func (service DomainService) configureSSL(ctx context.Context, ownerType model.DomainOwnerType, ownerID int64, domainName string, challenge model.SSLChallenge, provider string) (model.CustomDomain, error) {
	domain, lookupError := service.getOwnedDomain(ctx, ownerType, ownerID, domainName)
	if lookupError != nil {
		return model.CustomDomain{}, lookupError
	}
	if domain.VerificationStatus != model.VerificationStatusVerified {
		return model.CustomDomain{}, &DomainError{Code: "DOMAIN_NOT_VERIFIED", Message: "domain not yet verified"}
	}
	if challenge == "" || challenge == model.SSLChallengeAuto {
		challenge = model.SelectSSLChallenge(domain, service.tlsALPNAvailable, service.httpAvailable)
	}
	if challenge == model.SSLChallengeDNS01 && strings.TrimSpace(provider) == "" {
		return model.CustomDomain{}, &DomainError{Code: "SSL_PROVIDER_INVALID", Message: "dns provider is required for dns-01"}
	}
	domain.SSLEnabled = true
	domain.SSLProvider = strings.TrimSpace(strings.ToLower(provider))
	domain.SSLChallenge = challenge
	domain.SSLStatus = model.SSLStatusPending
	domain.SSLError = ""
	domain.UpdatedAt = service.now()
	return service.store.SaveCustomDomain(ctx, domain)
}

func (service DomainService) verifyOwnedDomain(ctx context.Context, ownerType model.DomainOwnerType, ownerID int64, domainName string) (DomainVerifyResult, error) {
	domain, lookupError := service.getOwnedDomain(ctx, ownerType, ownerID, domainName)
	if lookupError != nil {
		return DomainVerifyResult{}, lookupError
	}
	return service.verifyDomain(ctx, domain)
}

func (service DomainService) verifyDomain(ctx context.Context, domain model.CustomDomain) (DomainVerifyResult, error) {
	resolvedIPs, lookupError := service.resolver.LookupIP(domain.Domain)
	if lookupError != nil && len(service.platformHosts) == 0 {
		domain.Status = model.DomainStatusError
		domain.VerificationStatus = model.VerificationStatusFailed
		domain.VerificationError = "DNS_LOOKUP_FAILED"
		domain.UpdatedAt = service.now()
		savedDomain, saveError := service.store.SaveCustomDomain(ctx, domain)
		if saveError != nil {
			return DomainVerifyResult{}, saveError
		}
		return DomainVerifyResult{Domain: savedDomain}, &DomainError{Code: "DNS_LOOKUP_FAILED", Message: "DNS lookup failed"}
	}

	matchedTarget := ""
	if lookupError == nil {
		for _, resolvedIP := range resolvedIPs {
			for _, publicIP := range service.publicIPs {
				if resolvedIP.Equal(publicIP) {
					matchedTarget = resolvedIP.String()
					break
				}
			}
			if matchedTarget != "" {
				break
			}
		}
	}
	if matchedTarget == "" && len(service.platformHosts) > 0 {
		resolvedCNAME, cnameError := service.resolver.LookupCNAME(domain.Domain)
		if cnameError == nil {
			normalizedCNAME := strings.TrimSuffix(normalizeHost(resolvedCNAME), ".")
			for _, platformHost := range service.platformHosts {
				if normalizedCNAME == normalizeHost(platformHost) {
					matchedTarget = normalizedCNAME
					break
				}
			}
		}
	}
	if matchedTarget == "" {
		code := "DNS_MISMATCH"
		message := "Domain does not resolve to this server. DNS propagation can take up to 48 hours."
		if lookupError != nil {
			code = "DNS_LOOKUP_FAILED"
			message = "DNS lookup failed"
		}
		domain.Status = model.DomainStatusError
		domain.VerificationStatus = model.VerificationStatusFailed
		domain.VerificationError = code
		domain.UpdatedAt = service.now()
		savedDomain, saveError := service.store.SaveCustomDomain(ctx, domain)
		if saveError != nil {
			return DomainVerifyResult{}, saveError
		}
		return DomainVerifyResult{Domain: savedDomain}, &DomainError{Code: code, Message: message}
	}

	now := service.now()
	domain.Status = model.DomainStatusActive
	domain.VerificationStatus = model.VerificationStatusVerified
	domain.VerificationError = ""
	domain.VerifiedAt = now
	domain.ResolvedTarget = matchedTarget
	domain.UpdatedAt = now
	savedDomain, saveError := service.store.SaveCustomDomain(ctx, domain)
	if saveError != nil {
		return DomainVerifyResult{}, saveError
	}
	return DomainVerifyResult{Domain: savedDomain, ResolvedTo: matchedTarget}, nil
}

func (service DomainService) lookupDomain(ctx context.Context, domainName string) (model.CustomDomain, error) {
	domain, lookupError := service.store.FindCustomDomainByDomain(ctx, domainName)
	if lookupError != nil {
		if errors.Is(lookupError, store.ErrNotFound) {
			return model.CustomDomain{}, &DomainError{Code: "DOMAIN_NOT_FOUND", Message: "domain not found"}
		}
		return model.CustomDomain{}, lookupError
	}
	return domain, nil
}

func (service DomainService) limitForOwner(ownerType model.DomainOwnerType) int {
	if ownerType == model.DomainOwnerTypeOrg {
		return service.constraints.MaxDomainsPerOrg
	}
	return service.constraints.MaxDomainsPerUser
}

func sortCustomDomains(domains []model.CustomDomain) {
	slices.SortFunc(domains, func(left model.CustomDomain, right model.CustomDomain) int {
		return strings.Compare(left.Domain, right.Domain)
	})
}

type netDomainResolver struct{}

func (netDomainResolver) LookupIP(host string) ([]net.IP, error) {
	return net.LookupIP(host)
}

func (netDomainResolver) LookupCNAME(host string) (string, error) {
	return net.LookupCNAME(host)
}
