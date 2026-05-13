package model

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type DomainOwnerType string
type DomainStatus string
type VerificationStatus string
type SSLStatus string
type SSLChallenge string

const (
	DomainOwnerTypeUser DomainOwnerType = "user"
	DomainOwnerTypeOrg  DomainOwnerType = "org"

	DomainStatusPending   DomainStatus = "pending"
	DomainStatusActive    DomainStatus = "active"
	DomainStatusSuspended DomainStatus = "suspended"
	DomainStatusError     DomainStatus = "error"

	VerificationStatusPending  VerificationStatus = "pending"
	VerificationStatusVerified VerificationStatus = "verified"
	VerificationStatusFailed   VerificationStatus = "failed"

	SSLStatusNone    SSLStatus = "none"
	SSLStatusPending SSLStatus = "pending"
	SSLStatusActive  SSLStatus = "active"
	SSLStatusExpired SSLStatus = "expired"
	SSLStatusError   SSLStatus = "error"

	SSLChallengeAuto    SSLChallenge = "auto"
	SSLChallengeHTTP01  SSLChallenge = "http-01"
	SSLChallengeTLSALPN SSLChallenge = "tls-alpn-01"
	SSLChallengeDNS01   SSLChallenge = "dns-01"
)

var domainLabelPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

type DomainConstraints struct {
	Enabled           bool
	MaxDomainsPerUser int
	MaxDomainsPerOrg  int
	RequireSSL        bool
	AllowApex         bool
	AllowSubdomain    bool
	AllowWildcard     bool
	VerificationTTL   time.Duration
	SSLRenewalDays    int
	Reserved          []string
	BlockedPatterns   []string
}

type CustomDomain struct {
	ID                 int64
	TenantID           int64
	OrganizationID     int64
	UserID             int64
	OwnerType          DomainOwnerType
	OwnerID            int64
	Domain             string
	IsWildcard         bool
	VerificationError  string
	VerifiedAt         time.Time
	ResolvedTarget     string
	SSLProvider        string
	SSLChallenge       SSLChallenge
	SSLError           string
	SSLIssuedAt        time.Time
	SSLExpiresAt       time.Time
	SuspensionReason   string
	Status             DomainStatus
	VerificationStatus VerificationStatus
	SSLStatus          SSLStatus
	SSLEnabled         bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func DefaultDomainConstraints() DomainConstraints {
	return DomainConstraints{
		Enabled:           false,
		MaxDomainsPerUser: 5,
		MaxDomainsPerOrg:  20,
		RequireSSL:        true,
		AllowApex:         true,
		AllowSubdomain:    true,
		AllowWildcard:     false,
		VerificationTTL:   24 * time.Hour,
		SSLRenewalDays:    7,
		Reserved: []string{
			"localhost",
			"*.local",
			"*.test",
			"*.example",
			"*.invalid",
		},
		BlockedPatterns: []string{
			`.*\.(gov|mil|edu)$`,
		},
	}
}

func NormalizeDomainName(input string) string {
	return strings.TrimSuffix(strings.TrimSpace(strings.ToLower(input)), ".")
}

func ValidateDomainName(input string, constraints DomainConstraints) error {
	domain := NormalizeDomainName(input)
	if domain == "" {
		return errors.New("domain is required")
	}
	if strings.Contains(domain, "://") || strings.Contains(domain, "/") || strings.Contains(domain, ":") {
		return errors.New("domain must not include scheme, path, or port")
	}

	isWildcard := strings.HasPrefix(domain, "*.")
	normalizedDomain := strings.TrimPrefix(domain, "*.")

	if isWildcard && !constraints.AllowWildcard {
		return errors.New("wildcard domains are disabled")
	}

	labels := strings.Split(normalizedDomain, ".")
	if len(labels) < 2 {
		return errors.New("domain must contain a valid TLD")
	}

	isApex := len(labels) == 2
	if isApex && !constraints.AllowApex {
		return errors.New("apex domains are disabled")
	}
	if !isApex && !constraints.AllowSubdomain && !isWildcard {
		return errors.New("subdomains are disabled")
	}

	for _, reservedDomain := range constraints.Reserved {
		if matchesReservedDomain(domain, reservedDomain) {
			return fmt.Errorf("domain %q is reserved", domain)
		}
	}

	for _, blockedPattern := range constraints.BlockedPatterns {
		if regexp.MustCompile(blockedPattern).MatchString(normalizedDomain) {
			return fmt.Errorf("domain %q matches blocked pattern", domain)
		}
	}

	for _, label := range labels {
		if !domainLabelPattern.MatchString(label) {
			return errors.New("domain contains invalid label")
		}
	}

	topLevelDomain := labels[len(labels)-1]
	if !regexp.MustCompile(`^[a-z]{2,}$`).MatchString(topLevelDomain) {
		return errors.New("domain must contain a valid TLD")
	}

	return nil
}

func SelectSSLChallenge(domain CustomDomain, tlsALPNAvailable bool, httpAvailable bool) SSLChallenge {
	if domain.SSLProvider != "" {
		return SSLChallengeDNS01
	}
	if domain.IsWildcard {
		return SSLChallengeDNS01
	}
	if tlsALPNAvailable {
		return SSLChallengeTLSALPN
	}
	if httpAvailable {
		return SSLChallengeHTTP01
	}
	return SSLChallengeDNS01
}

func (domain CustomDomain) IsActive() bool {
	return domain.Status == DomainStatusActive && domain.VerificationStatus == VerificationStatusVerified
}

func matchesReservedDomain(domain string, reservedDomain string) bool {
	if strings.HasPrefix(reservedDomain, "*.") {
		return strings.HasSuffix(domain, strings.TrimPrefix(reservedDomain, "*"))
	}
	return domain == reservedDomain
}
