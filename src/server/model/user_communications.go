package model

import (
	"slices"
	"strings"
	"time"
)

type UserContact struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	DisplayName     string    `json:"display_name"`
	ExtensionNumber string    `json:"extension_number,omitempty"`
	PhoneNumber     string    `json:"phone_number,omitempty"`
	Email           string    `json:"email,omitempty"`
	Favorite        bool      `json:"favorite"`
	CreatedAt       time.Time `json:"created_at,omitempty"`
	UpdatedAt       time.Time `json:"updated_at,omitempty"`
}

type UserVoicemail struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	ExtensionNumber string    `json:"extension_number,omitempty"`
	From            string    `json:"from"`
	DurationSeconds int       `json:"duration_seconds"`
	ReceivedAt      time.Time `json:"received_at,omitempty"`
	Read            bool      `json:"read"`
	Recording       bool      `json:"recording"`
}

type UserCallRecord struct {
	ID                int64     `json:"id"`
	UserID            int64     `json:"user_id"`
	ExtensionNumber   string    `json:"extension_number,omitempty"`
	Direction         string    `json:"direction"`
	Counterparty      string    `json:"counterparty"`
	StartedAt         time.Time `json:"started_at,omitempty"`
	DurationSeconds   int       `json:"duration_seconds"`
	Disposition       string    `json:"disposition,omitempty"`
	RecordingAvailable bool     `json:"recording_available"`
}

type UserMessage struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	Direction    string    `json:"direction"`
	Counterparty string    `json:"counterparty"`
	Transport    string    `json:"transport,omitempty"`
	Body         string    `json:"body"`
	ReceivedAt   time.Time `json:"received_at,omitempty"`
	Read         bool      `json:"read"`
}

type UserCommunicationSettings struct {
	UserID                int64     `json:"user_id"`
	ExtensionID           int64     `json:"extension_id,omitempty"`
	DoNotDisturb          bool      `json:"do_not_disturb"`
	CallForwardingTarget  string    `json:"call_forwarding_target,omitempty"`
	VoicemailEnabled      bool      `json:"voicemail_enabled"`
	WebphoneEnabled       bool      `json:"webphone_enabled"`
	PresenceEnabled       bool      `json:"presence_enabled"`
	MessagingEnabled      bool      `json:"messaging_enabled"`
	PreferredEndpoint     string    `json:"preferred_endpoint,omitempty"`
	PreferredContactEmail string    `json:"preferred_contact_email,omitempty"`
	UpdatedAt             time.Time `json:"updated_at,omitempty"`
}

func DefaultUserCommunicationSettings(userID int64) UserCommunicationSettings {
	return UserCommunicationSettings{
		UserID:           userID,
		VoicemailEnabled: true,
		WebphoneEnabled:  true,
		PresenceEnabled:  true,
		MessagingEnabled: true,
	}
}

func NormalizeUserContact(contact UserContact) UserContact {
	contact.DisplayName = strings.TrimSpace(contact.DisplayName)
	contact.ExtensionNumber = strings.TrimSpace(contact.ExtensionNumber)
	contact.PhoneNumber = strings.TrimSpace(contact.PhoneNumber)
	contact.Email = NormalizeEmail(contact.Email)
	return contact
}

func NormalizeUserCommunicationSettings(settings UserCommunicationSettings) UserCommunicationSettings {
	settings.CallForwardingTarget = strings.TrimSpace(settings.CallForwardingTarget)
	settings.PreferredEndpoint = strings.TrimSpace(settings.PreferredEndpoint)
	settings.PreferredContactEmail = NormalizeEmail(settings.PreferredContactEmail)
	return settings
}

func SortUserContacts(contacts []UserContact) {
	slices.SortFunc(contacts, func(left UserContact, right UserContact) int {
		if compare := strings.Compare(strings.ToLower(left.DisplayName), strings.ToLower(right.DisplayName)); compare != 0 {
			return compare
		}
		return int(left.ID - right.ID)
	})
}

func SortUserVoicemails(voicemails []UserVoicemail) {
	slices.SortFunc(voicemails, func(left UserVoicemail, right UserVoicemail) int {
		switch {
		case left.ReceivedAt.After(right.ReceivedAt):
			return -1
		case left.ReceivedAt.Before(right.ReceivedAt):
			return 1
		default:
			return int(left.ID - right.ID)
		}
	})
}

func SortUserCallRecords(records []UserCallRecord) {
	slices.SortFunc(records, func(left UserCallRecord, right UserCallRecord) int {
		switch {
		case left.StartedAt.After(right.StartedAt):
			return -1
		case left.StartedAt.Before(right.StartedAt):
			return 1
		default:
			return int(left.ID - right.ID)
		}
	})
}

func SortUserMessages(messages []UserMessage) {
	slices.SortFunc(messages, func(left UserMessage, right UserMessage) int {
		switch {
		case left.ReceivedAt.After(right.ReceivedAt):
			return -1
		case left.ReceivedAt.Before(right.ReceivedAt):
			return 1
		default:
			return int(left.ID - right.ID)
		}
	})
}
