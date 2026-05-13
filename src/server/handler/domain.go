package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/casapps/caspbx/src/server/model"
	"github.com/casapps/caspbx/src/server/service"
)

type domainRequest struct {
	Domain    string `json:"domain"`
	Challenge string `json:"challenge"`
	Provider  string `json:"provider"`
	Reason    string `json:"reason"`
}

func decodeDomainRequest(r *http.Request) (domainRequest, error) {
	request := domainRequest{}
	if r.Body == nil {
		return request, nil
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil && !errors.Is(err, io.EOF) {
		return domainRequest{}, err
	}
	return request, nil
}

func domainRouteParts(relativePath string, prefix string) (string, string, bool) {
	if relativePath == prefix {
		return "", "", true
	}
	if !strings.HasPrefix(relativePath, prefix+"/") {
		return "", "", false
	}
	trimmed := strings.TrimPrefix(relativePath, prefix+"/")
	parts := strings.Split(trimmed, "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		return "", "", false
	}
	return model.NormalizeDomainName(parts[0]), strings.Join(parts[1:], "/"), true
}

func writeDomainTextList(w http.ResponseWriter, domains []model.CustomDomain, instructions service.DomainDNSInstructions) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	for _, domain := range domains {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", domain.Domain, domain.Status, domain.VerificationStatus, domain.SSLStatus)
	}
	if instructions.Target != "" || len(instructions.TargetIPs) > 0 {
		fmt.Fprintf(w, "\nTarget: %s\nIPs: %s\n", instructions.Target, strings.Join(instructions.TargetIPs, ", "))
	}
}

func writeDomainTextDetail(w http.ResponseWriter, domain model.CustomDomain, instructions service.DomainDNSInstructions) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "Domain: %s\nStatus: %s\nVerification: %s\nSSL: %s\n", domain.Domain, domain.Status, domain.VerificationStatus, domain.SSLStatus)
	if instructions.Target != "" || len(instructions.TargetIPs) > 0 {
		fmt.Fprintf(w, "Target: %s\nTarget IPs: %s\n", instructions.Target, strings.Join(instructions.TargetIPs, ", "))
	}
}

func domainResponse(domain model.CustomDomain, instructions service.DomainDNSInstructions) map[string]any {
	return map[string]any{
		"id":                  domain.ID,
		"domain":              domain.Domain,
		"owner_type":          domain.OwnerType,
		"owner_id":            domain.OwnerID,
		"status":              domain.Status,
		"verification_status": domain.VerificationStatus,
		"verification_error":  domain.VerificationError,
		"verified_at":         formatOptionalTime(domain.VerifiedAt),
		"resolved_to":         domain.ResolvedTarget,
		"ssl_status":          domain.SSLStatus,
		"ssl_enabled":         domain.SSLEnabled,
		"ssl_provider":        domain.SSLProvider,
		"ssl_challenge":       domain.SSLChallenge,
		"ssl_error":           domain.SSLError,
		"ssl_issued_at":       formatOptionalTime(domain.SSLIssuedAt),
		"ssl_expires_at":      formatOptionalTime(domain.SSLExpiresAt),
		"suspension_reason":   domain.SuspensionReason,
		"dns_instructions": map[string]any{
			"target":       instructions.Target,
			"target_ips":   instructions.TargetIPs,
			"instructions": instructions.Instructions,
		},
	}
}

func writeDomainServiceError(w http.ResponseWriter, err error) {
	domainError, ok := err.(*service.DomainError)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":      false,
			"error":   "INTERNAL_ERROR",
			"message": err.Error(),
		})
		return
	}
	writeJSON(w, domainErrorStatus(domainError.Code), map[string]any{
		"ok":      false,
		"error":   domainError.Code,
		"message": domainError.Message,
	})
}

func domainErrorStatus(code string) int {
	switch code {
	case "DOMAIN_EXISTS":
		return http.StatusConflict
	case "DOMAIN_RESERVED", "DOMAIN_LIMIT", "DOMAIN_INVALID", "DOMAIN_NOT_VERIFIED", "DNS_LOOKUP_FAILED", "DNS_MISMATCH", "SSL_PROVIDER_INVALID":
		return http.StatusBadRequest
	case "DOMAIN_NOT_FOUND":
		return http.StatusNotFound
	case "DOMAIN_SUSPENDED":
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

func (handler UserHandler) handleDomainSurface(w http.ResponseWriter, r *http.Request, user model.User) {
	if !handler.domainService.Enabled() {
		http.NotFound(w, r)
		return
	}
	relativePath := routeTail(handler.routePrefix, r.URL.Path)
	domainName, action, found := domainRouteParts(relativePath, "domains")
	if !found {
		http.NotFound(w, r)
		return
	}

	switch {
	case domainName == "":
		switch r.Method {
		case http.MethodGet, http.MethodHead:
			domains, listError := handler.domainService.ListUserDomains(r.Context(), user.ID)
			if listError != nil {
				writeDomainServiceError(w, listError)
				return
			}
			if prefersJSON(r.Header.Get("Accept")) {
				data := make([]map[string]any, 0, len(domains))
				for _, domain := range domains {
					data = append(data, domainResponse(domain, handler.domainService.DNSInstructions()))
				}
				writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": data})
				return
			}
			writeDomainTextList(w, domains, handler.domainService.DNSInstructions())
		case http.MethodPost:
			requestBody, parseError := decodeDomainRequest(r)
			if parseError != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid domain request"})
				return
			}
			domain, createError := handler.domainService.CreateUserDomain(r.Context(), user.ID, requestBody.Domain)
			if createError != nil {
				writeDomainServiceError(w, createError)
				return
			}
			writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "data": domainResponse(domain, handler.domainService.DNSInstructions())})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	case action == "":
		switch r.Method {
		case http.MethodGet, http.MethodHead:
			domain, lookupError := handler.domainService.GetUserDomain(r.Context(), user.ID, domainName)
			if lookupError != nil {
				writeDomainServiceError(w, lookupError)
				return
			}
			if prefersJSON(r.Header.Get("Accept")) {
				writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": domainResponse(domain, handler.domainService.DNSInstructions())})
				return
			}
			writeDomainTextDetail(w, domain, handler.domainService.DNSInstructions())
		case http.MethodDelete:
			if deleteError := handler.domainService.DeleteUserDomain(r.Context(), user.ID, domainName); deleteError != nil {
				writeDomainServiceError(w, deleteError)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": "deleted"})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	case action == "verify":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		result, verifyError := handler.domainService.VerifyUserDomain(r.Context(), user.ID, domainName)
		if verifyError != nil {
			writeDomainServiceError(w, verifyError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": map[string]any{
			"domain":              result.Domain.Domain,
			"verification_status": result.Domain.VerificationStatus,
			"verified_at":         formatOptionalTime(result.Domain.VerifiedAt),
			"status":              result.Domain.Status,
			"resolved_to":         result.ResolvedTo,
		}})
	case action == "ssl":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		requestBody, parseError := decodeDomainRequest(r)
		if parseError != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid domain request"})
			return
		}
		domain, sslError := handler.domainService.ConfigureUserSSL(r.Context(), user.ID, domainName, model.SSLChallenge(strings.TrimSpace(requestBody.Challenge)), requestBody.Provider)
		if sslError != nil {
			writeDomainServiceError(w, sslError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": domainResponse(domain, handler.domainService.DNSInstructions())})
	default:
		http.NotFound(w, r)
	}
}

func (handler APIUserHandler) handleDomainSurface(w http.ResponseWriter, r *http.Request, user model.User) {
	handler.UserHandler().handleDomainSurface(w, r, user)
}

func (handler APIUserHandler) UserHandler() UserHandler {
	return UserHandler{
		routePrefix:   handler.routePrefix,
		authService:   handler.authService,
		domainService: handler.domainService,
	}
}

func (handler OrgHandler) handleDomainSurface(w http.ResponseWriter, r *http.Request, access orgAccessContext, relativePath string) {
	if !handler.domainService.Enabled() {
		http.NotFound(w, r)
		return
	}
	if !access.hasMember {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}
	if !access.member.CanManageOrganization() {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	domainName, action, found := domainRouteParts(relativePath, "domains")
	if !found {
		http.NotFound(w, r)
		return
	}

	switch {
	case domainName == "":
		switch r.Method {
		case http.MethodGet, http.MethodHead:
			domains, listError := handler.domainService.ListOrganizationDomains(r.Context(), access.organization.ID)
			if listError != nil {
				writeDomainServiceError(w, listError)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": domainCollection(domains, handler.domainService.DNSInstructions())})
		case http.MethodPost:
			requestBody, parseError := decodeDomainRequest(r)
			if parseError != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid domain request"})
				return
			}
			domain, createError := handler.domainService.CreateOrganizationDomain(r.Context(), access.organization.ID, requestBody.Domain)
			if createError != nil {
				writeDomainServiceError(w, createError)
				return
			}
			writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "data": domainResponse(domain, handler.domainService.DNSInstructions())})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	case action == "":
		switch r.Method {
		case http.MethodGet, http.MethodHead:
			domain, lookupError := handler.domainService.GetOrganizationDomain(r.Context(), access.organization.ID, domainName)
			if lookupError != nil {
				writeDomainServiceError(w, lookupError)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": domainResponse(domain, handler.domainService.DNSInstructions())})
		case http.MethodDelete:
			if deleteError := handler.domainService.DeleteOrganizationDomain(r.Context(), access.organization.ID, domainName); deleteError != nil {
				writeDomainServiceError(w, deleteError)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": "deleted"})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	case action == "verify":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		result, verifyError := handler.domainService.VerifyOrganizationDomain(r.Context(), access.organization.ID, domainName)
		if verifyError != nil {
			writeDomainServiceError(w, verifyError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": map[string]any{
			"domain":              result.Domain.Domain,
			"verification_status": result.Domain.VerificationStatus,
			"verified_at":         formatOptionalTime(result.Domain.VerifiedAt),
			"status":              result.Domain.Status,
			"resolved_to":         result.ResolvedTo,
		}})
	case action == "ssl":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		requestBody, parseError := decodeDomainRequest(r)
		if parseError != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid domain request"})
			return
		}
		domain, sslError := handler.domainService.ConfigureOrganizationSSL(r.Context(), access.organization.ID, domainName, model.SSLChallenge(strings.TrimSpace(requestBody.Challenge)), requestBody.Provider)
		if sslError != nil {
			writeDomainServiceError(w, sslError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": domainResponse(domain, handler.domainService.DNSInstructions())})
	default:
		http.NotFound(w, r)
	}
}

func (handler APIOrgHandler) handleDomainSurface(w http.ResponseWriter, r *http.Request, access orgAccessContext, relativePath string) {
	OrgHandler{
		routePrefix:   handler.routePrefix,
		authService:   handler.authService,
		domainService: handler.domainService,
		orgStore:      handler.orgStore,
	}.handleDomainSurface(w, r, access, relativePath)
}

func (handler AdminHandler) handleDomainSurface(w http.ResponseWriter, r *http.Request, relativePath string) {
	if !handler.domainService.Enabled() {
		http.NotFound(w, r)
		return
	}
	if _, _, authorized := handler.resolveAdmin(r); !authorized {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "admin authentication required"})
		return
	}
	writeAdminDomainSurface(w, r, relativePath, handler.domainService)
}

func writeAdminDomainSurface(w http.ResponseWriter, r *http.Request, relativePath string, domainService service.DomainService) {
	domainName, action, found := domainRouteParts(relativePath, "server/domains")
	if !found {
		http.NotFound(w, r)
		return
	}
	switch {
	case domainName == "":
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		domains, listError := domainService.ListDomains(r.Context())
		if listError != nil {
			writeDomainServiceError(w, listError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": domainCollection(domains, domainService.DNSInstructions())})
	case action == "":
		switch r.Method {
		case http.MethodGet, http.MethodHead:
			domain, lookupError := domainService.GetDomain(r.Context(), domainName)
			if lookupError != nil {
				writeDomainServiceError(w, lookupError)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": domainResponse(domain, domainService.DNSInstructions())})
		case http.MethodDelete:
			if deleteError := domainService.DeleteDomain(r.Context(), domainName); deleteError != nil {
				writeDomainServiceError(w, deleteError)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": "deleted"})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	case action == "suspend":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		requestBody, parseError := decodeDomainRequest(r)
		if parseError != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid domain request"})
			return
		}
		domain, suspendError := domainService.SuspendDomain(r.Context(), domainName, requestBody.Reason)
		if suspendError != nil {
			writeDomainServiceError(w, suspendError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": domainResponse(domain, domainService.DNSInstructions())})
	case action == "unsuspend":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		domain, unsuspendError := domainService.UnsuspendDomain(r.Context(), domainName)
		if unsuspendError != nil {
			writeDomainServiceError(w, unsuspendError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": domainResponse(domain, domainService.DNSInstructions())})
	case action == "ssl/renew":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		domain, renewError := domainService.RenewDomainSSL(r.Context(), domainName)
		if renewError != nil {
			writeDomainServiceError(w, renewError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": domainResponse(domain, domainService.DNSInstructions())})
	default:
		http.NotFound(w, r)
	}
}

func (handler APIAdminHandler) handleDomainSurface(w http.ResponseWriter, r *http.Request, relativePath string) {
	if !handler.domainService.Enabled() {
		http.NotFound(w, r)
		return
	}
	if _, _, authorized := handler.resolveAdmin(r); !authorized {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "admin authentication required"})
		return
	}
	writeAdminDomainSurface(w, r, relativePath, handler.domainService)
}

func domainCollection(domains []model.CustomDomain, instructions service.DomainDNSInstructions) []map[string]any {
	data := make([]map[string]any, 0, len(domains))
	for _, domain := range domains {
		data = append(data, domainResponse(domain, instructions))
	}
	return data
}
