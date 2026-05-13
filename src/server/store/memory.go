package store

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/casapps/caspbx/src/server/model"
)

type MemoryStore struct {
	mu              sync.RWMutex
	nextAdminID     int64
	nextUserID      int64
	nextOrgID       int64
	nextOrgMemberID int64
	nextDomainID    int64
	nextTokenID     int64
	nextContactID   int64
	nextVoicemailID int64
	nextCallID      int64
	nextMessageID   int64
	adminsByID      map[int64]model.Admin
	adminIDsByName  map[string]int64
	usersByID       map[int64]model.User
	userIDsByName   map[string]int64
	userIDsByEmail  map[string]int64
	orgsByID        map[int64]model.Organization
	orgIDsBySlug    map[string]int64
	orgPrefsByOrgID map[int64]model.OrganizationPreferences
	orgMembersByID  map[int64]model.OrganizationMember
	domainsByID     map[int64]model.CustomDomain
	domainIDsByName map[string]int64
	asteriskState   model.AsteriskState
	pbxPlan         model.PBXPlan
	operatorState   model.OperatorRuntimeState
	userCommPrefs   map[int64]model.UserCommunicationSettings
	userContacts    map[int64][]model.UserContact
	userVoicemails  map[int64][]model.UserVoicemail
	userCallRecords map[int64][]model.UserCallRecord
	userMessages    map[int64][]model.UserMessage
	adminSessions   map[string]model.Session
	userSessions    map[string]model.Session
	adminTokens     map[string]model.Token
	userTokens      map[string]model.Token
	orgTokens       map[string]model.Token
	usernames       map[string]struct{}
	orgSlugs        map[string]struct{}
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		nextAdminID:     1,
		nextUserID:      1,
		nextOrgID:       1,
		nextOrgMemberID: 1,
		nextDomainID:    1,
		nextTokenID:     1,
		nextContactID:   1,
		nextVoicemailID: 1,
		nextCallID:      1,
		nextMessageID:   1,
		adminsByID:      map[int64]model.Admin{},
		adminIDsByName:  map[string]int64{},
		usersByID:       map[int64]model.User{},
		userIDsByName:   map[string]int64{},
		userIDsByEmail:  map[string]int64{},
		orgsByID:        map[int64]model.Organization{},
		orgIDsBySlug:    map[string]int64{},
		orgPrefsByOrgID: map[int64]model.OrganizationPreferences{},
		orgMembersByID:  map[int64]model.OrganizationMember{},
		domainsByID:     map[int64]model.CustomDomain{},
		domainIDsByName: map[string]int64{},
		asteriskState:   model.DefaultAsteriskState(),
		pbxPlan:         model.DefaultPBXPlan(),
		operatorState:   model.DefaultOperatorRuntimeState(),
		userCommPrefs:   map[int64]model.UserCommunicationSettings{},
		userContacts:    map[int64][]model.UserContact{},
		userVoicemails:  map[int64][]model.UserVoicemail{},
		userCallRecords: map[int64][]model.UserCallRecord{},
		userMessages:    map[int64][]model.UserMessage{},
		adminSessions:   map[string]model.Session{},
		userSessions:    map[string]model.Session{},
		adminTokens:     map[string]model.Token{},
		userTokens:      map[string]model.Token{},
		orgTokens:       map[string]model.Token{},
		usernames:       map[string]struct{}{},
		orgSlugs:        map[string]struct{}{},
	}
}

func (memoryStore *MemoryStore) UserExistsByName(_ context.Context, username string) (bool, error) {
	memoryStore.mu.RLock()
	defer memoryStore.mu.RUnlock()
	_, exists := memoryStore.usernames[strings.TrimSpace(username)]
	return exists, nil
}

func (memoryStore *MemoryStore) OrgExistsBySlug(_ context.Context, slug string) (bool, error) {
	memoryStore.mu.RLock()
	defer memoryStore.mu.RUnlock()
	_, exists := memoryStore.orgSlugs[strings.TrimSpace(slug)]
	return exists, nil
}

func (memoryStore *MemoryStore) SaveAdmin(_ context.Context, admin model.Admin) (model.Admin, error) {
	memoryStore.mu.Lock()
	defer memoryStore.mu.Unlock()

	admin.Username = strings.TrimSpace(admin.Username)
	if admin.ID == 0 {
		admin.ID = memoryStore.nextAdminID
		memoryStore.nextAdminID++
	}
	memoryStore.adminsByID[admin.ID] = admin
	memoryStore.adminIDsByName[admin.Username] = admin.ID
	return admin, nil
}

func (memoryStore *MemoryStore) SaveAsteriskState(_ context.Context, state model.AsteriskState) (model.AsteriskState, error) {
	memoryStore.mu.Lock()
	defer memoryStore.mu.Unlock()

	if state.MinimumSupportedVersion == "" {
		state.MinimumSupportedVersion = model.DefaultAsteriskState().MinimumSupportedVersion
	}
	if state.DetectionStatus == "" {
		state.DetectionStatus = model.DefaultAsteriskState().DetectionStatus
	}
	if state.HealthStatus == "" {
		state.HealthStatus = model.DefaultAsteriskState().HealthStatus
	}
	if state.ChannelDrivers == nil {
		state.ChannelDrivers = []string{}
	}
	if state.EndpointStacks == nil {
		state.EndpointStacks = []string{}
	}
	if state.Codecs == nil {
		state.Codecs = []string{}
	}
	if state.Capabilities == nil {
		state.Capabilities = []model.AsteriskCapability{}
	}
	if state.Subsystems == nil {
		state.Subsystems = []model.AsteriskManagedSubsystem{}
	}
	if state.UpdatedAt.IsZero() {
		state.UpdatedAt = time.Now().UTC()
	}
	memoryStore.asteriskState = state
	return state, nil
}

func (memoryStore *MemoryStore) GetAsteriskState(_ context.Context) (model.AsteriskState, error) {
	memoryStore.mu.RLock()
	defer memoryStore.mu.RUnlock()
	return memoryStore.asteriskState, nil
}

func (memoryStore *MemoryStore) SavePBXPlan(_ context.Context, plan model.PBXPlan) (model.PBXPlan, error) {
	memoryStore.mu.Lock()
	defer memoryStore.mu.Unlock()
	normalizedPlan := model.NormalizePBXPlan(plan)
	if normalizedPlan.UpdatedAt.IsZero() {
		normalizedPlan.UpdatedAt = time.Now().UTC()
	}
	model.SortPBXPlan(&normalizedPlan)
	memoryStore.pbxPlan = normalizedPlan
	return normalizedPlan, nil
}

func (memoryStore *MemoryStore) GetPBXPlan(_ context.Context) (model.PBXPlan, error) {
	memoryStore.mu.RLock()
	defer memoryStore.mu.RUnlock()
	return model.NormalizePBXPlan(memoryStore.pbxPlan), nil
}

func (memoryStore *MemoryStore) SaveOperatorRuntimeState(_ context.Context, state model.OperatorRuntimeState) (model.OperatorRuntimeState, error) {
	memoryStore.mu.Lock()
	defer memoryStore.mu.Unlock()

	normalized := model.NormalizeOperatorRuntimeState(state)
	if normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = time.Now().UTC()
	}
	model.SortOperatorRuntimeState(&normalized)
	memoryStore.operatorState = normalized
	return normalized, nil
}

func (memoryStore *MemoryStore) GetOperatorRuntimeState(_ context.Context) (model.OperatorRuntimeState, error) {
	memoryStore.mu.RLock()
	defer memoryStore.mu.RUnlock()
	return model.NormalizeOperatorRuntimeState(memoryStore.operatorState), nil
}

func (memoryStore *MemoryStore) SaveUserCommunicationSettings(_ context.Context, settings model.UserCommunicationSettings) (model.UserCommunicationSettings, error) {
	memoryStore.mu.Lock()
	defer memoryStore.mu.Unlock()

	normalized := model.NormalizeUserCommunicationSettings(settings)
	if normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = time.Now().UTC()
	}
	memoryStore.userCommPrefs[normalized.UserID] = normalized
	return normalized, nil
}

func (memoryStore *MemoryStore) FindUserCommunicationSettings(_ context.Context, userID int64) (model.UserCommunicationSettings, error) {
	memoryStore.mu.RLock()
	defer memoryStore.mu.RUnlock()

	settings, found := memoryStore.userCommPrefs[userID]
	if !found {
		return model.UserCommunicationSettings{}, ErrNotFound
	}
	return settings, nil
}

func (memoryStore *MemoryStore) SaveUserContact(_ context.Context, contact model.UserContact) (model.UserContact, error) {
	memoryStore.mu.Lock()
	defer memoryStore.mu.Unlock()

	normalized := model.NormalizeUserContact(contact)
	if normalized.ID == 0 {
		normalized.ID = memoryStore.nextContactID
		memoryStore.nextContactID++
		if normalized.CreatedAt.IsZero() {
			normalized.CreatedAt = time.Now().UTC()
		}
	}
	normalized.UpdatedAt = time.Now().UTC()
	contacts := append([]model.UserContact{}, memoryStore.userContacts[normalized.UserID]...)
	replaced := false
	for index := range contacts {
		if contacts[index].ID == normalized.ID {
			normalized.CreatedAt = contacts[index].CreatedAt
			contacts[index] = normalized
			replaced = true
			break
		}
	}
	if !replaced {
		contacts = append(contacts, normalized)
	}
	model.SortUserContacts(contacts)
	memoryStore.userContacts[normalized.UserID] = contacts
	return normalized, nil
}

func (memoryStore *MemoryStore) FindUserContact(_ context.Context, userID int64, contactID int64) (model.UserContact, error) {
	memoryStore.mu.RLock()
	defer memoryStore.mu.RUnlock()

	for _, contact := range memoryStore.userContacts[userID] {
		if contact.ID == contactID {
			return contact, nil
		}
	}
	return model.UserContact{}, ErrNotFound
}

func (memoryStore *MemoryStore) ListUserContacts(_ context.Context, userID int64) ([]model.UserContact, error) {
	memoryStore.mu.RLock()
	defer memoryStore.mu.RUnlock()

	contacts := append([]model.UserContact{}, memoryStore.userContacts[userID]...)
	model.SortUserContacts(contacts)
	return contacts, nil
}

func (memoryStore *MemoryStore) DeleteUserContact(_ context.Context, userID int64, contactID int64) error {
	memoryStore.mu.Lock()
	defer memoryStore.mu.Unlock()

	contacts := append([]model.UserContact{}, memoryStore.userContacts[userID]...)
	filtered := make([]model.UserContact, 0, len(contacts))
	removed := false
	for _, contact := range contacts {
		if contact.ID == contactID {
			removed = true
			continue
		}
		filtered = append(filtered, contact)
	}
	if !removed {
		return ErrNotFound
	}
	memoryStore.userContacts[userID] = filtered
	return nil
}

func (memoryStore *MemoryStore) SaveUserVoicemail(_ context.Context, voicemail model.UserVoicemail) (model.UserVoicemail, error) {
	memoryStore.mu.Lock()
	defer memoryStore.mu.Unlock()

	if voicemail.ID == 0 {
		voicemail.ID = memoryStore.nextVoicemailID
		memoryStore.nextVoicemailID++
	}
	if voicemail.ReceivedAt.IsZero() {
		voicemail.ReceivedAt = time.Now().UTC()
	}
	voicemail.ExtensionNumber = strings.TrimSpace(voicemail.ExtensionNumber)
	voicemail.From = strings.TrimSpace(voicemail.From)
	mailbox := append([]model.UserVoicemail{}, memoryStore.userVoicemails[voicemail.UserID]...)
	mailbox = upsertVoicemail(mailbox, voicemail)
	model.SortUserVoicemails(mailbox)
	memoryStore.userVoicemails[voicemail.UserID] = mailbox
	return voicemail, nil
}

func (memoryStore *MemoryStore) ListUserVoicemails(_ context.Context, userID int64) ([]model.UserVoicemail, error) {
	memoryStore.mu.RLock()
	defer memoryStore.mu.RUnlock()

	voicemails := append([]model.UserVoicemail{}, memoryStore.userVoicemails[userID]...)
	model.SortUserVoicemails(voicemails)
	return voicemails, nil
}

func (memoryStore *MemoryStore) SaveUserCallRecord(_ context.Context, record model.UserCallRecord) (model.UserCallRecord, error) {
	memoryStore.mu.Lock()
	defer memoryStore.mu.Unlock()

	if record.ID == 0 {
		record.ID = memoryStore.nextCallID
		memoryStore.nextCallID++
	}
	if record.StartedAt.IsZero() {
		record.StartedAt = time.Now().UTC()
	}
	record.ExtensionNumber = strings.TrimSpace(record.ExtensionNumber)
	record.Direction = strings.TrimSpace(strings.ToLower(record.Direction))
	record.Counterparty = strings.TrimSpace(record.Counterparty)
	record.Disposition = strings.TrimSpace(strings.ToLower(record.Disposition))
	records := append([]model.UserCallRecord{}, memoryStore.userCallRecords[record.UserID]...)
	records = upsertCallRecord(records, record)
	model.SortUserCallRecords(records)
	memoryStore.userCallRecords[record.UserID] = records
	return record, nil
}

func (memoryStore *MemoryStore) ListUserCallRecords(_ context.Context, userID int64) ([]model.UserCallRecord, error) {
	memoryStore.mu.RLock()
	defer memoryStore.mu.RUnlock()

	records := append([]model.UserCallRecord{}, memoryStore.userCallRecords[userID]...)
	model.SortUserCallRecords(records)
	return records, nil
}

func (memoryStore *MemoryStore) SaveUserMessage(_ context.Context, message model.UserMessage) (model.UserMessage, error) {
	memoryStore.mu.Lock()
	defer memoryStore.mu.Unlock()

	if message.ID == 0 {
		message.ID = memoryStore.nextMessageID
		memoryStore.nextMessageID++
	}
	if message.ReceivedAt.IsZero() {
		message.ReceivedAt = time.Now().UTC()
	}
	message.Direction = strings.TrimSpace(strings.ToLower(message.Direction))
	message.Counterparty = strings.TrimSpace(message.Counterparty)
	message.Transport = strings.TrimSpace(strings.ToLower(message.Transport))
	message.Body = strings.TrimSpace(message.Body)
	messages := append([]model.UserMessage{}, memoryStore.userMessages[message.UserID]...)
	messages = upsertMessage(messages, message)
	model.SortUserMessages(messages)
	memoryStore.userMessages[message.UserID] = messages
	return message, nil
}

func (memoryStore *MemoryStore) ListUserMessages(_ context.Context, userID int64) ([]model.UserMessage, error) {
	memoryStore.mu.RLock()
	defer memoryStore.mu.RUnlock()

	messages := append([]model.UserMessage{}, memoryStore.userMessages[userID]...)
	model.SortUserMessages(messages)
	return messages, nil
}

func (memoryStore *MemoryStore) FindAdminByUsername(_ context.Context, username string) (model.Admin, error) {
	memoryStore.mu.RLock()
	defer memoryStore.mu.RUnlock()

	adminID, found := memoryStore.adminIDsByName[strings.TrimSpace(username)]
	if !found {
		return model.Admin{}, ErrNotFound
	}
	return memoryStore.adminsByID[adminID], nil
}

func (memoryStore *MemoryStore) FindAdminByID(_ context.Context, id int64) (model.Admin, error) {
	memoryStore.mu.RLock()
	defer memoryStore.mu.RUnlock()

	admin, found := memoryStore.adminsByID[id]
	if !found {
		return model.Admin{}, ErrNotFound
	}
	return admin, nil
}

func (memoryStore *MemoryStore) SaveUser(_ context.Context, user model.User) (model.User, error) {
	memoryStore.mu.Lock()
	defer memoryStore.mu.Unlock()

	user.Username = strings.TrimSpace(user.Username)
	user.AccountEmail = model.NormalizeEmail(user.AccountEmail)
	if user.ID == 0 {
		user.ID = memoryStore.nextUserID
		memoryStore.nextUserID++
	}
	memoryStore.usersByID[user.ID] = user
	memoryStore.userIDsByName[user.Username] = user.ID
	memoryStore.usernames[user.Username] = struct{}{}
	if user.AccountEmail != "" {
		memoryStore.userIDsByEmail[user.AccountEmail] = user.ID
	}
	return user, nil
}

func (memoryStore *MemoryStore) FindUserByUsername(_ context.Context, username string) (model.User, error) {
	memoryStore.mu.RLock()
	defer memoryStore.mu.RUnlock()

	userID, found := memoryStore.userIDsByName[strings.TrimSpace(username)]
	if !found {
		return model.User{}, ErrNotFound
	}
	return memoryStore.usersByID[userID], nil
}

func (memoryStore *MemoryStore) FindUserByEmail(_ context.Context, email string) (model.User, error) {
	memoryStore.mu.RLock()
	defer memoryStore.mu.RUnlock()

	userID, found := memoryStore.userIDsByEmail[model.NormalizeEmail(email)]
	if !found {
		return model.User{}, ErrNotFound
	}
	return memoryStore.usersByID[userID], nil
}

func (memoryStore *MemoryStore) FindUserByID(_ context.Context, id int64) (model.User, error) {
	memoryStore.mu.RLock()
	defer memoryStore.mu.RUnlock()

	user, found := memoryStore.usersByID[id]
	if !found {
		return model.User{}, ErrNotFound
	}
	return user, nil
}

func (memoryStore *MemoryStore) SaveSession(_ context.Context, session model.Session) (model.Session, error) {
	memoryStore.mu.Lock()
	defer memoryStore.mu.Unlock()

	switch session.Kind {
	case model.SessionKindAdmin:
		memoryStore.adminSessions[session.TokenHash] = session
	case model.SessionKindUser:
		memoryStore.userSessions[session.TokenHash] = session
	default:
		return model.Session{}, ErrNotFound
	}
	return session, nil
}

func (memoryStore *MemoryStore) FindSessionByTokenHash(_ context.Context, kind model.SessionKind, tokenHash string) (model.Session, error) {
	memoryStore.mu.RLock()
	defer memoryStore.mu.RUnlock()

	switch kind {
	case model.SessionKindAdmin:
		session, found := memoryStore.adminSessions[tokenHash]
		if !found {
			return model.Session{}, ErrNotFound
		}
		return session, nil
	case model.SessionKindUser:
		session, found := memoryStore.userSessions[tokenHash]
		if !found {
			return model.Session{}, ErrNotFound
		}
		return session, nil
	default:
		return model.Session{}, ErrNotFound
	}
}

func (memoryStore *MemoryStore) DeleteSessionByTokenHash(_ context.Context, kind model.SessionKind, tokenHash string) error {
	memoryStore.mu.Lock()
	defer memoryStore.mu.Unlock()

	switch kind {
	case model.SessionKindAdmin:
		if _, found := memoryStore.adminSessions[tokenHash]; !found {
			return ErrNotFound
		}
		delete(memoryStore.adminSessions, tokenHash)
	case model.SessionKindUser:
		if _, found := memoryStore.userSessions[tokenHash]; !found {
			return ErrNotFound
		}
		delete(memoryStore.userSessions, tokenHash)
	default:
		return ErrNotFound
	}

	return nil
}

func (memoryStore *MemoryStore) SaveOrganization(_ context.Context, organization model.Organization) (model.Organization, error) {
	memoryStore.mu.Lock()
	defer memoryStore.mu.Unlock()

	organization.Slug = strings.TrimSpace(organization.Slug)
	now := organization.UpdatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if organization.ID == 0 {
		organization.ID = memoryStore.nextOrgID
		memoryStore.nextOrgID++
		if organization.CreatedAt.IsZero() {
			organization.CreatedAt = now
		}
	}
	organization.UpdatedAt = now
	memoryStore.orgsByID[organization.ID] = organization
	memoryStore.orgIDsBySlug[organization.Slug] = organization.ID
	memoryStore.orgSlugs[organization.Slug] = struct{}{}
	return organization, nil
}

func (memoryStore *MemoryStore) FindOrganizationBySlug(_ context.Context, slug string) (model.Organization, error) {
	memoryStore.mu.RLock()
	defer memoryStore.mu.RUnlock()

	orgID, found := memoryStore.orgIDsBySlug[strings.TrimSpace(slug)]
	if !found {
		return model.Organization{}, ErrNotFound
	}
	return memoryStore.orgsByID[orgID], nil
}

func (memoryStore *MemoryStore) FindOrganizationByID(_ context.Context, id int64) (model.Organization, error) {
	memoryStore.mu.RLock()
	defer memoryStore.mu.RUnlock()

	organization, found := memoryStore.orgsByID[id]
	if !found {
		return model.Organization{}, ErrNotFound
	}
	return organization, nil
}

func (memoryStore *MemoryStore) SaveOrganizationPreferences(_ context.Context, preferences model.OrganizationPreferences) (model.OrganizationPreferences, error) {
	memoryStore.mu.Lock()
	defer memoryStore.mu.Unlock()

	now := preferences.UpdatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if preferences.CreatedAt.IsZero() {
		preferences.CreatedAt = now
	}
	preferences.UpdatedAt = now
	memoryStore.orgPrefsByOrgID[preferences.OrgID] = preferences
	return preferences, nil
}

func (memoryStore *MemoryStore) FindOrganizationPreferencesByOrgID(_ context.Context, orgID int64) (model.OrganizationPreferences, error) {
	memoryStore.mu.RLock()
	defer memoryStore.mu.RUnlock()

	preferences, found := memoryStore.orgPrefsByOrgID[orgID]
	if !found {
		return model.OrganizationPreferences{}, ErrNotFound
	}
	return preferences, nil
}

func (memoryStore *MemoryStore) SaveOrganizationMember(_ context.Context, member model.OrganizationMember) (model.OrganizationMember, error) {
	memoryStore.mu.Lock()
	defer memoryStore.mu.Unlock()

	if member.ID == 0 {
		member.ID = memoryStore.nextOrgMemberID
		memoryStore.nextOrgMemberID++
	}
	if member.CreatedAt.IsZero() {
		member.CreatedAt = time.Now().UTC()
	}
	memoryStore.orgMembersByID[member.ID] = member
	return member, nil
}

func (memoryStore *MemoryStore) FindOrganizationMember(_ context.Context, id int64) (model.OrganizationMember, error) {
	memoryStore.mu.RLock()
	defer memoryStore.mu.RUnlock()

	member, found := memoryStore.orgMembersByID[id]
	if !found {
		return model.OrganizationMember{}, ErrNotFound
	}
	return member, nil
}

func (memoryStore *MemoryStore) FindOrganizationMemberByUserID(_ context.Context, orgID int64, userID int64) (model.OrganizationMember, error) {
	memoryStore.mu.RLock()
	defer memoryStore.mu.RUnlock()

	for _, member := range memoryStore.orgMembersByID {
		if member.OrgID == orgID && member.UserID == userID {
			return member, nil
		}
	}
	return model.OrganizationMember{}, ErrNotFound
}

func (memoryStore *MemoryStore) ListOrganizationMembers(_ context.Context, orgID int64) ([]model.OrganizationMember, error) {
	memoryStore.mu.RLock()
	defer memoryStore.mu.RUnlock()

	members := make([]model.OrganizationMember, 0)
	for _, member := range memoryStore.orgMembersByID {
		if member.OrgID == orgID {
			members = append(members, member)
		}
	}
	return members, nil
}

func (memoryStore *MemoryStore) SaveCustomDomain(_ context.Context, domain model.CustomDomain) (model.CustomDomain, error) {
	memoryStore.mu.Lock()
	defer memoryStore.mu.Unlock()

	domain.Domain = model.NormalizeDomainName(domain.Domain)
	now := domain.UpdatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if domain.ID == 0 {
		domain.ID = memoryStore.nextDomainID
		memoryStore.nextDomainID++
		if domain.CreatedAt.IsZero() {
			domain.CreatedAt = now
		}
	}
	domain.UpdatedAt = now
	memoryStore.domainsByID[domain.ID] = domain
	memoryStore.domainIDsByName[domain.Domain] = domain.ID
	return domain, nil
}

func (memoryStore *MemoryStore) FindCustomDomainByID(_ context.Context, id int64) (model.CustomDomain, error) {
	memoryStore.mu.RLock()
	defer memoryStore.mu.RUnlock()

	domain, found := memoryStore.domainsByID[id]
	if !found {
		return model.CustomDomain{}, ErrNotFound
	}
	return domain, nil
}

func (memoryStore *MemoryStore) FindCustomDomainByDomain(_ context.Context, domain string) (model.CustomDomain, error) {
	memoryStore.mu.RLock()
	defer memoryStore.mu.RUnlock()

	domainID, found := memoryStore.domainIDsByName[model.NormalizeDomainName(domain)]
	if !found {
		return model.CustomDomain{}, ErrNotFound
	}
	return memoryStore.domainsByID[domainID], nil
}

func (memoryStore *MemoryStore) FindDomainByHost(ctx context.Context, host string) (model.CustomDomain, error) {
	return memoryStore.FindCustomDomainByDomain(ctx, host)
}

func (memoryStore *MemoryStore) ListCustomDomains(_ context.Context) ([]model.CustomDomain, error) {
	memoryStore.mu.RLock()
	defer memoryStore.mu.RUnlock()

	domains := make([]model.CustomDomain, 0, len(memoryStore.domainsByID))
	for _, domain := range memoryStore.domainsByID {
		domains = append(domains, domain)
	}
	return domains, nil
}

func (memoryStore *MemoryStore) ListCustomDomainsByOwner(_ context.Context, ownerType model.DomainOwnerType, ownerID int64) ([]model.CustomDomain, error) {
	memoryStore.mu.RLock()
	defer memoryStore.mu.RUnlock()

	domains := make([]model.CustomDomain, 0)
	for _, domain := range memoryStore.domainsByID {
		if domain.OwnerType == ownerType && domain.OwnerID == ownerID {
			domains = append(domains, domain)
		}
	}
	return domains, nil
}

func (memoryStore *MemoryStore) DeleteCustomDomainByID(_ context.Context, id int64) error {
	memoryStore.mu.Lock()
	defer memoryStore.mu.Unlock()

	domain, found := memoryStore.domainsByID[id]
	if !found {
		return ErrNotFound
	}
	delete(memoryStore.domainsByID, id)
	delete(memoryStore.domainIDsByName, domain.Domain)
	return nil
}

func (memoryStore *MemoryStore) SaveToken(_ context.Context, token model.Token) (model.Token, error) {
	memoryStore.mu.Lock()
	defer memoryStore.mu.Unlock()

	if token.ID == 0 {
		token.ID = memoryStore.nextTokenID
		memoryStore.nextTokenID++
	}

	switch token.OwnerType {
	case model.TokenOwnerAdmin:
		memoryStore.adminTokens[token.TokenHash] = token
	case model.TokenOwnerUser:
		memoryStore.userTokens[token.TokenHash] = token
	case model.TokenOwnerOrg:
		memoryStore.orgTokens[token.TokenHash] = token
	default:
		return model.Token{}, ErrNotFound
	}

	return token, nil
}

func (memoryStore *MemoryStore) FindTokenByHash(_ context.Context, ownerType model.TokenOwnerType, tokenHash string) (model.Token, error) {
	memoryStore.mu.RLock()
	defer memoryStore.mu.RUnlock()

	switch ownerType {
	case model.TokenOwnerAdmin:
		token, found := memoryStore.adminTokens[tokenHash]
		if !found {
			return model.Token{}, ErrNotFound
		}
		return token, nil
	case model.TokenOwnerUser:
		token, found := memoryStore.userTokens[tokenHash]
		if !found {
			return model.Token{}, ErrNotFound
		}
		return token, nil
	case model.TokenOwnerOrg:
		token, found := memoryStore.orgTokens[tokenHash]
		if !found {
			return model.Token{}, ErrNotFound
		}
		return token, nil
	default:
		return model.Token{}, ErrNotFound
	}
}

func (memoryStore *MemoryStore) DeleteTokenByHash(_ context.Context, ownerType model.TokenOwnerType, tokenHash string) error {
	memoryStore.mu.Lock()
	defer memoryStore.mu.Unlock()

	switch ownerType {
	case model.TokenOwnerAdmin:
		if _, found := memoryStore.adminTokens[tokenHash]; !found {
			return ErrNotFound
		}
		delete(memoryStore.adminTokens, tokenHash)
	case model.TokenOwnerUser:
		if _, found := memoryStore.userTokens[tokenHash]; !found {
			return ErrNotFound
		}
		delete(memoryStore.userTokens, tokenHash)
	case model.TokenOwnerOrg:
		if _, found := memoryStore.orgTokens[tokenHash]; !found {
			return ErrNotFound
		}
		delete(memoryStore.orgTokens, tokenHash)
	default:
		return ErrNotFound
	}

	return nil
}

func upsertVoicemail(existing []model.UserVoicemail, voicemail model.UserVoicemail) []model.UserVoicemail {
	for index := range existing {
		if existing[index].ID == voicemail.ID {
			existing[index] = voicemail
			return existing
		}
	}
	return append(existing, voicemail)
}

func upsertCallRecord(existing []model.UserCallRecord, record model.UserCallRecord) []model.UserCallRecord {
	for index := range existing {
		if existing[index].ID == record.ID {
			existing[index] = record
			return existing
		}
	}
	return append(existing, record)
}

func upsertMessage(existing []model.UserMessage, message model.UserMessage) []model.UserMessage {
	for index := range existing {
		if existing[index].ID == message.ID {
			existing[index] = message
			return existing
		}
	}
	return append(existing, message)
}
