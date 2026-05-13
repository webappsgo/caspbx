package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/casapps/caspbx/src/server/model"
	"github.com/casapps/caspbx/src/server/service"
	"github.com/casapps/caspbx/src/server/store"
)

type OrgHandler struct {
	routePrefix   string
	authService   service.AuthService
	domainService service.DomainService
	orgStore      store.OrganizationStore
	userCookie    SessionCookieConfig
}

type APIOrgHandler struct {
	routePrefix   string
	authService   service.AuthService
	domainService service.DomainService
	orgStore      store.OrganizationStore
}

type orgAccessContext struct {
	organization model.Organization
	preferences  model.OrganizationPreferences
	member       model.OrganizationMember
	hasMember    bool
}

type orgAccessState int

const (
	orgAccessAllowed orgAccessState = iota
	orgAccessUnauthorized
	orgAccessForbidden
	orgAccessNotFound
)

func NewOrgHandler(routePrefix string, authService service.AuthService, domainService service.DomainService, orgStore store.OrganizationStore, userCookie SessionCookieConfig) http.Handler {
	return OrgHandler{
		routePrefix:   routePrefix,
		authService:   authService,
		domainService: domainService,
		orgStore:      orgStore,
		userCookie:    userCookie,
	}
}

func NewAPIOrgHandler(routePrefix string, authService service.AuthService, domainService service.DomainService, orgStore store.OrganizationStore) http.Handler {
	return APIOrgHandler{
		routePrefix:   routePrefix,
		authService:   authService,
		domainService: domainService,
		orgStore:      orgStore,
	}
}

func (handler OrgHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	slug, relativePath, found := orgRouteParts(handler.routePrefix, r.URL.Path)
	if !found {
		http.NotFound(w, r)
		return
	}
	if relativePath != "domains" && !strings.HasPrefix(relativePath, "domains/") && !allowsReadOnlyMethod(w, r) {
		return
	}

	access, accessState := handler.resolveWebOrgAccess(r, slug)
	if !writeOrgAccessError(w, r, accessState) {
		return
	}

	switch relativePath {
	case "":
		handler.writeOrgProfile(w, r, access)
	case "members":
		handler.writeOrgMembers(w, r, access)
	case "settings":
		if !access.hasMember {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			return
		}
		if !access.member.CanManageOrganization() {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		writeJSON(w, http.StatusOK, orgSettingsResponse(access))
	default:
		if relativePath == "domains" || strings.HasPrefix(relativePath, "domains/") {
			handler.handleDomainSurface(w, r, access, relativePath)
			return
		}
		http.NotFound(w, r)
	}
}

func (handler APIOrgHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	slug, relativePath, found := orgRouteParts(handler.routePrefix, r.URL.Path)
	if !found {
		http.NotFound(w, r)
		return
	}
	if relativePath != "domains" && !strings.HasPrefix(relativePath, "domains/") && !allowsReadOnlyMethod(w, r) {
		return
	}

	access, accessState := handler.resolveAPIOrgAccess(r, slug)
	if !writeOrgAccessError(w, r, accessState) {
		return
	}

	switch {
	case relativePath == "":
		writeJSON(w, http.StatusOK, orgProfileResponse(access))
	case relativePath == "members":
		handler.writeAPIMembers(w, r, access)
	case relativePath == "settings":
		if !access.hasMember {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			return
		}
		if !access.member.CanManageOrganization() {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		writeJSON(w, http.StatusOK, orgSettingsResponse(access))
	case strings.HasPrefix(relativePath, "members/"):
		handler.writeAPIMember(w, r, access, strings.TrimPrefix(relativePath, "members/"))
	default:
		if relativePath == "domains" || strings.HasPrefix(relativePath, "domains/") {
			handler.handleDomainSurface(w, r, access, relativePath)
			return
		}
		http.NotFound(w, r)
	}
}

func writeOrgAccessError(w http.ResponseWriter, r *http.Request, accessState orgAccessState) bool {
	switch accessState {
	case orgAccessAllowed:
		return true
	case orgAccessUnauthorized:
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	case orgAccessForbidden:
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
	default:
		http.NotFound(w, r)
	}
	return false
}

func (handler OrgHandler) resolveWebOrgAccess(r *http.Request, slug string) (orgAccessContext, orgAccessState) {
	organization, preferences, loadError := loadOrganizationContext(r.Context(), handler.orgStore, slug)
	if loadError != nil {
		return orgAccessContext{}, orgAccessNotFound
	}

	access := orgAccessContext{
		organization: organization,
		preferences:  preferences,
	}

	member, memberFound := handler.resolveWebMember(r, organization.ID)
	if memberFound {
		access.member = member
		access.hasMember = true
	}

	if organization.Visibility != model.OrganizationVisibilityPublic && !access.hasMember {
		return orgAccessContext{}, orgAccessNotFound
	}

	return access, orgAccessAllowed
}

func (handler APIOrgHandler) resolveAPIOrgAccess(r *http.Request, slug string) (orgAccessContext, orgAccessState) {
	organization, preferences, loadError := loadOrganizationContext(r.Context(), handler.orgStore, slug)
	if loadError != nil {
		return orgAccessContext{}, orgAccessNotFound
	}

	access := orgAccessContext{
		organization: organization,
		preferences:  preferences,
	}

	member, hasMember, _ := handler.resolveAPIMember(r, organization.ID)
	if hasMember {
		access.member = member
		access.hasMember = true
	}

	if organization.Visibility == model.OrganizationVisibilityPublic {
		return access, orgAccessAllowed
	}
	if !access.hasMember {
		return orgAccessContext{}, orgAccessNotFound
	}

	return access, orgAccessAllowed
}

func (handler OrgHandler) resolveWebMember(r *http.Request, orgID int64) (model.OrganizationMember, bool) {
	sessionToken, tokenError := readSessionCookie(r, handler.userCookie.Name)
	if tokenError != nil {
		return model.OrganizationMember{}, false
	}

	session, resolveError := handler.authService.ResolveUserSession(r.Context(), sessionToken)
	if resolveError != nil {
		return model.OrganizationMember{}, false
	}

	member, memberError := handler.orgStore.FindOrganizationMemberByUserID(r.Context(), orgID, session.SubjectID)
	if memberError != nil {
		return model.OrganizationMember{}, false
	}

	return member, true
}

func (handler APIOrgHandler) resolveAPIMember(r *http.Request, orgID int64) (model.OrganizationMember, bool, orgAccessState) {
	token, tokenError := readBearerToken(r)
	if tokenError != nil {
		return model.OrganizationMember{}, false, orgAccessUnauthorized
	}

	if strings.HasPrefix(token, "org_") {
		orgToken, resolveError := handler.authService.ResolveOrgToken(r.Context(), token)
		if resolveError != nil {
			return model.OrganizationMember{}, false, orgAccessUnauthorized
		}
		if orgToken.OwnerID != orgID {
			return model.OrganizationMember{}, false, orgAccessForbidden
		}
		return model.OrganizationMember{
			OrgID:     orgID,
			Role:      model.OrganizationRoleOwner,
			CreatedAt: time.Now().UTC(),
		}, true, orgAccessAllowed
	}

	userToken, resolveError := handler.authService.ResolveUserToken(r.Context(), token)
	if resolveError != nil {
		return model.OrganizationMember{}, false, orgAccessUnauthorized
	}

	member, memberError := handler.orgStore.FindOrganizationMemberByUserID(r.Context(), orgID, userToken.OwnerID)
	if memberError != nil {
		return model.OrganizationMember{}, false, orgAccessForbidden
	}

	return member, true, orgAccessAllowed
}

func loadOrganizationContext(ctx context.Context, orgStore store.OrganizationStore, slug string) (model.Organization, model.OrganizationPreferences, error) {
	organization, orgError := orgStore.FindOrganizationBySlug(ctx, slug)
	if orgError != nil {
		return model.Organization{}, model.OrganizationPreferences{}, orgError
	}

	preferences, prefError := orgStore.FindOrganizationPreferencesByOrgID(ctx, organization.ID)
	if prefError == nil {
		return organization, preferences, nil
	}
	if prefError != store.ErrNotFound {
		return model.Organization{}, model.OrganizationPreferences{}, prefError
	}

	preferences = model.DefaultOrganizationPreferences()
	preferences.OrgID = organization.ID
	savedPreferences, saveError := orgStore.SaveOrganizationPreferences(ctx, preferences)
	if saveError != nil {
		return model.Organization{}, model.OrganizationPreferences{}, saveError
	}

	return organization, savedPreferences, nil
}

func (handler OrgHandler) writeOrgProfile(w http.ResponseWriter, r *http.Request, access orgAccessContext) {
	if prefersJSON(r.Header.Get("Accept")) {
		writeJSON(w, http.StatusOK, orgProfileResponse(access))
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(
		w,
		"Organization: %s\nSlug: %s\nVisibility: %s\nMembers visible: %t\n",
		access.organization.Name,
		access.organization.Slug,
		access.organization.Visibility,
		access.preferences.ShowMembers || access.hasMember,
	)
}

func (handler OrgHandler) writeOrgMembers(w http.ResponseWriter, r *http.Request, access orgAccessContext) {
	memberProfiles, membersError := buildOrganizationMemberResponses(r.Context(), handler.orgStore, handler.authService, access, access.hasMember)
	if membersError != nil {
		http.NotFound(w, r)
		return
	}

	if prefersJSON(r.Header.Get("Accept")) {
		writeJSON(w, http.StatusOK, organizationMembersResponse(access.organization.Slug, memberProfiles))
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "Organization Members: %s (%d)\n", access.organization.Slug, len(memberProfiles))
}

func (handler APIOrgHandler) writeAPIMembers(w http.ResponseWriter, r *http.Request, access orgAccessContext) {
	memberProfiles, membersError := buildOrganizationMemberResponses(r.Context(), handler.orgStore, handler.authService, access, access.hasMember)
	if membersError != nil {
		http.NotFound(w, r)
		return
	}

	writeJSON(w, http.StatusOK, organizationMembersResponse(access.organization.Slug, memberProfiles))
}

func (handler APIOrgHandler) writeAPIMember(w http.ResponseWriter, r *http.Request, access orgAccessContext, memberIDValue string) {
	memberID, parseError := strconv.ParseInt(memberIDValue, 10, 64)
	if parseError != nil {
		http.NotFound(w, r)
		return
	}

	member, memberError := handler.orgStore.FindOrganizationMember(r.Context(), memberID)
	if memberError != nil || member.OrgID != access.organization.ID {
		http.NotFound(w, r)
		return
	}

	user, userError := handler.authService.FindUserByID(r.Context(), member.UserID)
	if userError != nil {
		http.NotFound(w, r)
		return
	}
	if !access.hasMember && user.Visibility != model.UserVisibilityPublic {
		http.NotFound(w, r)
		return
	}

	writeJSON(w, http.StatusOK, memberProfileResponse(model.BuildOrganizationMemberProfile(user, member.Role, member.CreatedAt), member.ID))
}

func buildOrganizationMemberResponses(ctx context.Context, orgStore store.OrganizationStore, authService service.AuthService, access orgAccessContext, includePrivate bool) ([]map[string]any, error) {
	members, memberError := orgStore.ListOrganizationMembers(ctx, access.organization.ID)
	if memberError != nil {
		return nil, memberError
	}

	memberProfiles := make([]map[string]any, 0, len(members))
	for _, member := range members {
		user, userError := authService.FindUserByID(ctx, member.UserID)
		if userError != nil {
			continue
		}
		if !includePrivate && user.Visibility != model.UserVisibilityPublic {
			continue
		}
		memberProfiles = append(memberProfiles, memberProfileResponse(model.BuildOrganizationMemberProfile(user, member.Role, member.CreatedAt), member.ID))
	}

	return memberProfiles, nil
}

func organizationMembersResponse(slug string, members []map[string]any) map[string]any {
	return map[string]any{
		"organization": slug,
		"members":      members,
		"member_count": len(members),
	}
}

func orgProfileResponse(access orgAccessContext) map[string]any {
	return map[string]any{
		"slug":           access.organization.Slug,
		"name":           access.organization.Name,
		"description":    access.organization.Description,
		"website":        access.organization.Website,
		"location":       access.organization.Location,
		"visibility":     access.organization.Visibility,
		"show_members":   access.preferences.ShowMembers,
		"show_activity":  access.preferences.ShowActivity,
		"allow_invites":  access.preferences.AllowInvites,
		"member_context": access.hasMember,
		"created_at":     access.organization.CreatedAt.UTC().Format(time.RFC3339),
		"updated_at":     access.organization.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func orgSettingsResponse(access orgAccessContext) map[string]any {
	return map[string]any{
		"general": map[string]any{
			"name":        access.organization.Name,
			"slug":        access.organization.Slug,
			"description": access.organization.Description,
			"website":     access.organization.Website,
			"location":    access.organization.Location,
			"visibility":  access.organization.Visibility,
		},
		"members": map[string]any{
			"default_role":  access.preferences.DefaultMemberRole,
			"require_2fa":   access.preferences.RequireTwoFactor,
			"allow_invites": access.preferences.AllowInvites,
		},
		"visibility": map[string]any{
			"show_members":  access.preferences.ShowMembers,
			"show_activity": access.preferences.ShowActivity,
		},
		"notifications": map[string]any{
			"notify_new_member":   access.preferences.NotifyNewMember,
			"notify_member_leave": access.preferences.NotifyMemberLeave,
		},
		"your_permissions": map[string]any{
			"can_edit_settings":  access.member.CanManageOrganization(),
			"can_manage_members": access.member.CanManageOrganization(),
			"can_delete_org":     access.member.Role == model.OrganizationRoleOwner,
		},
	}
}

func memberProfileResponse(profile model.OrganizationMemberProfile, memberID int64) map[string]any {
	return map[string]any{
		"id":                 memberID,
		"username":           profile.Username,
		"display_name":       profile.DisplayName,
		"avatar":             profile.Avatar,
		"role":               profile.Role,
		"joined_at":          profile.JoinedAt.UTC().Format(time.RFC3339),
		"profile_visibility": profile.ProfileVisibility,
	}
}

func orgRouteParts(routePrefix string, requestPath string) (string, string, bool) {
	relativePath := routeTail(routePrefix, requestPath)
	if relativePath == "" {
		return "", "", false
	}

	parts := strings.Split(relativePath, "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		return "", "", false
	}
	if len(parts) == 1 {
		return parts[0], "", true
	}

	return parts[0], strings.Join(parts[1:], "/"), true
}
