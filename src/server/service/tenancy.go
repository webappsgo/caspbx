package service

import (
	"context"
	"errors"
	"net"
	"strings"

	"github.com/casapps/caspbx/src/server/model"
	"github.com/casapps/caspbx/src/server/store"
)

var (
	ErrUnknownTenantHost  = errors.New("unknown tenant host")
	ErrInactiveDomainHost = errors.New("inactive tenant domain")
)

type DomainBinding struct {
	Domain         string
	TenantID       int64
	OrganizationID int64
	UserID         int64
	Status         model.DomainStatus
}

type TenantContext struct {
	Host           string
	TenantID       int64
	OrganizationID int64
	UserID         int64
	CustomDomain   string
	PlatformHost   bool
}

type TenantResolver struct {
	platformHosts  map[string]struct{}
	domainBindings map[string]DomainBinding
}

func NewTenantResolver(platformHosts []string, bindings []DomainBinding) TenantResolver {
	resolver := TenantResolver{
		platformHosts:  map[string]struct{}{},
		domainBindings: map[string]DomainBinding{},
	}

	for _, platformHost := range platformHosts {
		resolver.platformHosts[normalizeHost(platformHost)] = struct{}{}
	}
	for _, binding := range bindings {
		resolver.domainBindings[normalizeHost(binding.Domain)] = binding
	}

	return resolver
}

func NewTenantResolverFromStore(ctx context.Context, platformHosts []string, domainStore store.DomainStore) (TenantResolver, error) {
	domains, listError := domainStore.ListCustomDomains(ctx)
	if listError != nil {
		return TenantResolver{}, listError
	}

	bindings := make([]DomainBinding, 0, len(domains))
	for _, domain := range domains {
		bindings = append(bindings, DomainBinding{
			Domain:         domain.Domain,
			TenantID:       domain.TenantID,
			OrganizationID: domain.OrganizationID,
			UserID:         domain.UserID,
			Status:         domain.Status,
		})
	}

	return NewTenantResolver(platformHosts, bindings), nil
}

func (resolver TenantResolver) Resolve(host string) (TenantContext, error) {
	normalizedHost := normalizeHost(host)
	if _, found := resolver.platformHosts[normalizedHost]; found {
		return TenantContext{
			Host:         normalizedHost,
			PlatformHost: true,
		}, nil
	}

	binding, found := resolver.domainBindings[normalizedHost]
	if !found {
		return TenantContext{}, ErrUnknownTenantHost
	}
	if binding.Status != model.DomainStatusActive {
		return TenantContext{}, ErrInactiveDomainHost
	}

	return TenantContext{
		Host:           normalizedHost,
		TenantID:       binding.TenantID,
		OrganizationID: binding.OrganizationID,
		UserID:         binding.UserID,
		CustomDomain:   normalizedHost,
	}, nil
}

func normalizeHost(input string) string {
	trimmedHost := strings.TrimSpace(strings.ToLower(input))
	if strings.Contains(trimmedHost, ":") {
		if parsedHost, _, splitError := net.SplitHostPort(trimmedHost); splitError == nil {
			return parsedHost
		}
	}
	return trimmedHost
}
