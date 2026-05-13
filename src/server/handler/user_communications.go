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

type userCommunicationRoute struct {
	surface string
	id      int64
	hasID   bool
}

type userContactRequest struct {
	DisplayName     string `json:"display_name"`
	ExtensionNumber string `json:"extension_number"`
	PhoneNumber     string `json:"phone_number"`
	Email           string `json:"email"`
	Favorite        bool   `json:"favorite"`
}

type userCommunicationSettingsRequest struct {
	DoNotDisturb          bool   `json:"do_not_disturb"`
	CallForwardingTarget  string `json:"call_forwarding_target"`
	VoicemailEnabled      bool   `json:"voicemail_enabled"`
	WebphoneEnabled       bool   `json:"webphone_enabled"`
	PresenceEnabled       bool   `json:"presence_enabled"`
	MessagingEnabled      bool   `json:"messaging_enabled"`
	PreferredEndpoint     string `json:"preferred_endpoint"`
	PreferredContactEmail string `json:"preferred_contact_email"`
}

func parseUserCommunicationRoute(relativePath string) (userCommunicationRoute, bool) {
	normalized := strings.Trim(strings.TrimSpace(relativePath), "/")
	switch normalized {
	case "dashboard", "call-history", "voicemail", "messages", "presence", "webphone", "communications/settings":
		return userCommunicationRoute{surface: normalized}, true
	}
	if normalized == "contacts" {
		return userCommunicationRoute{surface: "contacts"}, true
	}
	if strings.HasPrefix(normalized, "contacts/") {
		id, err := strconv.ParseInt(strings.TrimPrefix(normalized, "contacts/"), 10, 64)
		if err == nil && id > 0 {
			return userCommunicationRoute{surface: "contacts", id: id, hasID: true}, true
		}
	}
	return userCommunicationRoute{}, false
}

func writeUserCommunicationsError(w http.ResponseWriter, responseError error) {
	communicationError, ok := responseError.(*service.UserCommunicationsError)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}
	switch communicationError.Code {
	case "COMMUNICATION_INVALID":
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": communicationError.Message, "code": communicationError.Code})
	case "COMMUNICATION_NOT_FOUND", "COMMUNICATION_UNAVAILABLE":
		writeJSON(w, http.StatusNotFound, map[string]string{"error": communicationError.Message, "code": communicationError.Code})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": communicationError.Message, "code": communicationError.Code})
	}
}

func (handler UserHandler) handleUserCommunicationSurface(w http.ResponseWriter, r *http.Request, route userCommunicationRoute, user model.User, sessionID string) {
	switch route.surface {
	case "contacts":
		handler.handleUserContactSurface(w, r, route, user, sessionID)
	case "communications/settings":
		handler.handleUserSettingsSurface(w, r, user, sessionID)
	default:
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		data, err := lookupUserCommunicationSurface(r.Context(), handler.communicationService, route, user.ID)
		if err != nil {
			writeUserCommunicationsError(w, err)
			return
		}
		if prefersJSON(r.Header.Get("Accept")) {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "session_id": sessionID, "user": user.Username, "surface": route.surface, "data": data})
			return
		}
		writeUserCommunicationText(w, user.Username, route.surface, data)
	}
}

func (handler APIUserHandler) handleUserCommunicationSurface(w http.ResponseWriter, r *http.Request, route userCommunicationRoute, user model.User, tokenPrefix string, tokenScope model.TokenScope) {
	switch route.surface {
	case "contacts":
		handler.handleUserContactSurface(w, r, route, user, tokenPrefix, tokenScope)
	case "communications/settings":
		handler.handleUserSettingsSurface(w, r, user, tokenPrefix, tokenScope)
	default:
		if !allowsReadOnlyMethod(w, r) {
			return
		}
		data, err := lookupUserCommunicationSurface(r.Context(), handler.communicationService, route, user.ID)
		if err != nil {
			writeUserCommunicationsError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "token_prefix": tokenPrefix, "token_scope": tokenScope, "user": user.Username, "surface": route.surface, "data": data})
	}
}

func (handler UserHandler) handleUserContactSurface(w http.ResponseWriter, r *http.Request, route userCommunicationRoute, user model.User, sessionID string) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		data, err := lookupUserCommunicationSurface(r.Context(), handler.communicationService, route, user.ID)
		if err != nil {
			writeUserCommunicationsError(w, err)
			return
		}
		if prefersJSON(r.Header.Get("Accept")) {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "session_id": sessionID, "user": user.Username, "surface": route.surface, "data": data})
			return
		}
		writeUserCommunicationText(w, user.Username, route.surface, data)
	case http.MethodPost:
		if route.hasID {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		contact, err := readUserContactRequest(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid contact request"})
			return
		}
		created, err := handler.communicationService.CreateContact(r.Context(), user.ID, contact)
		if err != nil {
			writeUserCommunicationsError(w, err)
			return
		}
		if prefersJSON(r.Header.Get("Accept")) {
			writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "session_id": sessionID, "user": user.Username, "surface": route.surface, "data": created})
			return
		}
		writeUserCommunicationText(w, user.Username, route.surface, created)
	case http.MethodDelete:
		if !route.hasID {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if err := handler.communicationService.DeleteContact(r.Context(), user.ID, route.id); err != nil {
			writeUserCommunicationsError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": "deleted", "id": route.id})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (handler APIUserHandler) handleUserContactSurface(w http.ResponseWriter, r *http.Request, route userCommunicationRoute, user model.User, tokenPrefix string, tokenScope model.TokenScope) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		data, err := lookupUserCommunicationSurface(r.Context(), handler.communicationService, route, user.ID)
		if err != nil {
			writeUserCommunicationsError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "token_prefix": tokenPrefix, "token_scope": tokenScope, "user": user.Username, "surface": route.surface, "data": data})
	case http.MethodPost:
		if route.hasID {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		contact, err := readUserContactRequest(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid contact request"})
			return
		}
		created, err := handler.communicationService.CreateContact(r.Context(), user.ID, contact)
		if err != nil {
			writeUserCommunicationsError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "token_prefix": tokenPrefix, "token_scope": tokenScope, "user": user.Username, "surface": route.surface, "data": created})
	case http.MethodDelete:
		if !route.hasID {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if err := handler.communicationService.DeleteContact(r.Context(), user.ID, route.id); err != nil {
			writeUserCommunicationsError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": "deleted", "id": route.id})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (handler UserHandler) handleUserSettingsSurface(w http.ResponseWriter, r *http.Request, user model.User, sessionID string) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		data, err := handler.communicationService.GetSettings(r.Context(), user.ID)
		if err != nil {
			writeUserCommunicationsError(w, err)
			return
		}
		if prefersJSON(r.Header.Get("Accept")) {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "session_id": sessionID, "user": user.Username, "surface": "communications/settings", "data": data})
			return
		}
		writeUserCommunicationText(w, user.Username, "communications/settings", data)
	case http.MethodPost:
		settings, err := readUserCommunicationSettingsRequest(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid communications settings request"})
			return
		}
		updated, err := handler.communicationService.UpdateSettings(r.Context(), user.ID, settings)
		if err != nil {
			writeUserCommunicationsError(w, err)
			return
		}
		if prefersJSON(r.Header.Get("Accept")) {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "session_id": sessionID, "user": user.Username, "surface": "communications/settings", "data": updated})
			return
		}
		writeUserCommunicationText(w, user.Username, "communications/settings", updated)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (handler APIUserHandler) handleUserSettingsSurface(w http.ResponseWriter, r *http.Request, user model.User, tokenPrefix string, tokenScope model.TokenScope) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		data, err := handler.communicationService.GetSettings(r.Context(), user.ID)
		if err != nil {
			writeUserCommunicationsError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "token_prefix": tokenPrefix, "token_scope": tokenScope, "user": user.Username, "surface": "communications/settings", "data": data})
	case http.MethodPost:
		settings, err := readUserCommunicationSettingsRequest(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid communications settings request"})
			return
		}
		updated, err := handler.communicationService.UpdateSettings(r.Context(), user.ID, settings)
		if err != nil {
			writeUserCommunicationsError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "token_prefix": tokenPrefix, "token_scope": tokenScope, "user": user.Username, "surface": "communications/settings", "data": updated})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func lookupUserCommunicationSurface(ctx context.Context, communicationService service.UserCommunicationsService, route userCommunicationRoute, userID int64) (any, error) {
	switch route.surface {
	case "dashboard":
		return communicationService.Dashboard(ctx, userID)
	case "contacts":
		if route.hasID {
			return communicationService.GetContact(ctx, userID, route.id)
		}
		return communicationService.ListContacts(ctx, userID)
	case "call-history":
		return communicationService.ListCallHistory(ctx, userID)
	case "voicemail":
		return communicationService.ListVoicemails(ctx, userID)
	case "messages":
		return communicationService.ListMessages(ctx, userID)
	case "presence":
		return communicationService.Presence(ctx, userID)
	case "webphone":
		return communicationService.Webphone(ctx, userID)
	case "communications/settings":
		return communicationService.GetSettings(ctx, userID)
	default:
		return nil, &service.UserCommunicationsError{Code: "COMMUNICATION_NOT_FOUND", Message: "surface not found"}
	}
}

func readUserContactRequest(r *http.Request) (model.UserContact, error) {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type"))), "application/json") {
		defer r.Body.Close()
		var requestBody userContactRequest
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			return model.UserContact{}, err
		}
		return model.UserContact{
			DisplayName:     requestBody.DisplayName,
			ExtensionNumber: requestBody.ExtensionNumber,
			PhoneNumber:     requestBody.PhoneNumber,
			Email:           requestBody.Email,
			Favorite:        requestBody.Favorite,
		}, nil
	}
	if err := r.ParseForm(); err != nil {
		return model.UserContact{}, err
	}
	return model.UserContact{
		DisplayName:     firstNonEmpty(r.FormValue("display_name"), r.FormValue("displayName")),
		ExtensionNumber: firstNonEmpty(r.FormValue("extension_number"), r.FormValue("extensionNumber")),
		PhoneNumber:     firstNonEmpty(r.FormValue("phone_number"), r.FormValue("phoneNumber")),
		Email:           r.FormValue("email"),
		Favorite:        formBool(r, "favorite"),
	}, nil
}

func readUserCommunicationSettingsRequest(r *http.Request) (model.UserCommunicationSettings, error) {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type"))), "application/json") {
		defer r.Body.Close()
		var requestBody userCommunicationSettingsRequest
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			return model.UserCommunicationSettings{}, err
		}
		return model.UserCommunicationSettings{
			DoNotDisturb:          requestBody.DoNotDisturb,
			CallForwardingTarget:  requestBody.CallForwardingTarget,
			VoicemailEnabled:      requestBody.VoicemailEnabled,
			WebphoneEnabled:       requestBody.WebphoneEnabled,
			PresenceEnabled:       requestBody.PresenceEnabled,
			MessagingEnabled:      requestBody.MessagingEnabled,
			PreferredEndpoint:     requestBody.PreferredEndpoint,
			PreferredContactEmail: requestBody.PreferredContactEmail,
		}, nil
	}
	if err := r.ParseForm(); err != nil {
		return model.UserCommunicationSettings{}, err
	}
	return model.UserCommunicationSettings{
		DoNotDisturb:          formBool(r, "do_not_disturb"),
		CallForwardingTarget:  firstNonEmpty(r.FormValue("call_forwarding_target"), r.FormValue("callForwardingTarget")),
		VoicemailEnabled:      formBool(r, "voicemail_enabled"),
		WebphoneEnabled:       formBool(r, "webphone_enabled"),
		PresenceEnabled:       formBool(r, "presence_enabled"),
		MessagingEnabled:      formBool(r, "messaging_enabled"),
		PreferredEndpoint:     firstNonEmpty(r.FormValue("preferred_endpoint"), r.FormValue("preferredEndpoint")),
		PreferredContactEmail: firstNonEmpty(r.FormValue("preferred_contact_email"), r.FormValue("preferredContactEmail")),
	}, nil
}

func writeUserCommunicationText(w http.ResponseWriter, username string, surface string, data any) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "User: %s\nSurface: %s\nData: %+v\n", username, surface, data)
}

func formBool(r *http.Request, key string) bool {
	value := strings.TrimSpace(strings.ToLower(firstNonEmpty(r.FormValue(key), r.FormValue(camelKey(key)))))
	return value == "1" || value == "true" || value == "on" || value == "yes"
}

func camelKey(value string) string {
	parts := strings.Split(value, "_")
	result := parts[0]
	for _, part := range parts[1:] {
		if part == "" {
			continue
		}
		result += strings.ToUpper(part[:1]) + part[1:]
	}
	return result
}
