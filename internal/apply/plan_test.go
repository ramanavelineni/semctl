package apply

import (
	"strings"
	"testing"
)

func TestActionSymbol(t *testing.T) {
	tests := []struct {
		action Action
		want   string
	}{
		{ActionSkip, "="},
		{ActionCreate, "+"},
		{ActionUpdate, "~"},
		{ActionDelete, "-"},
	}
	for _, tt := range tests {
		if got := tt.action.Symbol(); got != tt.want {
			t.Errorf("Action(%d).Symbol() = %q, want %q", tt.action, got, tt.want)
		}
	}
}

func TestActionString(t *testing.T) {
	tests := []struct {
		action Action
		want   string
	}{
		{ActionSkip, "unchanged"},
		{ActionCreate, "create"},
		{ActionUpdate, "update"},
		{ActionDelete, "delete"},
	}
	for _, tt := range tests {
		if got := tt.action.String(); got != tt.want {
			t.Errorf("Action(%d).String() = %q, want %q", tt.action, got, tt.want)
		}
	}
}

func TestPlanHasChanges(t *testing.T) {
	// Empty plan
	p := &Plan{}
	if p.HasChanges() {
		t.Error("empty plan should not have changes")
	}

	// All skip
	p = &Plan{Actions: []ResourceAction{
		{Action: ActionSkip},
		{Action: ActionSkip},
	}}
	if p.HasChanges() {
		t.Error("all-skip plan should not have changes")
	}

	// Has create
	p = &Plan{Actions: []ResourceAction{
		{Action: ActionSkip},
		{Action: ActionCreate},
	}}
	if !p.HasChanges() {
		t.Error("plan with create should have changes")
	}
}

func TestPlanSummary(t *testing.T) {
	p := &Plan{Actions: []ResourceAction{
		{Action: ActionCreate},
		{Action: ActionCreate},
		{Action: ActionUpdate},
		{Action: ActionDelete},
		{Action: ActionSkip},
		{Action: ActionSkip},
		{Action: ActionSkip},
	}}

	creates, updates, deletes, unchanged := p.Summary()
	if creates != 2 {
		t.Errorf("creates = %d, want 2", creates)
	}
	if updates != 1 {
		t.Errorf("updates = %d, want 1", updates)
	}
	if deletes != 1 {
		t.Errorf("deletes = %d, want 1", deletes)
	}
	if unchanged != 3 {
		t.Errorf("unchanged = %d, want 3", unchanged)
	}
}

func TestPlanActionsByType(t *testing.T) {
	p := &Plan{Actions: []ResourceAction{
		{Type: ResourceKey, Action: ActionCreate, Label: "k1"},
		{Type: ResourceEnvironment, Action: ActionCreate, Label: "e1"},
		{Type: ResourceKey, Action: ActionSkip, Label: "k2"},
	}}

	keys := p.ActionsByType(ResourceKey)
	if len(keys) != 2 {
		t.Fatalf("ActionsByType(ResourceKey) returned %d, want 2", len(keys))
	}
	if keys[0].Label != "k1" || keys[1].Label != "k2" {
		t.Errorf("unexpected key labels: %q, %q", keys[0].Label, keys[1].Label)
	}

	envs := p.ActionsByType(ResourceEnvironment)
	if len(envs) != 1 {
		t.Fatalf("ActionsByType(ResourceEnvironment) returned %d, want 1", len(envs))
	}
}

func TestFormatPlan(t *testing.T) {
	p := &Plan{Actions: []ResourceAction{
		{Type: ResourceProject, Action: ActionSkip, Label: "My Project"},
		{Type: ResourceKey, Action: ActionCreate, Label: "SSH Key"},
		{Type: ResourceKey, Action: ActionDelete, Label: "Old Key"},
	}}

	output := p.FormatPlan()

	if !strings.Contains(output, "+ SSH Key") {
		t.Error("FormatPlan missing '+ SSH Key'")
	}
	if !strings.Contains(output, "- Old Key") {
		t.Error("FormatPlan missing '- Old Key'")
	}
	if !strings.Contains(output, "= My Project") {
		t.Error("FormatPlan missing '= My Project'")
	}
	if !strings.Contains(output, "1 to create") {
		t.Error("FormatPlan missing '1 to create'")
	}
	if !strings.Contains(output, "1 to delete") {
		t.Error("FormatPlan missing '1 to delete'")
	}
}

func TestFormatPlanEmpty(t *testing.T) {
	p := &Plan{}
	output := p.FormatPlan()
	if !strings.Contains(output, "No resources") {
		t.Error("empty plan FormatPlan should mention 'No resources'")
	}
}

func TestFormatPlanWithDescription(t *testing.T) {
	p := &Plan{Actions: []ResourceAction{
		{Type: ResourceKey, Action: ActionUpdate, Label: "My Key", Description: "secrets always re-applied"},
	}}

	output := p.FormatPlan()
	if !strings.Contains(output, "(secrets always re-applied)") {
		t.Errorf("FormatPlan missing description, got:\n%s", output)
	}
}
