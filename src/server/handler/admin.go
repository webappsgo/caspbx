package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/casapps/caspbx/src/server/model"
	"github.com/casapps/caspbx/src/server/service"
)

type AdminHandler struct {
	routePrefix     string
	authService     service.AuthService
	domainService   service.DomainService
	asteriskService service.AsteriskService
	pbxService      service.PBXService
	operatorService service.OperatorService
	adminCookie     SessionCookieConfig
}

func NewAdminHandler(routePrefix string, authService service.AuthService, domainService service.DomainService, asteriskService service.AsteriskService, pbxService service.PBXService, adminCookie SessionCookieConfig, operatorService ...service.OperatorService) http.Handler {
	var resolvedOperatorService service.OperatorService
	if len(operatorService) > 0 {
		resolvedOperatorService = operatorService[0]
	}
	return AdminHandler{
		routePrefix:     routePrefix,
		authService:     authService,
		domainService:   domainService,
		asteriskService: asteriskService,
		pbxService:      pbxService,
		operatorService: resolvedOperatorService,
		adminCookie:     adminCookie,
	}
}

func (handler AdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

func (handler AdminHandler) handleRoot(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		session, admin, authorized := handler.resolveAdmin(r)
		if !authorized {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "admin authentication required",
				"route": handler.routePrefix,
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"scope":      "admin",
			"username":   admin.Username,
			"role":       admin.Role,
			"session_id": session.ID,
		})
	case http.MethodPost:
		requestBody, parseError := readAuthLoginRequest(r)
		if parseError != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid login request"})
			return
		}

		admin, issuedSession, authError := handler.authService.AuthenticateAdmin(r.Context(), requestBody.Username, requestBody.Password, requestIP(r), r.UserAgent())
		if authError != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
			return
		}

		http.SetCookie(w, handler.adminCookie.Build(r, issuedSession.Token, issuedSession.Session.ExpiresAt))
		writeJSON(w, http.StatusOK, map[string]any{
			"status":      "authenticated",
			"username":    admin.Username,
			"session_id":  issuedSession.Session.ID,
			"expires_at":  issuedSession.Session.ExpiresAt.UTC().Format(time.RFC3339),
			"rehash_hint": issuedSession.RehashHint,
		})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (handler AdminHandler) handleProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	session, admin, authorized := handler.resolveAdmin(r)
	if !authorized {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "admin authentication required"})
		return
	}

	if prefersJSON(r.Header.Get("Accept")) {
		writeJSON(w, http.StatusOK, map[string]any{
			"username":      admin.Username,
			"role":          admin.Role,
			"account_email": admin.AccountEmail,
			"session_id":    session.ID,
			"expires_at":    session.ExpiresAt.UTC().Format(time.RFC3339),
		})
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "Admin: %s\nRole: %s\nSession expires: %s\n", admin.Username, admin.Role, session.ExpiresAt.UTC().Format(time.RFC3339))
}

func (handler AdminHandler) handleProtectedSurface(w http.ResponseWriter, r *http.Request, relativePath string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	session, admin, authorized := handler.resolveAdmin(r)
	if !authorized {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "admin authentication required"})
		return
	}

	if prefersJSON(r.Header.Get("Accept")) {
		writeJSON(w, http.StatusOK, map[string]any{
			"surface":      relativePath,
			"admin":        admin.Username,
			"session_id":   session.ID,
			"session_kind": session.Kind,
		})
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "Admin surface: %s\nAdmin: %s\n", relativePath, admin.Username)
}

func (handler AdminHandler) resolveAdmin(r *http.Request) (model.Session, model.Admin, bool) {
	sessionToken, tokenError := readSessionCookie(r, handler.adminCookie.Name)
	if tokenError != nil {
		return model.Session{}, model.Admin{}, false
	}

	session, resolveError := handler.authService.ResolveAdminSession(r.Context(), sessionToken)
	if resolveError != nil {
		return model.Session{}, model.Admin{}, false
	}

	admin, adminError := handler.authService.FindAdminByID(r.Context(), session.SubjectID)
	if adminError != nil {
		return model.Session{}, model.Admin{}, false
	}

	return session, admin, true
}
