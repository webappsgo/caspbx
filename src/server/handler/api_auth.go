package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/casapps/caspbx/src/server/model"
	"github.com/casapps/caspbx/src/server/service"
)

type APIAuthHandler struct {
	routePrefix      string
	authService      service.AuthService
	registrationMode model.RegistrationMode
}

type APIUserHandler struct {
	routePrefix          string
	authService          service.AuthService
	domainService        service.DomainService
	communicationService service.UserCommunicationsService
}

type APIAdminHandler struct {
	routePrefix     string
	authService     service.AuthService
	domainService   service.DomainService
	asteriskService service.AsteriskService
	pbxService      service.PBXService
	operatorService service.OperatorService
}

func NewAPIAuthHandler(routePrefix string, authService service.AuthService, registrationMode model.RegistrationMode) http.Handler {
	return APIAuthHandler{
		routePrefix:      routePrefix,
		authService:      authService,
		registrationMode: registrationMode,
	}
}

func NewAPIUserHandler(routePrefix string, authService service.AuthService, domainService service.DomainService, communicationService ...service.UserCommunicationsService) http.Handler {
	var resolvedCommunicationService service.UserCommunicationsService
	if len(communicationService) > 0 {
		resolvedCommunicationService = communicationService[0]
	}
	return APIUserHandler{
		routePrefix:          routePrefix,
		authService:          authService,
		domainService:        domainService,
		communicationService: resolvedCommunicationService,
	}
}

func NewAPIAdminHandler(routePrefix string, authService service.AuthService, domainService service.DomainService, asteriskService service.AsteriskService, pbxService service.PBXService, operatorService ...service.OperatorService) http.Handler {
	var resolvedOperatorService service.OperatorService
	if len(operatorService) > 0 {
		resolvedOperatorService = operatorService[0]
	}
	return APIAdminHandler{
		routePrefix:     routePrefix,
		authService:     authService,
		domainService:   domainService,
		asteriskService: asteriskService,
		pbxService:      pbxService,
		operatorService: resolvedOperatorService,
	}
}

func (handler APIAuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch relativePath := routeTail(handler.routePrefix, r.URL.Path); relativePath {
	case "", "/":
		if !allowsReadOnlyMethod(w, r) {
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"scope":             "auth_api",
			"login_path":        joinPath(handler.routePrefix, "login"),
			"register_path":     joinPath(handler.routePrefix, "register"),
			"logout_path":       joinPath(handler.routePrefix, "logout"),
			"refresh_path":      joinPath(handler.routePrefix, "refresh"),
			"registration_mode": string(handler.registrationMode),
		})
	case "login":
		handler.handleLogin(w, r)
	case "register":
		handler.handleRegister(w, r)
	case "logout":
		handler.handleLogout(w, r)
	case "refresh":
		handler.handleRefresh(w, r)
	case "2fa", "passkey/challenge", "passkey/verify", "password/forgot", "password/reset", "username/forgot", "recovery/use", "verify", "ldap":
		writeAPINotImplemented(w, r, joinPath(handler.routePrefix, relativePath))
	default:
		switch {
		case strings.HasPrefix(relativePath, "invite/user/"),
			strings.HasPrefix(relativePath, "invite/server/"),
			strings.HasPrefix(relativePath, "oidc/"):
			writeAPINotImplemented(w, r, joinPath(handler.routePrefix, relativePath))
		default:
			http.NotFound(w, r)
		}
	}
}

func (handler APIAuthHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		writeJSON(w, http.StatusOK, map[string]string{
			"route":      joinPath(handler.routePrefix, "login"),
			"method":     http.MethodPost,
			"auth_model": "Bearer usr_ token",
		})
	case http.MethodPost:
		requestBody, parseError := readAuthLoginRequest(r)
		if parseError != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid login request"})
			return
		}

		user, issuedToken, authError := handler.authService.AuthenticateUserAPI(r.Context(), firstNonEmpty(requestBody.Identifier, requestBody.Username), requestBody.Password)
		if authError != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":       "authenticated",
			"token_type":   "Bearer",
			"token":        issuedToken.Value,
			"token_prefix": issuedToken.Token.TokenPrefix,
			"token_scope":  issuedToken.Token.Scope,
			"username":     user.Username,
			"expires_at":   formatOptionalTime(issuedToken.Token.ExpiresAt),
			"rehash_hint":  issuedToken.RehashHint,
		})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (handler APIAuthHandler) handleRegister(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		writeJSON(w, http.StatusOK, map[string]string{
			"route":             joinPath(handler.routePrefix, "register"),
			"registration_mode": string(handler.registrationMode),
		})
	case http.MethodPost:
		if handler.registrationMode != model.RegistrationModePublic {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": service.ErrRegistrationRestricted.Error()})
			return
		}
		writeAPINotImplemented(w, r, joinPath(handler.routePrefix, "register"))
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (handler APIAuthHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	token, tokenError := readBearerToken(r)
	if tokenError != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}
	if logoutError := handler.authService.LogoutUserToken(r.Context(), token); logoutError != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

func (handler APIAuthHandler) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	token, tokenError := readBearerToken(r)
	if tokenError != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}

	refreshedToken, refreshError := handler.authService.RefreshUserToken(r.Context(), token)
	if refreshError != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "refreshed",
		"token_type":   "Bearer",
		"token":        refreshedToken.Value,
		"token_prefix": refreshedToken.Token.TokenPrefix,
		"token_scope":  refreshedToken.Token.Scope,
		"expires_at":   formatOptionalTime(refreshedToken.Token.ExpiresAt),
	})
}

func (handler APIUserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	relativePath := routeTail(handler.routePrefix, r.URL.Path)
	if (relativePath == "" || relativePath == "profile") && r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	tokenRecord, user, authorized := handler.resolveUser(r)
	if !authorized {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}

	switch relativePath {
	case "", "profile":
		writeJSON(w, http.StatusOK, map[string]any{
			"username":      user.Username,
			"display_name":  user.DisplayName,
			"account_email": user.AccountEmail,
			"visibility":    user.Visibility,
			"token_prefix":  tokenRecord.TokenPrefix,
			"token_scope":   tokenRecord.Scope,
			"expires_at":    formatOptionalTime(tokenRecord.ExpiresAt),
		})
	default:
		if route, ok := parseUserCommunicationRoute(relativePath); ok {
			handler.handleUserCommunicationSurface(w, r, route, user, tokenRecord.TokenPrefix, tokenRecord.Scope)
			return
		}
		if relativePath == "domains" || strings.HasPrefix(relativePath, "domains/") {
			handler.handleDomainSurface(w, r, user)
			return
		}
		http.NotFound(w, r)
	}
}

func (handler APIAdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch relativePath := routeTail(handler.routePrefix, r.URL.Path); relativePath {
	case "":
		handler.handleRoot(w, r)
	case "profile", "profile/security":
		handler.handleProfile(w, r)
	case "server":
		handler.handleProtectedSurface(w, r, relativePath)
	default:
		if _, isAsteriskRoute := normalizeAsteriskRoute(handler.routePrefix, relativePath); isAsteriskRoute {
			handler.handleAsteriskSurface(w, r, relativePath)
			return
		}
		if relativePath == "server/domains" || strings.HasPrefix(relativePath, "server/domains/") {
			handler.handleDomainSurface(w, r, relativePath)
			return
		}
		if relativePath == "server/settings" || relativePath == "server/users" || relativePath == "server/security/auth" {
			handler.handleProtectedSurface(w, r, relativePath)
			return
		}
		http.NotFound(w, r)
	}
}

func (handler APIAdminHandler) handleRoot(w http.ResponseWriter, r *http.Request) {
	if !allowsReadOnlyMethod(w, r) {
		return
	}

	tokenRecord, admin, authorized := handler.resolveAdmin(r)
	if !authorized {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "admin authentication required"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"scope":        "admin_api",
		"username":     admin.Username,
		"role":         admin.Role,
		"token_prefix": tokenRecord.TokenPrefix,
		"token_scope":  tokenRecord.Scope,
		"expires_at":   formatOptionalTime(tokenRecord.ExpiresAt),
	})
}

func (handler APIAdminHandler) handleProfile(w http.ResponseWriter, r *http.Request) {
	if !allowsReadOnlyMethod(w, r) {
		return
	}

	tokenRecord, admin, authorized := handler.resolveAdmin(r)
	if !authorized {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "admin authentication required"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"username":      admin.Username,
		"role":          admin.Role,
		"account_email": admin.AccountEmail,
		"token_prefix":  tokenRecord.TokenPrefix,
		"token_scope":   tokenRecord.Scope,
		"expires_at":    formatOptionalTime(tokenRecord.ExpiresAt),
	})
}

func (handler APIAdminHandler) handleProtectedSurface(w http.ResponseWriter, r *http.Request, relativePath string) {
	if !allowsReadOnlyMethod(w, r) {
		return
	}

	tokenRecord, admin, authorized := handler.resolveAdmin(r)
	if !authorized {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "admin authentication required"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"surface":      relativePath,
		"admin":        admin.Username,
		"token_prefix": tokenRecord.TokenPrefix,
		"token_scope":  tokenRecord.Scope,
		"expires_at":   formatOptionalTime(tokenRecord.ExpiresAt),
	})
}

func (handler APIAdminHandler) resolveAdmin(r *http.Request) (model.Token, model.Admin, bool) {
	token, tokenError := readBearerToken(r)
	if tokenError != nil {
		return model.Token{}, model.Admin{}, false
	}

	tokenRecord, resolveError := handler.authService.ResolveAdminToken(r.Context(), token)
	if resolveError != nil {
		return model.Token{}, model.Admin{}, false
	}

	admin, adminError := handler.authService.FindAdminByID(r.Context(), tokenRecord.OwnerID)
	if adminError != nil {
		return model.Token{}, model.Admin{}, false
	}

	return tokenRecord, admin, true
}

func (handler APIUserHandler) resolveUser(r *http.Request) (model.Token, model.User, bool) {
	token, tokenError := readBearerToken(r)
	if tokenError != nil {
		return model.Token{}, model.User{}, false
	}

	tokenRecord, resolveError := handler.authService.ResolveUserToken(r.Context(), token)
	if resolveError != nil {
		return model.Token{}, model.User{}, false
	}

	user, userError := handler.authService.FindUserByID(r.Context(), tokenRecord.OwnerID)
	if userError != nil {
		return model.Token{}, model.User{}, false
	}

	return tokenRecord, user, true
}

func readBearerToken(r *http.Request) (string, error) {
	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	if authorization == "" {
		return "", http.ErrNoCookie
	}

	scheme, token, found := strings.Cut(authorization, " ")
	if !found || !strings.EqualFold(strings.TrimSpace(scheme), "Bearer") || strings.TrimSpace(token) == "" {
		return "", fmt.Errorf("%w: invalid bearer token", http.ErrNoCookie)
	}

	return strings.TrimSpace(token), nil
}

func writeAPINotImplemented(w http.ResponseWriter, r *http.Request, endpoint string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, http.StatusNotImplemented, map[string]string{
		"error":    "not implemented",
		"endpoint": endpoint,
	})
}

func formatOptionalTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
