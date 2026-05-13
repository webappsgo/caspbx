package store

import (
	"context"
	"errors"

	"github.com/casapps/caspbx/src/server/model"
)

var ErrNotFound = errors.New("record not found")

type NamespaceLookup interface {
	UserExistsByName(context.Context, string) (bool, error)
	OrgExistsBySlug(context.Context, string) (bool, error)
}

type TenantLookup interface {
	FindTenantByHost(context.Context, string) (int64, error)
}

type DomainLookup interface {
	FindDomainByHost(context.Context, string) (model.CustomDomain, error)
}

type DomainStore interface {
	DomainLookup
	SaveCustomDomain(context.Context, model.CustomDomain) (model.CustomDomain, error)
	FindCustomDomainByID(context.Context, int64) (model.CustomDomain, error)
	FindCustomDomainByDomain(context.Context, string) (model.CustomDomain, error)
	ListCustomDomains(context.Context) ([]model.CustomDomain, error)
	ListCustomDomainsByOwner(context.Context, model.DomainOwnerType, int64) ([]model.CustomDomain, error)
	DeleteCustomDomainByID(context.Context, int64) error
}

type AsteriskStore interface {
	SaveAsteriskState(context.Context, model.AsteriskState) (model.AsteriskState, error)
	GetAsteriskState(context.Context) (model.AsteriskState, error)
}

type PBXStore interface {
	SavePBXPlan(context.Context, model.PBXPlan) (model.PBXPlan, error)
	GetPBXPlan(context.Context) (model.PBXPlan, error)
}

type CommunicationStore interface {
	SaveUserCommunicationSettings(context.Context, model.UserCommunicationSettings) (model.UserCommunicationSettings, error)
	FindUserCommunicationSettings(context.Context, int64) (model.UserCommunicationSettings, error)
	SaveUserContact(context.Context, model.UserContact) (model.UserContact, error)
	FindUserContact(context.Context, int64, int64) (model.UserContact, error)
	ListUserContacts(context.Context, int64) ([]model.UserContact, error)
	DeleteUserContact(context.Context, int64, int64) error
	SaveUserVoicemail(context.Context, model.UserVoicemail) (model.UserVoicemail, error)
	ListUserVoicemails(context.Context, int64) ([]model.UserVoicemail, error)
	SaveUserCallRecord(context.Context, model.UserCallRecord) (model.UserCallRecord, error)
	ListUserCallRecords(context.Context, int64) ([]model.UserCallRecord, error)
	SaveUserMessage(context.Context, model.UserMessage) (model.UserMessage, error)
	ListUserMessages(context.Context, int64) ([]model.UserMessage, error)
}

type OperatorStore interface {
	SaveOperatorRuntimeState(context.Context, model.OperatorRuntimeState) (model.OperatorRuntimeState, error)
	GetOperatorRuntimeState(context.Context) (model.OperatorRuntimeState, error)
}

type AdminCredentialStore interface {
	SaveAdmin(context.Context, model.Admin) (model.Admin, error)
	FindAdminByUsername(context.Context, string) (model.Admin, error)
	FindAdminByID(context.Context, int64) (model.Admin, error)
}

type UserCredentialStore interface {
	SaveUser(context.Context, model.User) (model.User, error)
	FindUserByUsername(context.Context, string) (model.User, error)
	FindUserByEmail(context.Context, string) (model.User, error)
	FindUserByID(context.Context, int64) (model.User, error)
}

type SessionStore interface {
	SaveSession(context.Context, model.Session) (model.Session, error)
	FindSessionByTokenHash(context.Context, model.SessionKind, string) (model.Session, error)
	DeleteSessionByTokenHash(context.Context, model.SessionKind, string) error
}

type TokenStore interface {
	SaveToken(context.Context, model.Token) (model.Token, error)
	FindTokenByHash(context.Context, model.TokenOwnerType, string) (model.Token, error)
	DeleteTokenByHash(context.Context, model.TokenOwnerType, string) error
}

type OrganizationStore interface {
	SaveOrganization(context.Context, model.Organization) (model.Organization, error)
	FindOrganizationBySlug(context.Context, string) (model.Organization, error)
	FindOrganizationByID(context.Context, int64) (model.Organization, error)
	SaveOrganizationPreferences(context.Context, model.OrganizationPreferences) (model.OrganizationPreferences, error)
	FindOrganizationPreferencesByOrgID(context.Context, int64) (model.OrganizationPreferences, error)
	SaveOrganizationMember(context.Context, model.OrganizationMember) (model.OrganizationMember, error)
	FindOrganizationMember(context.Context, int64) (model.OrganizationMember, error)
	FindOrganizationMemberByUserID(context.Context, int64, int64) (model.OrganizationMember, error)
	ListOrganizationMembers(context.Context, int64) ([]model.OrganizationMember, error)
}

type AuthStore interface {
	AdminCredentialStore
	UserCredentialStore
	SessionStore
	TokenStore
}

type RuntimeStore interface {
	AuthStore
	OrganizationStore
	DomainStore
	AsteriskStore
	PBXStore
	CommunicationStore
	OperatorStore
}
