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

- `47a1de2` — feat(agent): generate agent workflow from .dflow.yaml (pkg/agent, cmd/commands/agent.go, init integration, tests, docs)
