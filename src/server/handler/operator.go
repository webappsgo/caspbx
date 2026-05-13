package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/casapps/caspbx/src/server/model"
	"github.com/casapps/caspbx/src/server/service"
)

type operatorRoute struct {
	surface string
	preview bool
}

type supervisorActionRequest struct {
	Action     string `json:"action"`
	TargetKind string `json:"target_kind"`
	TargetRef  string `json:"target_ref"`
}

func parseOperatorRoute(relativePath string) (operatorRoute, bool) {
	normalizedPath := strings.Trim(strings.TrimSpace(relativePath), "/")
	switch normalizedPath {
	case "operator", "operator/trunks", "operator/conferences", "operator/parked-calls", "callcenter", "callcenter/queues", "callcenter/agents", "callcenter/supervisor-actions":
		return operatorRoute{surface: normalizedPath}, true
	case "callcenter/supervisor-actions/preview":
		return operatorRoute{surface: "callcenter/supervisor-actions", preview: true}, true
	default:
		return operatorRoute{}, false
	}
}

func writeOperatorServiceError(w http.ResponseWriter, responseError error) {
	operatorError, ok := responseError.(*service.OperatorError)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	switch operatorError.Code {
	case "OPERATOR_INVALID":
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": operatorError.Message, "code": operatorError.Code})
	case "OPERATOR_UNAVAILABLE":
		writeJSON(w, http.StatusNotFound, map[string]string{"error": operatorError.Message, "code": operatorError.Code})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": operatorError.Message, "code": operatorError.Code})
	}
}

func (handler AdminHandler) handleOperatorSurface(w http.ResponseWriter, r *http.Request, route operatorRoute, admin string, sessionID string) {
	if route.preview {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		requestBody, err := readSupervisorActionPreviewRequest(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid supervisor action preview request"})
			return
		}
		data, err := handler.operatorService.PreviewSupervisorAction(r.Context(), requestBody)
		if err != nil {
			writeOperatorServiceError(w, err)
			return
		}
		if prefersJSON(r.Header.Get("Accept")) {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "admin": admin, "session_id": sessionID, "surface": route.surface, "data": data})
			return
		}
		writeOperatorTextSurface(w, admin, route.surface, data)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	data, err := lookupOperatorSurface(r.Context(), handler.operatorService, route)
	if err != nil {
		writeOperatorServiceError(w, err)
		return
	}
	if prefersJSON(r.Header.Get("Accept")) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "admin": admin, "session_id": sessionID, "surface": route.surface, "data": data})
		return
	}
	writeOperatorTextSurface(w, admin, route.surface, data)
}

func (handler APIAdminHandler) handleOperatorSurface(w http.ResponseWriter, r *http.Request, route operatorRoute, admin string, tokenPrefix string, tokenScope model.TokenScope) {
	if route.preview {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		requestBody, err := readSupervisorActionPreviewRequest(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid supervisor action preview request"})
			return
		}
		data, err := handler.operatorService.PreviewSupervisorAction(r.Context(), requestBody)
		if err != nil {
			writeOperatorServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "admin": admin, "token_prefix": tokenPrefix, "token_scope": tokenScope, "surface": route.surface, "data": data})
		return
	}
	if !allowsReadOnlyMethod(w, r) {
		return
	}
	data, err := lookupOperatorSurface(r.Context(), handler.operatorService, route)
	if err != nil {
		writeOperatorServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "admin": admin, "token_prefix": tokenPrefix, "token_scope": tokenScope, "surface": route.surface, "data": data})
}

func lookupOperatorSurface(ctx context.Context, operatorService service.OperatorService, route operatorRoute) (any, error) {
	switch route.surface {
	case "operator":
		return operatorService.Dashboard(ctx)
	case "operator/trunks":
		return operatorService.Trunks(ctx)
	case "operator/conferences":
		return operatorService.Conferences(ctx)
	case "operator/parked-calls":
		return operatorService.Parking(ctx)
	case "callcenter":
		return operatorService.Dashboard(ctx)
	case "callcenter/queues":
		return operatorService.Queues(ctx)
	case "callcenter/agents":
		return operatorService.Agents(ctx)
	case "callcenter/supervisor-actions":
		return operatorService.SupervisorActions(ctx)
	default:
		return nil, &service.OperatorError{Code: "OPERATOR_UNAVAILABLE", Message: "operator surface is not available"}
	}
}

func readSupervisorActionPreviewRequest(r *http.Request) (service.SupervisorActionRequest, error) {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type"))), "application/json") {
		defer r.Body.Close()
		var requestBody supervisorActionRequest
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			return service.SupervisorActionRequest{}, err
		}
		return service.SupervisorActionRequest{
			Action:     requestBody.Action,
			TargetKind: requestBody.TargetKind,
			TargetRef:  requestBody.TargetRef,
		}, nil
	}
	if err := r.ParseForm(); err != nil {
		return service.SupervisorActionRequest{}, err
	}
	return service.SupervisorActionRequest{
		Action:     firstNonEmpty(r.FormValue("action")),
		TargetKind: firstNonEmpty(r.FormValue("target_kind"), r.FormValue("targetKind")),
		TargetRef:  firstNonEmpty(r.FormValue("target_ref"), r.FormValue("targetRef")),
	}, nil
}

func writeOperatorTextSurface(w http.ResponseWriter, admin string, surface string, data any) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "Operator surface: %s\nAdmin: %s\nData: %+v\n", surface, admin, data)
}
