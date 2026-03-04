package apply

import (
	"fmt"
	"strings"
)

// Action represents what will happen to a resource during apply.
type Action int

const (
	ActionSkip   Action = iota // No changes needed
	ActionCreate               // Resource will be created
	ActionUpdate               // Resource will be updated
	ActionDelete               // Resource will be deleted
)

// Symbol returns a single-character indicator for the action.
func (a Action) Symbol() string {
	switch a {
	case ActionSkip:
		return "="
	case ActionCreate:
		return "+"
	case ActionUpdate:
		return "~"
	case ActionDelete:
		return "-"
	default:
		return "?"
	}
}

// String returns a human-readable name for the action.
func (a Action) String() string {
	switch a {
	case ActionSkip:
		return "unchanged"
	case ActionCreate:
		return "create"
	case ActionUpdate:
		return "update"
	case ActionDelete:
		return "delete"
	default:
		return "unknown"
	}
}

// ResourceType identifies the kind of Semaphore resource.
type ResourceType string

const (
	ResourceProject     ResourceType = "project"
	ResourceKey         ResourceType = "keys"
	ResourceVariableGroup ResourceType = "variable_groups"
	ResourceRepository  ResourceType = "repositories"
	ResourceInventory   ResourceType = "inventories"
	ResourceTemplate    ResourceType = "templates"
	ResourceSchedule    ResourceType = "schedules"
)

// ResourceAction describes a planned action for a single resource.
type ResourceAction struct {
	Type        ResourceType
	Action      Action
	Label       string // Display name (e.g. resource name)
	Description string // Optional detail about what changed
	ExistingID  int64  // ID of existing resource (0 if new)
	Index       int    // Index in the config slice
}

// Plan holds all planned actions for an apply operation.
type Plan struct {
	Actions []ResourceAction
}

// HasChanges returns true if any action is not ActionSkip.
func (p *Plan) HasChanges() bool {
	for _, a := range p.Actions {
		if a.Action != ActionSkip {
			return true
		}
	}
	return false
}

// Summary returns counts of each action type.
func (p *Plan) Summary() (creates, updates, deletes, unchanged int) {
	for _, a := range p.Actions {
		switch a.Action {
		case ActionCreate:
			creates++
		case ActionUpdate:
			updates++
		case ActionDelete:
			deletes++
		case ActionSkip:
			unchanged++
		}
	}
	return
}

// ActionsByType returns actions filtered by resource type.
func (p *Plan) ActionsByType(rt ResourceType) []ResourceAction {
	var result []ResourceAction
	for _, a := range p.Actions {
		if a.Type == rt {
			result = append(result, a)
		}
	}
	return result
}

// FormatPlan returns a human-readable plan summary.
func (p *Plan) FormatPlan() string {
	if len(p.Actions) == 0 {
		return "No resources to manage."
	}

	var sb strings.Builder

	// Group by resource type, preserving order
	types := []ResourceType{
		ResourceProject,
		ResourceKey,
		ResourceVariableGroup,
		ResourceRepository,
		ResourceInventory,
		ResourceTemplate,
		ResourceSchedule,
	}

	for _, rt := range types {
		actions := p.ActionsByType(rt)
		if len(actions) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("\n%s:\n", rt))
		for _, a := range actions {
			line := fmt.Sprintf("  %s %s", a.Action.Symbol(), a.Label)
			if a.Description != "" {
				line += fmt.Sprintf(" (%s)", a.Description)
			}
			sb.WriteString(line + "\n")
		}
	}

	creates, updates, deletes, unchanged := p.Summary()
	sb.WriteString(fmt.Sprintf("\nPlan: %d to create, %d to update, %d to delete, %d unchanged.\n",
		creates, updates, deletes, unchanged))

	return sb.String()
}
