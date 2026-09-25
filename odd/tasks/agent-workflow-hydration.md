# agent-workflow-hydration

Hydrate an agent workflow file from `.dflow.yaml` configuration.

## Issue

Closes #31

## Checklist

- [ ] Generator: `pkg/agent/generate.go` — `GenerateAgentDoc(cfg *flow.Config) []byte`
- [ ] Command: `cmd/commands/agent.go` — `dflow agent [--path] [--force] [--json]`
- [ ] Init integration: prompt in `dflow init` after `.dflow.yaml` creation
- [ ] AGENTS.md reference: append workflow reference if file exists or create it
- [ ] Tests: golden file + idempotency + init integration
- [ ] Docs: README.md + command help text

## Commit identity

_Work-unit commits will be recorded here as they land._
