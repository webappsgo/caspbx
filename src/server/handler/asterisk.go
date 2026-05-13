package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/casapps/caspbx/src/server/service"
)

func (handler AdminHandler) handleAsteriskSurface(w http.ResponseWriter, r *http.Request, relativePath string) {
	session, admin, authorized := handler.resolveAdmin(r)
	if !authorized {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "admin authentication required"})
		return
	}

	surfaceKey, found := normalizeAsteriskRoute(handler.routePrefix, relativePath)
	if !found {
		http.NotFound(w, r)
		return
	}
	if route, isPBXRoute := parsePBXRoute(surfaceKey); isPBXRoute {
		handler.handlePBXSurface(w, r, route, admin.Username, session.ID)
		return
	}
	if route, isOperatorRoute := parseOperatorRoute(surfaceKey); isOperatorRoute {
		handler.handleOperatorSurface(w, r, route, admin.Username, session.ID)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	surface, surfaceError := handler.asteriskService.Surface(r.Context(), surfaceKey)
	if surfaceError != nil {
		writeAsteriskServiceError(w, surfaceError)
		return
	}

	if prefersJSON(r.Header.Get("Accept")) {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":         true,
			"admin":      admin.Username,
			"session_id": session.ID,
			"data":       surface,
		})
		return
	}

	writeAsteriskTextSurface(w, surface, admin.Username)
}

func (handler APIAdminHandler) handleAsteriskSurface(w http.ResponseWriter, r *http.Request, relativePath string) {
	tokenRecord, admin, authorized := handler.resolveAdmin(r)
	if !authorized {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "admin authentication required"})
		return
	}

	surfaceKey, found := normalizeAsteriskRoute(handler.routePrefix, relativePath)
	if !found {
		http.NotFound(w, r)
		return
	}
	if route, isPBXRoute := parsePBXRoute(surfaceKey); isPBXRoute {
		handler.handlePBXSurface(w, r, route, admin.Username, tokenRecord.TokenPrefix, tokenRecord.Scope)
		return
	}
	if route, isOperatorRoute := parseOperatorRoute(surfaceKey); isOperatorRoute {
		handler.handleOperatorSurface(w, r, route, admin.Username, tokenRecord.TokenPrefix, tokenRecord.Scope)
		return
	}
	if !allowsReadOnlyMethod(w, r) {
		return
	}

	surface, surfaceError := handler.asteriskService.Surface(r.Context(), surfaceKey)
	if surfaceError != nil {
		writeAsteriskServiceError(w, surfaceError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":           true,
		"admin":        admin.Username,
		"token_prefix": tokenRecord.TokenPrefix,
		"token_scope":  tokenRecord.Scope,
		"data":         surface,
	})
}

func normalizeAsteriskRoute(routePrefix string, relativePath string) (string, bool) {
	normalizedPath := strings.Trim(strings.TrimSpace(relativePath), "/")
	if strings.Contains(routePrefix, "/asterisk") {
		return normalizedPath, true
	}
	if normalizedPath == "server/asterisk" {
		return "", true
	}
	if strings.HasPrefix(normalizedPath, "server/asterisk/") {
		return strings.TrimPrefix(normalizedPath, "server/asterisk/"), true
	}
	return "", false
}

func writeAsteriskServiceError(w http.ResponseWriter, responseError error) {
	asteriskError, isAsteriskError := responseError.(*service.AsteriskError)
	if !isAsteriskError {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	if asteriskError.Code == "ASTERISK_SURFACE_NOT_FOUND" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": asteriskError.Message, "code": asteriskError.Code})
		return
	}
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": asteriskError.Message, "code": asteriskError.Code})
}

func writeAsteriskTextSurface(w http.ResponseWriter, surface service.AsteriskSurfaceView, admin string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "Asterisk surface: %s\nAdmin: %s\nSummary: %s\nDetection: %s\nHealth: %s\n", surface.Surface.Label, admin, surface.Summary, surface.DetectionStatus, surface.HealthStatus)
	if len(surface.Items) > 0 {
		fmt.Fprintln(w, "Items:")
		for _, item := range surface.Items {
			fmt.Fprintf(w, "- %s [%s]", item.Label, item.Status)
			if item.Value != "" {
				fmt.Fprintf(w, ": %s", item.Value)
			}
			if item.Detail != "" {
				fmt.Fprintf(w, " (%s)", item.Detail)
			}
			fmt.Fprintln(w)
		}
	}
	if len(surface.AvailableSurfaces) > 0 {
		fmt.Fprintln(w, "Visible surfaces:")
		for _, availableSurface := range surface.AvailableSurfaces {
			fmt.Fprintf(w, "- %s (%s)\n", availableSurface.Label, availableSurface.Key)
		}
	}
}
