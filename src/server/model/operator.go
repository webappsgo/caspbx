package model

import (
	"slices"
	"strings"
	"time"
)

type OperatorQueueState struct {
	Name               string    `json:"name"`
	WaitingCalls       int       `json:"waiting_calls"`
	ActiveCalls        int       `json:"active_calls"`
	AvailableAgents    int       `json:"available_agents"`
	PausedAgents       int       `json:"paused_agents"`
	LongestWaitSeconds int       `json:"longest_wait_seconds"`
	ServiceLevelPct    int       `json:"service_level_pct"`
	UpdatedAt          time.Time `json:"updated_at,omitempty"`
}

type OperatorAgentState struct {
	ID              int64     `json:"id"`
	QueueName       string    `json:"queue_name,omitempty"`
	UserID          int64     `json:"user_id,omitempty"`
	ExtensionNumber string    `json:"extension_number,omitempty"`
	DisplayName     string    `json:"display_name"`
	Status          string    `json:"status"`
	Paused          bool      `json:"paused"`
	LastChangeAt    time.Time `json:"last_change_at,omitempty"`
}

type OperatorTrunkState struct {
	Name        string    `json:"name"`
	Technology  string    `json:"technology"`
	Registered  bool      `json:"registered"`
	ActiveCalls int       `json:"active_calls"`
	Healthy     bool      `json:"healthy"`
	Reason      string    `json:"reason,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

type OperatorConferenceState struct {
	Name             string    `json:"name"`
	ParticipantCount int       `json:"participant_count"`
	Recording        bool      `json:"recording"`
	Locked           bool      `json:"locked"`
	UpdatedAt        time.Time `json:"updated_at,omitempty"`
}

type ParkedCallState struct {
	Slot            string    `json:"slot"`
	Caller          string    `json:"caller"`
	ReturnTarget    string    `json:"return_target,omitempty"`
	DurationSeconds int       `json:"duration_seconds"`
	UpdatedAt       time.Time `json:"updated_at,omitempty"`
}

type OperatorActiveCall struct {
	ID              int64     `json:"id"`
	Direction       string    `json:"direction"`
	Source          string    `json:"source"`
	Destination     string    `json:"destination"`
	QueueName       string    `json:"queue_name,omitempty"`
	AgentExtension  string    `json:"agent_extension,omitempty"`
	DurationSeconds int       `json:"duration_seconds"`
	Parked          bool      `json:"parked"`
	UpdatedAt       time.Time `json:"updated_at,omitempty"`
}

type OperatorRuntimeState struct {
	Queues      []OperatorQueueState      `json:"queues,omitempty"`
	Agents      []OperatorAgentState      `json:"agents,omitempty"`
	Trunks      []OperatorTrunkState      `json:"trunks,omitempty"`
	Conferences []OperatorConferenceState `json:"conferences,omitempty"`
	ParkedCalls []ParkedCallState         `json:"parked_calls,omitempty"`
	ActiveCalls []OperatorActiveCall      `json:"active_calls,omitempty"`
	UpdatedAt   time.Time                 `json:"updated_at,omitempty"`
}

func DefaultOperatorRuntimeState() OperatorRuntimeState {
	return OperatorRuntimeState{
		Queues:      []OperatorQueueState{},
		Agents:      []OperatorAgentState{},
		Trunks:      []OperatorTrunkState{},
		Conferences: []OperatorConferenceState{},
		ParkedCalls: []ParkedCallState{},
		ActiveCalls: []OperatorActiveCall{},
	}
}

func NormalizeOperatorRuntimeState(state OperatorRuntimeState) OperatorRuntimeState {
	if state.Queues == nil {
		state.Queues = []OperatorQueueState{}
	}
	if state.Agents == nil {
		state.Agents = []OperatorAgentState{}
	}
	if state.Trunks == nil {
		state.Trunks = []OperatorTrunkState{}
	}
	if state.Conferences == nil {
		state.Conferences = []OperatorConferenceState{}
	}
	if state.ParkedCalls == nil {
		state.ParkedCalls = []ParkedCallState{}
	}
	if state.ActiveCalls == nil {
		state.ActiveCalls = []OperatorActiveCall{}
	}
	return state
}

func SortOperatorRuntimeState(state *OperatorRuntimeState) {
	slices.SortFunc(state.Queues, func(left OperatorQueueState, right OperatorQueueState) int {
		return strings.Compare(left.Name, right.Name)
	})
	slices.SortFunc(state.Agents, func(left OperatorAgentState, right OperatorAgentState) int {
		if compare := strings.Compare(left.QueueName, right.QueueName); compare != 0 {
			return compare
		}
		if compare := strings.Compare(left.DisplayName, right.DisplayName); compare != 0 {
			return compare
		}
		return int(left.ID - right.ID)
	})
	slices.SortFunc(state.Trunks, func(left OperatorTrunkState, right OperatorTrunkState) int {
		return strings.Compare(left.Name, right.Name)
	})
	slices.SortFunc(state.Conferences, func(left OperatorConferenceState, right OperatorConferenceState) int {
		return strings.Compare(left.Name, right.Name)
	})
	slices.SortFunc(state.ParkedCalls, func(left ParkedCallState, right ParkedCallState) int {
		return strings.Compare(left.Slot, right.Slot)
	})
	slices.SortFunc(state.ActiveCalls, func(left OperatorActiveCall, right OperatorActiveCall) int {
		return int(left.ID - right.ID)
	})
}
