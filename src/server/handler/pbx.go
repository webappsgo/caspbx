package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/casapps/caspbx/src/server/model"
	"github.com/casapps/caspbx/src/server/service"
)

type pbxRoute struct {
	resource string
	id       int64
	hasID    bool
}

type extensionRequest struct {
	Number           string `json:"number"`
	DisplayName      string `json:"display_name"`
	Technology       string `json:"technology"`
	Endpoint         string `json:"endpoint"`
	VoicemailEnabled bool   `json:"voicemail_enabled"`
}

type trunkRequest struct {
	Name       string `json:"name"`
	Technology string `json:"technology"`
	Endpoint   string `json:"endpoint"`
	Active     bool   `json:"active"`
}

type routeRequest struct {
	Name        string `json:"name"`
	Direction   string `json:"direction"`
	Match       string `json:"match"`
	Destination string `json:"destination"`
}

type queueRequest struct {
	Name                   string   `json:"name"`
	Strategy               string   `json:"strategy"`
	MemberExtensionNumbers []string `json:"member_extension_numbers"`
}

type conferenceRequest struct {
	Name             string `json:"name"`
	AccessCode       string `json:"access_code"`
	RecordingEnabled bool   `json:"recording_enabled"`
}

type ivrRequest struct {
	Name               string `json:"name"`
	RootPrompt         string `json:"root_prompt"`
	DefaultDestination string `json:"default_destination"`
	TimeoutSeconds     int    `json:"timeout_seconds"`
}

type promptAssignmentRequest struct {
	Name       string `json:"name"`
	PromptName string `json:"prompt_name"`
	TargetKind string `json:"target_kind"`
	TargetRef  string `json:"target_ref"`
}

type provisioningProfileRequest struct {
	Name               string   `json:"name"`
	Technology         string   `json:"technology"`
	Template           string   `json:"template"`
	AssignedExtensions []string `json:"assigned_extensions"`
}

func parsePBXRoute(relativePath string) (pbxRoute, bool) {
	normalizedPath := strings.Trim(strings.TrimSpace(relativePath), "/")
	if normalizedPath == "" {
		return pbxRoute{}, false
	}
	parts := strings.Split(normalizedPath, "/")
	switch parts[0] {
	case "extensions", "trunks", "routes", "queues", "conferences", "ivrs", "prompt-assignments", "provisioning-profiles":
		if len(parts) == 1 {
			return pbxRoute{resource: parts[0]}, true
		}
		if len(parts) == 2 {
			id, err := strconv.ParseInt(parts[1], 10, 64)
			if err == nil && id > 0 {
				return pbxRoute{resource: parts[0], id: id, hasID: true}, true
			}
		}
	case "apply-preview":
		if len(parts) == 1 {
			return pbxRoute{resource: parts[0]}, true
		}
	}
	return pbxRoute{}, false
}

func writePBXServiceError(w http.ResponseWriter, responseError error) {
	pbxError, isPBXError := responseError.(*service.PBXError)
	if !isPBXError {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	switch pbxError.Code {
	case "PBX_INVALID":
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": pbxError.Message, "code": pbxError.Code})
	case "PBX_NOT_FOUND":
		writeJSON(w, http.StatusNotFound, map[string]string{"error": pbxError.Message, "code": pbxError.Code})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": pbxError.Message, "code": pbxError.Code})
	}
}

func (handler AdminHandler) handlePBXSurface(w http.ResponseWriter, r *http.Request, route pbxRoute, admin string, sessionID string) {
	switch route.resource {
	case "apply-preview":
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		preview, err := handler.pbxService.ApplyPreview(r.Context())
		if err != nil {
			writePBXServiceError(w, err)
			return
		}
		if prefersJSON(r.Header.Get("Accept")) {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "admin": admin, "session_id": sessionID, "data": preview})
			return
		}
		writePBXPreviewText(w, admin, preview)
		return
	default:
		handler.handlePBXEntitySurface(w, r, route, admin, sessionID)
	}
}

func (handler APIAdminHandler) handlePBXSurface(w http.ResponseWriter, r *http.Request, route pbxRoute, admin string, tokenPrefix string, tokenScope model.TokenScope) {
	switch route.resource {
	case "apply-preview":
		if !allowsReadOnlyMethod(w, r) {
			return
		}
		preview, err := handler.pbxService.ApplyPreview(r.Context())
		if err != nil {
			writePBXServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "admin": admin, "token_prefix": tokenPrefix, "token_scope": tokenScope, "data": preview})
		return
	default:
		handler.handlePBXEntitySurface(w, r, route, admin, tokenPrefix)
	}
}

func (handler AdminHandler) handlePBXEntitySurface(w http.ResponseWriter, r *http.Request, route pbxRoute, admin string, sessionID string) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		data, err := handler.lookupPBXResource(r, route)
		if err != nil {
			writePBXServiceError(w, err)
			return
		}
		if prefersJSON(r.Header.Get("Accept")) {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "admin": admin, "session_id": sessionID, "resource": route.resource, "data": data})
			return
		}
		writePBXEntityText(w, admin, route.resource, data)
	case http.MethodPost:
		if route.hasID {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		data, err := handler.createPBXResource(r, route.resource)
		if err != nil {
			writePBXServiceError(w, err)
			return
		}
		if prefersJSON(r.Header.Get("Accept")) {
			writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "admin": admin, "session_id": sessionID, "resource": route.resource, "data": data})
			return
		}
		writePBXEntityText(w, admin, route.resource, data)
	case http.MethodDelete:
		if !route.hasID {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if err := handler.deletePBXResource(r, route); err != nil {
			writePBXServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "resource": route.resource, "id": route.id, "status": "deleted"})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (handler APIAdminHandler) handlePBXEntitySurface(w http.ResponseWriter, r *http.Request, route pbxRoute, admin string, tokenPrefix string) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		data, err := handler.lookupPBXResource(r, route)
		if err != nil {
			writePBXServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "admin": admin, "token_prefix": tokenPrefix, "resource": route.resource, "data": data})
	case http.MethodPost:
		if route.hasID {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		data, err := handler.createPBXResource(r, route.resource)
		if err != nil {
			writePBXServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "admin": admin, "token_prefix": tokenPrefix, "resource": route.resource, "data": data})
	case http.MethodDelete:
		if !route.hasID {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if err := handler.deletePBXResource(r, route); err != nil {
			writePBXServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "resource": route.resource, "id": route.id, "status": "deleted"})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (handler AdminHandler) lookupPBXResource(r *http.Request, route pbxRoute) (any, error) {
	return lookupPBXResource(r.Context(), handler.pbxService, route)
}

func (handler APIAdminHandler) lookupPBXResource(r *http.Request, route pbxRoute) (any, error) {
	return lookupPBXResource(r.Context(), handler.pbxService, route)
}

func lookupPBXResource(ctx context.Context, pbxService service.PBXService, route pbxRoute) (any, error) {
	switch route.resource {
	case "extensions":
		if route.hasID {
			return pbxService.GetExtension(ctx, route.id)
		}
		return pbxService.ListExtensions(ctx)
	case "trunks":
		if route.hasID {
			return pbxService.GetTrunk(ctx, route.id)
		}
		return pbxService.ListTrunks(ctx)
	case "routes":
		if route.hasID {
			return pbxService.GetRoute(ctx, route.id)
		}
		return pbxService.ListRoutes(ctx)
	case "queues":
		if route.hasID {
			return pbxService.GetQueue(ctx, route.id)
		}
		return pbxService.ListQueues(ctx)
	case "conferences":
		if route.hasID {
			return pbxService.GetConference(ctx, route.id)
		}
		return pbxService.ListConferences(ctx)
	case "ivrs":
		if route.hasID {
			return pbxService.GetIVR(ctx, route.id)
		}
		return pbxService.ListIVRs(ctx)
	case "prompt-assignments":
		if route.hasID {
			return pbxService.GetPromptAssignment(ctx, route.id)
		}
		return pbxService.ListPromptAssignments(ctx)
	case "provisioning-profiles":
		if route.hasID {
			return pbxService.GetProvisioningProfile(ctx, route.id)
		}
		return pbxService.ListProvisioningProfiles(ctx)
	default:
		return nil, &service.PBXError{Code: "PBX_NOT_FOUND", Message: "resource not found"}
	}
}

func (handler AdminHandler) createPBXResource(r *http.Request, resource string) (any, error) {
	return createPBXResource(r, handler.pbxService, resource)
}

func (handler APIAdminHandler) createPBXResource(r *http.Request, resource string) (any, error) {
	return createPBXResource(r, handler.pbxService, resource)
}

func createPBXResource(r *http.Request, pbxService service.PBXService, resource string) (any, error) {
	switch resource {
	case "extensions":
		request, err := readExtensionRequest(r)
		if err != nil {
			return nil, &service.PBXError{Code: "PBX_INVALID", Message: "invalid extension request"}
		}
		return pbxService.CreateExtension(r.Context(), request)
	case "trunks":
		request, err := readTrunkRequest(r)
		if err != nil {
			return nil, &service.PBXError{Code: "PBX_INVALID", Message: "invalid trunk request"}
		}
		return pbxService.CreateTrunk(r.Context(), request)
	case "routes":
		request, err := readRouteRequest(r)
		if err != nil {
			return nil, &service.PBXError{Code: "PBX_INVALID", Message: "invalid route request"}
		}
		return pbxService.CreateRoute(r.Context(), request)
	case "queues":
		request, err := readQueueRequest(r)
		if err != nil {
			return nil, &service.PBXError{Code: "PBX_INVALID", Message: "invalid queue request"}
		}
		return pbxService.CreateQueue(r.Context(), request)
	case "conferences":
		request, err := readConferenceRequest(r)
		if err != nil {
			return nil, &service.PBXError{Code: "PBX_INVALID", Message: "invalid conference request"}
		}
		return pbxService.CreateConference(r.Context(), request)
	case "ivrs":
		request, err := readIVRRequest(r)
		if err != nil {
			return nil, &service.PBXError{Code: "PBX_INVALID", Message: "invalid ivr request"}
		}
		return pbxService.CreateIVR(r.Context(), request)
	case "prompt-assignments":
		request, err := readPromptAssignmentRequest(r)
		if err != nil {
			return nil, &service.PBXError{Code: "PBX_INVALID", Message: "invalid prompt assignment request"}
		}
		return pbxService.CreatePromptAssignment(r.Context(), request)
	case "provisioning-profiles":
		request, err := readProvisioningProfileRequest(r)
		if err != nil {
			return nil, &service.PBXError{Code: "PBX_INVALID", Message: "invalid provisioning profile request"}
		}
		return pbxService.CreateProvisioningProfile(r.Context(), request)
	default:
		return nil, &service.PBXError{Code: "PBX_NOT_FOUND", Message: "resource not found"}
	}
}

func (handler AdminHandler) deletePBXResource(r *http.Request, route pbxRoute) error {
	return deletePBXResource(r, handler.pbxService, route)
}

func (handler APIAdminHandler) deletePBXResource(r *http.Request, route pbxRoute) error {
	return deletePBXResource(r, handler.pbxService, route)
}

func deletePBXResource(r *http.Request, pbxService service.PBXService, route pbxRoute) error {
	switch route.resource {
	case "extensions":
		return pbxService.DeleteExtension(r.Context(), route.id)
	case "trunks":
		return pbxService.DeleteTrunk(r.Context(), route.id)
	case "routes":
		return pbxService.DeleteRoute(r.Context(), route.id)
	case "queues":
		return pbxService.DeleteQueue(r.Context(), route.id)
	case "conferences":
		return pbxService.DeleteConference(r.Context(), route.id)
	case "ivrs":
		return pbxService.DeleteIVR(r.Context(), route.id)
	case "prompt-assignments":
		return pbxService.DeletePromptAssignment(r.Context(), route.id)
	case "provisioning-profiles":
		return pbxService.DeleteProvisioningProfile(r.Context(), route.id)
	default:
		return &service.PBXError{Code: "PBX_NOT_FOUND", Message: "resource not found"}
	}
}

func writePBXEntityText(w http.ResponseWriter, admin string, resource string, data any) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "PBX resource: %s\nAdmin: %s\n", resource, admin)
	switch value := data.(type) {
	case []model.Extension:
		fmt.Fprintf(w, "Count: %d\n", len(value))
		for _, entity := range value {
			fmt.Fprintf(w, "- %d %s %s [%s]\n", entity.ID, entity.Number, entity.DisplayName, entity.Technology)
		}
	case model.Extension:
		fmt.Fprintf(w, "ID: %d\nNumber: %s\nDisplay name: %s\nTechnology: %s\n", value.ID, value.Number, value.DisplayName, value.Technology)
	case []model.Trunk:
		fmt.Fprintf(w, "Count: %d\n", len(value))
		for _, entity := range value {
			fmt.Fprintf(w, "- %d %s [%s]\n", entity.ID, entity.Name, entity.Technology)
		}
	case model.Trunk:
		fmt.Fprintf(w, "ID: %d\nName: %s\nTechnology: %s\nEndpoint: %s\n", value.ID, value.Name, value.Technology, value.Endpoint)
	case []model.CallRoute:
		fmt.Fprintf(w, "Count: %d\n", len(value))
		for _, entity := range value {
			fmt.Fprintf(w, "- %d %s [%s]\n", entity.ID, entity.Name, entity.Direction)
		}
	case model.CallRoute:
		fmt.Fprintf(w, "ID: %d\nName: %s\nDirection: %s\nDestination: %s\n", value.ID, value.Name, value.Direction, value.Destination)
	case []model.Queue:
		fmt.Fprintf(w, "Count: %d\n", len(value))
		for _, entity := range value {
			fmt.Fprintf(w, "- %d %s\n", entity.ID, entity.Name)
		}
	case model.Queue:
		fmt.Fprintf(w, "ID: %d\nName: %s\nStrategy: %s\n", value.ID, value.Name, value.Strategy)
	case []model.Conference:
		fmt.Fprintf(w, "Count: %d\n", len(value))
		for _, entity := range value {
			fmt.Fprintf(w, "- %d %s\n", entity.ID, entity.Name)
		}
	case model.Conference:
		fmt.Fprintf(w, "ID: %d\nName: %s\nAccess code: %s\n", value.ID, value.Name, value.AccessCode)
	case []model.IVR:
		fmt.Fprintf(w, "Count: %d\n", len(value))
		for _, entity := range value {
			fmt.Fprintf(w, "- %d %s\n", entity.ID, entity.Name)
		}
	case model.IVR:
		fmt.Fprintf(w, "ID: %d\nName: %s\nDefault destination: %s\n", value.ID, value.Name, value.DefaultDestination)
	case []model.PromptAssignment:
		fmt.Fprintf(w, "Count: %d\n", len(value))
		for _, entity := range value {
			fmt.Fprintf(w, "- %d %s -> %s\n", entity.ID, entity.Name, entity.TargetKind)
		}
	case model.PromptAssignment:
		fmt.Fprintf(w, "ID: %d\nName: %s\nPrompt: %s\nTarget: %s %s\n", value.ID, value.Name, value.PromptName, value.TargetKind, value.TargetRef)
	case []model.ProvisioningProfile:
		fmt.Fprintf(w, "Count: %d\n", len(value))
		for _, entity := range value {
			fmt.Fprintf(w, "- %d %s [%s]\n", entity.ID, entity.Name, entity.Technology)
		}
	case model.ProvisioningProfile:
		fmt.Fprintf(w, "ID: %d\nName: %s\nTechnology: %s\nTemplate: %s\n", value.ID, value.Name, value.Technology, value.Template)
	}
}

func writePBXPreviewText(w http.ResponseWriter, admin string, preview service.PBXApplyPreview) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "PBX resource: apply-preview\nAdmin: %s\nUpdated: %s\n", admin, preview.UpdatedAt)
	fmt.Fprintln(w, "Summaries:")
	for _, summary := range preview.Summaries {
		fmt.Fprintf(w, "- %s: %d\n", summary.Resource, summary.Count)
	}
	if len(preview.Artifacts) > 0 {
		fmt.Fprintln(w, "Artifacts:")
		for _, artifact := range preview.Artifacts {
			fmt.Fprintf(w, "- %s [%s]: %s\n", artifact.Label, artifact.Status, artifact.Summary)
		}
	}
	if len(preview.Validations) > 0 {
		fmt.Fprintln(w, "Validations:")
		for _, validation := range preview.Validations {
			fmt.Fprintf(w, "- %s\n", validation)
		}
	}
	if len(preview.Actions) > 0 {
		fmt.Fprintln(w, "Actions:")
		for _, action := range preview.Actions {
			fmt.Fprintf(w, "- %s\n", action)
		}
	}
}

func readExtensionRequest(r *http.Request) (model.Extension, error) {
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		var request extensionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			return model.Extension{}, err
		}
		return model.Extension{
			Number:           request.Number,
			DisplayName:      request.DisplayName,
			Technology:       request.Technology,
			Endpoint:         request.Endpoint,
			VoicemailEnabled: request.VoicemailEnabled,
		}, nil
	}
	if err := r.ParseForm(); err != nil {
		return model.Extension{}, err
	}
	return model.Extension{
		Number:           r.FormValue("number"),
		DisplayName:      r.FormValue("display_name"),
		Technology:       r.FormValue("technology"),
		Endpoint:         r.FormValue("endpoint"),
		VoicemailEnabled: r.FormValue("voicemail_enabled") == "true",
	}, nil
}

func readTrunkRequest(r *http.Request) (model.Trunk, error) {
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		var request trunkRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			return model.Trunk{}, err
		}
		return model.Trunk{
			Name:       request.Name,
			Technology: request.Technology,
			Endpoint:   request.Endpoint,
			Active:     request.Active,
		}, nil
	}
	if err := r.ParseForm(); err != nil {
		return model.Trunk{}, err
	}
	return model.Trunk{
		Name:       r.FormValue("name"),
		Technology: r.FormValue("technology"),
		Endpoint:   r.FormValue("endpoint"),
		Active:     r.FormValue("active") != "false",
	}, nil
}

func readRouteRequest(r *http.Request) (model.CallRoute, error) {
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		var request routeRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			return model.CallRoute{}, err
		}
		return model.CallRoute{
			Name:        request.Name,
			Direction:   request.Direction,
			Match:       request.Match,
			Destination: request.Destination,
		}, nil
	}
	if err := r.ParseForm(); err != nil {
		return model.CallRoute{}, err
	}
	return model.CallRoute{
		Name:        r.FormValue("name"),
		Direction:   r.FormValue("direction"),
		Match:       r.FormValue("match"),
		Destination: r.FormValue("destination"),
	}, nil
}

func readQueueRequest(r *http.Request) (model.Queue, error) {
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		var request queueRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			return model.Queue{}, err
		}
		return model.Queue{
			Name:                   request.Name,
			Strategy:               request.Strategy,
			MemberExtensionNumbers: request.MemberExtensionNumbers,
		}, nil
	}
	if err := r.ParseForm(); err != nil {
		return model.Queue{}, err
	}
	return model.Queue{
		Name:                   r.FormValue("name"),
		Strategy:               r.FormValue("strategy"),
		MemberExtensionNumbers: splitCSV(r.FormValue("member_extension_numbers")),
	}, nil
}

func readConferenceRequest(r *http.Request) (model.Conference, error) {
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		var request conferenceRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			return model.Conference{}, err
		}
		return model.Conference{
			Name:             request.Name,
			AccessCode:       request.AccessCode,
			RecordingEnabled: request.RecordingEnabled,
		}, nil
	}
	if err := r.ParseForm(); err != nil {
		return model.Conference{}, err
	}
	return model.Conference{
		Name:             r.FormValue("name"),
		AccessCode:       r.FormValue("access_code"),
		RecordingEnabled: r.FormValue("recording_enabled") == "true",
	}, nil
}

func readIVRRequest(r *http.Request) (model.IVR, error) {
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		var request ivrRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			return model.IVR{}, err
		}
		return model.IVR{
			Name:               request.Name,
			RootPrompt:         request.RootPrompt,
			DefaultDestination: request.DefaultDestination,
			TimeoutSeconds:     request.TimeoutSeconds,
		}, nil
	}
	if err := r.ParseForm(); err != nil {
		return model.IVR{}, err
	}
	timeout, _ := strconv.Atoi(r.FormValue("timeout_seconds"))
	return model.IVR{
		Name:               r.FormValue("name"),
		RootPrompt:         r.FormValue("root_prompt"),
		DefaultDestination: r.FormValue("default_destination"),
		TimeoutSeconds:     timeout,
	}, nil
}

func readPromptAssignmentRequest(r *http.Request) (model.PromptAssignment, error) {
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		var request promptAssignmentRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			return model.PromptAssignment{}, err
		}
		return model.PromptAssignment{
			Name:       request.Name,
			PromptName: request.PromptName,
			TargetKind: request.TargetKind,
			TargetRef:  request.TargetRef,
		}, nil
	}
	if err := r.ParseForm(); err != nil {
		return model.PromptAssignment{}, err
	}
	return model.PromptAssignment{
		Name:       r.FormValue("name"),
		PromptName: r.FormValue("prompt_name"),
		TargetKind: r.FormValue("target_kind"),
		TargetRef:  r.FormValue("target_ref"),
	}, nil
}

func readProvisioningProfileRequest(r *http.Request) (model.ProvisioningProfile, error) {
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		var request provisioningProfileRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			return model.ProvisioningProfile{}, err
		}
		return model.ProvisioningProfile{
			Name:               request.Name,
			Technology:         request.Technology,
			Template:           request.Template,
			AssignedExtensions: request.AssignedExtensions,
		}, nil
	}
	if err := r.ParseForm(); err != nil {
		return model.ProvisioningProfile{}, err
	}
	return model.ProvisioningProfile{
		Name:               r.FormValue("name"),
		Technology:         r.FormValue("technology"),
		Template:           r.FormValue("template"),
		AssignedExtensions: splitCSV(r.FormValue("assigned_extensions")),
	}, nil
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}
