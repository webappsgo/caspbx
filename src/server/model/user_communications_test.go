package model

import (
	"testing"
	"time"
)

func TestUserCommunicationHelpers(t *testing.T) {
	settings := DefaultUserCommunicationSettings(7)
	if settings.UserID != 7 || !settings.WebphoneEnabled || !settings.MessagingEnabled {
		t.Fatalf("expected default communication settings, got %+v", settings)
	}

	normalizedContact := NormalizeUserContact(UserContact{
		DisplayName:     " Alice Example ",
		ExtensionNumber: " 1000 ",
		PhoneNumber:     " 18005551212 ",
		Email:           " Alice@Example.com ",
	})
	if normalizedContact.DisplayName != "Alice Example" || normalizedContact.ExtensionNumber != "1000" || normalizedContact.Email != "alice@example.com" {
		t.Fatalf("expected normalized contact, got %+v", normalizedContact)
	}

	normalizedSettings := NormalizeUserCommunicationSettings(UserCommunicationSettings{
		CallForwardingTarget:  " 1001 ",
		PreferredEndpoint:     " alice-softphone ",
		PreferredContactEmail: " Notify@Example.com ",
	})
	if normalizedSettings.CallForwardingTarget != "1001" || normalizedSettings.PreferredEndpoint != "alice-softphone" || normalizedSettings.PreferredContactEmail != "notify@example.com" {
		t.Fatalf("expected normalized settings, got %+v", normalizedSettings)
	}

	contacts := []UserContact{{ID: 2, DisplayName: "Zulu"}, {ID: 1, DisplayName: "alpha"}}
	SortUserContacts(contacts)
	if contacts[0].ID != 1 {
		t.Fatalf("expected contacts sorted by display name, got %+v", contacts)
	}
	contacts = []UserContact{{ID: 2, DisplayName: "same"}, {ID: 1, DisplayName: "same"}}
	SortUserContacts(contacts)
	if contacts[0].ID != 1 {
		t.Fatalf("expected contacts tie-break by id, got %+v", contacts)
	}

	voicemails := []UserVoicemail{{ID: 1, ReceivedAt: time.Unix(1, 0)}, {ID: 2, ReceivedAt: time.Unix(2, 0)}}
	SortUserVoicemails(voicemails)
	if voicemails[0].ID != 2 {
		t.Fatalf("expected voicemails sorted descending by received time, got %+v", voicemails)
	}
	voicemails = []UserVoicemail{{ID: 2, ReceivedAt: time.Unix(2, 0)}, {ID: 1, ReceivedAt: time.Unix(2, 0)}}
	SortUserVoicemails(voicemails)
	if voicemails[0].ID != 1 {
		t.Fatalf("expected voicemail tie-break by id, got %+v", voicemails)
	}
	voicemails = []UserVoicemail{{ID: 1, ReceivedAt: time.Unix(1, 0)}, {ID: 2, ReceivedAt: time.Unix(0, 0)}}
	SortUserVoicemails(voicemails)
	if voicemails[1].ID != 2 {
		t.Fatalf("expected voicemail before-branch coverage, got %+v", voicemails)
	}

	records := []UserCallRecord{{ID: 1, StartedAt: time.Unix(1, 0)}, {ID: 2, StartedAt: time.Unix(2, 0)}}
	SortUserCallRecords(records)
	if records[0].ID != 2 {
		t.Fatalf("expected call records sorted descending by start time, got %+v", records)
	}
	records = []UserCallRecord{{ID: 2, StartedAt: time.Unix(2, 0)}, {ID: 1, StartedAt: time.Unix(2, 0)}}
	SortUserCallRecords(records)
	if records[0].ID != 1 {
		t.Fatalf("expected call record tie-break by id, got %+v", records)
	}
	records = []UserCallRecord{{ID: 1, StartedAt: time.Unix(1, 0)}, {ID: 2, StartedAt: time.Unix(0, 0)}}
	SortUserCallRecords(records)
	if records[1].ID != 2 {
		t.Fatalf("expected call record before-branch coverage, got %+v", records)
	}

	messages := []UserMessage{{ID: 1, ReceivedAt: time.Unix(1, 0)}, {ID: 2, ReceivedAt: time.Unix(2, 0)}}
	SortUserMessages(messages)
	if messages[0].ID != 2 {
		t.Fatalf("expected messages sorted descending by received time, got %+v", messages)
	}
	messages = []UserMessage{{ID: 2, ReceivedAt: time.Unix(2, 0)}, {ID: 1, ReceivedAt: time.Unix(2, 0)}}
	SortUserMessages(messages)
	if messages[0].ID != 1 {
		t.Fatalf("expected message tie-break by id, got %+v", messages)
	}
	messages = []UserMessage{{ID: 1, ReceivedAt: time.Unix(1, 0)}, {ID: 2, ReceivedAt: time.Unix(0, 0)}}
	SortUserMessages(messages)
	if messages[1].ID != 2 {
		t.Fatalf("expected message before-branch coverage, got %+v", messages)
	}
}
