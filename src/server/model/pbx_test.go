package model

import "testing"

func TestPBXPlanHelpers(t *testing.T) {
	if NormalizePBXField("  demo  ") != "demo" {
		t.Fatalf("expected pbx field normalization")
	}
	if ValidatePBXField("") != ErrPBXFieldRequired {
		t.Fatalf("expected required pbx field error")
	}
	if ValidatePBXField("demo") != nil {
		t.Fatalf("expected valid pbx field")
	}

	plan := NormalizePBXPlan(PBXPlan{})
	if len(plan.Extensions) != 0 || len(plan.ProvisioningProfiles) != 0 {
		t.Fatalf("expected normalized empty pbx plan %+v", plan)
	}
	defaultPlan := DefaultPBXPlan()
	if len(defaultPlan.Routes) != 0 || len(defaultPlan.IVRs) != 0 {
		t.Fatalf("unexpected default pbx plan %+v", defaultPlan)
	}

	plan = PBXPlan{
		Extensions:           []Extension{{Number: "2000"}, {Number: "1000"}},
		Trunks:               []Trunk{{Name: "zeta"}, {Name: "alpha"}},
		Routes:               []CallRoute{{Name: "night"}, {Name: "day"}},
		Queues:               []Queue{{Name: "support"}, {Name: "billing"}},
		Conferences:          []Conference{{Name: "sales"}, {Name: "all-hands"}},
		IVRs:                 []IVR{{Name: "main"}, {Name: "after-hours"}},
		PromptAssignments:    []PromptAssignment{{Name: "welcome"}, {Name: "after-hours"}},
		ProvisioningProfiles: []ProvisioningProfile{{Name: "yealink"}, {Name: "browser"}},
	}
	SortPBXPlan(&plan)
	if plan.Extensions[0].Number != "1000" || plan.Trunks[0].Name != "alpha" || plan.Queues[0].Name != "billing" || plan.ProvisioningProfiles[0].Name != "browser" {
		t.Fatalf("expected sorted pbx plan %+v", plan)
	}
}
