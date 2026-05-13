package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/casapps/caspbx/src/server/service"
)

type UserHandler struct {
	routePrefix          string
	authService          service.AuthService
	domainService        service.DomainService
	communicationService service.UserCommunicationsService
	userCookie           SessionCookieConfig
}

func NewUserHandler(routePrefix string, authService service.AuthService, domainService service.DomainService, userCookie SessionCookieConfig, communicationService ...service.UserCommunicationsService) http.Handler {
	var resolvedCommunicationService service.UserCommunicationsService
	if len(communicationService) > 0 {
		resolvedCommunicationService = communicationService[0]
	}
	return UserHandler{
		routePrefix:          routePrefix,
		authService:          authService,
		domainService:        domainService,
		communicationService: resolvedCommunicationService,
		userCookie:           userCookie,
	}
}

func (handler UserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	relativePath := routeTail(handler.routePrefix, r.URL.Path)
	if (relativePath == "" || relativePath == "profile") && r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	sessionToken, tokenError := readSessionCookie(r, handler.userCookie.Name)
	if tokenError != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}

	session, resolveError := handler.authService.ResolveUserSession(r.Context(), sessionToken)
	if resolveError != nil {
		http.SetCookie(w, handler.userCookie.Clear(r))
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}

	user, userError := handler.authService.FindUserByID(r.Context(), session.SubjectID)
	if userError != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}

	switch relativePath {
	case "", "profile":
		if prefersJSON(r.Header.Get("Accept")) {
			writeJSON(w, http.StatusOK, map[string]any{
				"username":      user.Username,
				"display_name":  user.DisplayName,
				"account_email": user.AccountEmail,
				"visibility":    user.Visibility,
				"session_id":    session.ID,
				"expires_at":    session.ExpiresAt.UTC().Format(time.RFC3339),
			})
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "User: %s\nEmail: %s\nSession expires: %s\n", user.Username, user.AccountEmail, session.ExpiresAt.UTC().Format(time.RFC3339))
	default:
		if route, ok := parseUserCommunicationRoute(relativePath); ok {
			handler.handleUserCommunicationSurface(w, r, route, user, session.ID)
			return
		}
		if relativePath == "domains" || strings.HasPrefix(relativePath, "domains/") {
			handler.handleDomainSurface(w, r, user)
			return
		}
		http.NotFound(w, r)
	}
}
