package model

import (
	"testing"
	"time"
)

func TestOperatorRuntimeHelpers(t *testing.T) {
	state := DefaultOperatorRuntimeState()
	if len(state.Queues) != 0 || len(state.ActiveCalls) != 0 {
		t.Fatalf("expected default operator runtime state, got %+v", state)
	}

	normalized := NormalizeOperatorRuntimeState(OperatorRuntimeState{})
	if normalized.Queues == nil || normalized.Agents == nil || normalized.Trunks == nil || normalized.Conferences == nil || normalized.ParkedCalls == nil || normalized.ActiveCalls == nil {
		t.Fatalf("expected normalized slices, got %+v", normalized)
	}

	state = OperatorRuntimeState{
		Queues:      []OperatorQueueState{{Name: "Support"}, {Name: "Billing"}},
		Agents:      []OperatorAgentState{{ID: 2, QueueName: "Support", DisplayName: "Zulu"}, {ID: 1, QueueName: "Support", DisplayName: "Alpha"}, {ID: 3, QueueName: "Billing", DisplayName: "Beta"}},
		Trunks:      []OperatorTrunkState{{Name: "sip-b"}, {Name: "sip-a"}},
		Conferences: []OperatorConferenceState{{Name: "Zed"}, {Name: "Alpha"}},
		ParkedCalls: []ParkedCallState{{Slot: "702"}, {Slot: "701"}},
		ActiveCalls: []OperatorActiveCall{{ID: 2}, {ID: 1}},
	}
	SortOperatorRuntimeState(&state)
	if state.Queues[0].Name != "Billing" || state.Agents[0].ID != 3 || state.Trunks[0].Name != "sip-a" || state.Conferences[0].Name != "Alpha" || state.ParkedCalls[0].Slot != "701" || state.ActiveCalls[0].ID != 1 {
		t.Fatalf("expected sorted operator runtime state, got %+v", state)
	}

	state = OperatorRuntimeState{
		Agents: []OperatorAgentState{{ID: 2, QueueName: "Support", DisplayName: "Alpha"}, {ID: 1, QueueName: "Support", DisplayName: "Alpha"}},
	}
	SortOperatorRuntimeState(&state)
	if state.Agents[0].ID != 1 {
		t.Fatalf("expected agent tie-break by id, got %+v", state.Agents)
	}

	state = OperatorRuntimeState{
		Queues: []OperatorQueueState{{Name: "Zed", UpdatedAt: time.Unix(1, 0)}},
	}
	SortOperatorRuntimeState(&state)
	if state.Queues[0].Name != "Zed" {
		t.Fatalf("expected stable queue state, got %+v", state.Queues)
	}
}
