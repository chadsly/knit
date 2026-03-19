package agents

import (
	"strings"
)

const (
	IntentImplementChanges = "implement_changes"
	IntentDraftPlan        = "draft_plan"
	IntentCreateJira       = "create_jira_tickets"
)

const maxCustomInstructionsLen = 2000
const maxInstructionTextLen = 12000

type DeliveryIntent struct {
	Profile            string `json:"intent_profile,omitempty"`
	InstructionText    string `json:"instruction_text,omitempty"`
	CustomInstructions string `json:"custom_instructions,omitempty"`
}

func NormalizeDeliveryIntent(intent DeliveryIntent) DeliveryIntent {
	profile := strings.TrimSpace(strings.ToLower(intent.Profile))
	switch profile {
	case IntentDraftPlan, IntentCreateJira:
	default:
		profile = IntentImplementChanges
	}
	instructionText := strings.TrimSpace(intent.InstructionText)
	if len(instructionText) > maxInstructionTextLen {
		instructionText = strings.TrimSpace(instructionText[:maxInstructionTextLen])
	}
	custom := strings.TrimSpace(intent.CustomInstructions)
	if len(custom) > maxCustomInstructionsLen {
		custom = strings.TrimSpace(custom[:maxCustomInstructionsLen])
	}
	return DeliveryIntent{
		Profile:            profile,
		InstructionText:    instructionText,
		CustomInstructions: custom,
	}
}

func (intent DeliveryIntent) Label() string {
	switch NormalizeDeliveryIntent(intent).Profile {
	case IntentDraftPlan:
		return "Draft plan"
	case IntentCreateJira:
		return "Create Jira tickets"
	default:
		return "Implement changes"
	}
}

func (intent DeliveryIntent) AllowsTrackerOps() bool {
	return NormalizeDeliveryIntent(intent).Profile == IntentCreateJira
}

func RenderInstructionText(intent DeliveryIntent) string {
	intent = NormalizeDeliveryIntent(intent)
	base := strings.TrimSpace(intent.InstructionText)
	if base == "" {
		base = DefaultInstructionTemplate(intent.Profile)
	}
	if intent.CustomInstructions != "" {
		base = strings.TrimSpace(base) + "\n\nAdditional instructions from the Knit user:\n" + intent.CustomInstructions
	}
	return strings.TrimSpace(base)
}

func DefaultInstructionTemplate(profile string) string {
	switch NormalizeDeliveryIntent(DeliveryIntent{Profile: profile}).Profile {
	case IntentDraftPlan:
		return strings.Join([]string{
			"You are receiving a canonical Knit feedback payload JSON.",
			"Produce a concrete implementation plan for the requested software changes without editing the repository.",
			"",
			"Rules:",
			"- Do not edit files or make repository changes.",
			"- Focus on scope, intended behavior, risks, sequencing, and validation steps.",
			"- Call out assumptions and edge cases clearly.",
			"- Return a concise, actionable plan rather than implementation code.",
			"- If the repository is already dirty, use that only as context; do not modify it.",
			"",
			"External tracker operations are disabled for this run.",
			"Do not call Jira/Atlassian/GitHub issue tools.",
		}, "\n")
	case IntentCreateJira:
		return strings.Join([]string{
			"You are receiving a canonical Knit feedback payload JSON.",
			"Turn the approved feedback into Jira-ready implementation tickets.",
			"",
			"Rules:",
			"- Create a clear breakdown of work with concise titles, descriptions, and acceptance criteria.",
			"- If tracker tools are available, you may create or update Jira issues.",
			"- If tracker tools are unavailable, return a structured ticket bundle that can be copied into Jira.",
			"- Do not modify the repository unless it is necessary to inspect the codebase and understand the requested work.",
			"- Focus on actionable project-management output, not code edits.",
			"",
			"External tracker operations are allowed for this run when they help complete the requested Jira-ticket workflow.",
		}, "\n")
	default:
		return strings.Join([]string{
			"You are receiving a canonical Knit feedback payload JSON.",
			"Implement the requested software changes in the current repository.",
			"",
			"Rules:",
			"- Apply changes directly in the working tree.",
			"- Keep edits minimal and maintainable.",
			"- Run relevant tests.",
			"- Return a short summary at the end.",
			"- Focus on implementation and validation, not project-management workflow.",
			"- If the repository is already dirty, continue and only modify files required for this request.",
			"- Do not stop to ask \"how should I proceed\" in exec mode; make the safest minimal assumption and continue.",
			"- Do not exit after only creating/updating tickets, issues, or external tracker metadata.",
			"",
			"External tracker operations are disabled for this run.",
			"Do not call Jira/Atlassian/GitHub issue tools; implement code + tests directly.",
		}, "\n")
	}
}
