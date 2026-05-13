package model

import (
	"errors"
	"slices"
	"strings"
	"time"
)

var ErrPBXFieldRequired = errors.New("pbx field is required")

type Extension struct {
	ID               int64     `json:"id"`
	TenantID         int64     `json:"tenant_id,omitempty"`
	OrganizationID   int64     `json:"organization_id,omitempty"`
	UserID           int64     `json:"user_id,omitempty"`
	Number           string    `json:"number"`
	DisplayName      string    `json:"display_name"`
	Technology       string    `json:"technology"`
	Endpoint         string    `json:"endpoint,omitempty"`
	VoicemailEnabled bool      `json:"voicemail_enabled"`
	CreatedAt        time.Time `json:"created_at,omitempty"`
	UpdatedAt        time.Time `json:"updated_at,omitempty"`
}

type Trunk struct {
	ID         int64     `json:"id"`
	TenantID   int64     `json:"tenant_id,omitempty"`
	Name       string    `json:"name"`
	Technology string    `json:"technology"`
	Endpoint   string    `json:"endpoint"`
	Active     bool      `json:"active"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
	UpdatedAt  time.Time `json:"updated_at,omitempty"`
}

type CallRoute struct {
	ID          int64     `json:"id"`
	TenantID    int64     `json:"tenant_id,omitempty"`
	Name        string    `json:"name"`
	Direction   string    `json:"direction"`
	Match       string    `json:"match,omitempty"`
	Destination string    `json:"destination"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

type Queue struct {
	ID                     int64     `json:"id"`
	TenantID               int64     `json:"tenant_id,omitempty"`
	OrganizationID         int64     `json:"organization_id,omitempty"`
	Name                   string    `json:"name"`
	Strategy               string    `json:"strategy,omitempty"`
	MemberExtensionNumbers []string  `json:"member_extension_numbers,omitempty"`
	CreatedAt              time.Time `json:"created_at,omitempty"`
	UpdatedAt              time.Time `json:"updated_at,omitempty"`
}

type Conference struct {
	ID               int64     `json:"id"`
	TenantID         int64     `json:"tenant_id,omitempty"`
	OrganizationID   int64     `json:"organization_id,omitempty"`
	UserID           int64     `json:"user_id,omitempty"`
	Name             string    `json:"name"`
	AccessCode       string    `json:"access_code,omitempty"`
	RecordingEnabled bool      `json:"recording_enabled"`
	CreatedAt        time.Time `json:"created_at,omitempty"`
	UpdatedAt        time.Time `json:"updated_at,omitempty"`
}

type IVR struct {
	ID                 int64     `json:"id"`
	TenantID           int64     `json:"tenant_id,omitempty"`
	Name               string    `json:"name"`
	RootPrompt         string    `json:"root_prompt,omitempty"`
	DefaultDestination string    `json:"default_destination"`
	TimeoutSeconds     int       `json:"timeout_seconds,omitempty"`
	CreatedAt          time.Time `json:"created_at,omitempty"`
	UpdatedAt          time.Time `json:"updated_at,omitempty"`
}

type PromptAssignment struct {
	ID             int64     `json:"id"`
	TenantID       int64     `json:"tenant_id,omitempty"`
	OrganizationID int64     `json:"organization_id,omitempty"`
	Name           string    `json:"name"`
	PromptName     string    `json:"prompt_name"`
	TargetKind     string    `json:"target_kind"`
	TargetRef      string    `json:"target_ref"`
	CreatedAt      time.Time `json:"created_at,omitempty"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
}

type ProvisioningProfile struct {
	ID                 int64     `json:"id"`
	TenantID           int64     `json:"tenant_id,omitempty"`
	Name               string    `json:"name"`
	Technology         string    `json:"technology"`
	Template           string    `json:"template"`
	AssignedExtensions []string  `json:"assigned_extensions,omitempty"`
	CreatedAt          time.Time `json:"created_at,omitempty"`
	UpdatedAt          time.Time `json:"updated_at,omitempty"`
}

type PBXPlan struct {
	Extensions           []Extension           `json:"extensions,omitempty"`
	Trunks               []Trunk               `json:"trunks,omitempty"`
	Routes               []CallRoute           `json:"routes,omitempty"`
	Queues               []Queue               `json:"queues,omitempty"`
	Conferences          []Conference          `json:"conferences,omitempty"`
	IVRs                 []IVR                 `json:"ivrs,omitempty"`
	PromptAssignments    []PromptAssignment    `json:"prompt_assignments,omitempty"`
	ProvisioningProfiles []ProvisioningProfile `json:"provisioning_profiles,omitempty"`
	UpdatedAt            time.Time             `json:"updated_at,omitempty"`
}

func DefaultPBXPlan() PBXPlan {
	return PBXPlan{
		Extensions:           []Extension{},
		Trunks:               []Trunk{},
		Routes:               []CallRoute{},
		Queues:               []Queue{},
		Conferences:          []Conference{},
		IVRs:                 []IVR{},
		PromptAssignments:    []PromptAssignment{},
		ProvisioningProfiles: []ProvisioningProfile{},
	}
}

func NormalizePBXField(value string) string {
	return strings.TrimSpace(value)
}

func ValidatePBXField(value string) error {
	if NormalizePBXField(value) == "" {
		return ErrPBXFieldRequired
	}
	return nil
}

func NormalizePBXPlan(plan PBXPlan) PBXPlan {
	if plan.Extensions == nil {
		plan.Extensions = []Extension{}
	}
	if plan.Trunks == nil {
		plan.Trunks = []Trunk{}
	}
	if plan.Routes == nil {
		plan.Routes = []CallRoute{}
	}
	if plan.Queues == nil {
		plan.Queues = []Queue{}
	}
	if plan.Conferences == nil {
		plan.Conferences = []Conference{}
	}
	if plan.IVRs == nil {
		plan.IVRs = []IVR{}
	}
	if plan.PromptAssignments == nil {
		plan.PromptAssignments = []PromptAssignment{}
	}
	if plan.ProvisioningProfiles == nil {
		plan.ProvisioningProfiles = []ProvisioningProfile{}
	}
	return plan
}

func SortPBXPlan(plan *PBXPlan) {
	slices.SortFunc(plan.Extensions, func(left Extension, right Extension) int { return strings.Compare(left.Number, right.Number) })
	slices.SortFunc(plan.Trunks, func(left Trunk, right Trunk) int { return strings.Compare(left.Name, right.Name) })
	slices.SortFunc(plan.Routes, func(left CallRoute, right CallRoute) int { return strings.Compare(left.Name, right.Name) })
	slices.SortFunc(plan.Queues, func(left Queue, right Queue) int { return strings.Compare(left.Name, right.Name) })
	slices.SortFunc(plan.Conferences, func(left Conference, right Conference) int { return strings.Compare(left.Name, right.Name) })
	slices.SortFunc(plan.IVRs, func(left IVR, right IVR) int { return strings.Compare(left.Name, right.Name) })
	slices.SortFunc(plan.PromptAssignments, func(left PromptAssignment, right PromptAssignment) int { return strings.Compare(left.Name, right.Name) })
	slices.SortFunc(plan.ProvisioningProfiles, func(left ProvisioningProfile, right ProvisioningProfile) int {
		return strings.Compare(left.Name, right.Name)
	})
}
