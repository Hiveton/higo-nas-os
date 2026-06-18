package mcp

import (
	"context"
	"encoding/json"

	"higoos/server-go/internal/apiclient"
)

// AgentCreateInput is the body for instantiating an agent from a template.
type AgentCreateInput struct {
	TemplateID string   `json:"templateId" jsonschema:"agent template id to instantiate"`
	Name       string   `json:"name,omitempty" jsonschema:"display name for the new agent"`
	Scopes     []string `json:"scopes,omitempty" jsonschema:"scopes to grant the agent"`
}

// AgentToolsInput identifies an agent whose tools to list.
type AgentToolsInput struct {
	ID string `json:"id" jsonschema:"agent id to list tools for"`
}

// WorkflowPreviewInput is the body for previewing a workflow plan.
type WorkflowPreviewInput struct {
	ActorID    string   `json:"actorId,omitempty" jsonschema:"id of the actor requesting the preview"`
	TemplateID string   `json:"templateId,omitempty" jsonschema:"agent template id backing the workflow"`
	Goal       string   `json:"goal" jsonschema:"natural language goal for the workflow"`
	Scopes     []string `json:"scopes,omitempty" jsonschema:"scopes available to the workflow"`
}

// WorkflowStartRunInput is the body for starting a workflow run.
type WorkflowStartRunInput struct {
	ActorID    string   `json:"actorId,omitempty" jsonschema:"id of the actor starting the run"`
	TemplateID string   `json:"templateId,omitempty" jsonschema:"agent template id backing the workflow"`
	Goal       string   `json:"goal" jsonschema:"natural language goal for the workflow run"`
	Scopes     []string `json:"scopes,omitempty" jsonschema:"scopes available to the workflow run"`
}

// WorkflowConfirmRunInput is the body for confirming a pending workflow run.
type WorkflowConfirmRunInput struct {
	ID             string `json:"id" jsonschema:"workflow run id to confirm"`
	ActorID        string `json:"actorId,omitempty" jsonschema:"id of the actor confirming the run"`
	ConfirmationID string `json:"confirmationId,omitempty" jsonschema:"confirmationId returned by the run start"`
}

// WorkflowCancelRunInput is the body for cancelling a workflow run.
type WorkflowCancelRunInput struct {
	ID      string `json:"id" jsonschema:"workflow run id to cancel"`
	ActorID string `json:"actorId,omitempty" jsonschema:"id of the actor cancelling the run"`
	Reason  string `json:"reason,omitempty" jsonschema:"reason for cancelling the run"`
}

// WorkflowRunEventsInput identifies a workflow run whose events to fetch.
type WorkflowRunEventsInput struct {
	ID string `json:"id" jsonschema:"workflow run id to fetch events for"`
}

func registerAgents(r *registry) {
	addTool(r, "agents", "higo.agents.templates.list",
		"List available agent templates.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.AgentTemplates(ctx)
		})

	addTool(r, "agents", "higo.agents.create",
		"Create an agent instance from a template.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in AgentCreateInput) (json.RawMessage, error) {
			return c.AgentCreate(ctx, in)
		})

	addTool(r, "agents", "higo.agents.tools.list",
		"List the tools available to an agent.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in AgentToolsInput) (json.RawMessage, error) {
			return c.AgentTools(ctx, in.ID)
		})

	addTool(r, "agents", "higo.workflows.preview",
		"Preview a workflow plan, its impact and confirmation checkpoints.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in WorkflowPreviewInput) (json.RawMessage, error) {
			return c.WorkflowPreview(ctx, in)
		})

	addTool(r, "agents", "higo.workflows.runs.start",
		"Start a workflow run for a goal.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in WorkflowStartRunInput) (json.RawMessage, error) {
			return c.WorkflowStartRun(ctx, in)
		})

	addTool(r, "agents", "higo.workflows.runs.confirm",
		"Confirm a previously-returned pending workflow run by its confirmationId.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in WorkflowConfirmRunInput) (json.RawMessage, error) {
			return c.WorkflowConfirmRun(ctx, in.ID, map[string]any{
				"actorId":        in.ActorID,
				"confirmationId": in.ConfirmationID,
			})
		})

	addTool(r, "agents", "higo.workflows.runs.cancel",
		"Cancel a workflow run.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in WorkflowCancelRunInput) (json.RawMessage, error) {
			return c.WorkflowCancelRun(ctx, in.ID, map[string]any{
				"actorId": in.ActorID,
				"reason":  in.Reason,
			})
		})

	addTool(r, "agents", "higo.workflows.runs.events",
		"Get the current events for a workflow run.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in WorkflowRunEventsInput) (json.RawMessage, error) {
			return c.WorkflowRunEvents(ctx, in.ID)
		})
}
