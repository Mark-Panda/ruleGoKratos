# Default MCP For Agents Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make every Agent load all enabled MCP configurations by default, matching current Skill behavior and removing per-Agent MCP checkboxes.

**Architecture:** Runtime MCP access is derived only from globally enabled MCP config rows. Frontend removes MCP selection UI from Managed Agent and Agent Harness node forms, and Agent Harness DSL no longer carries MCP switch or allowlist fields.

**Tech Stack:** Go/Kratos backend, RuleGo custom node component, React/Semi UI Flowgram frontend, Go tests and Vitest mapping tests.

---

### Task 1: Backend Managed Agent Runtime

**Files:**
- Modify: `internal/biz/managed_agent_harness.go`
- Modify: `internal/biz/managed_agent_enrich.go`
- Modify: `internal/biz/managed_agent_enrich_test.go`
- Modify: `internal/data/managed_agent_harness_loader.go`
- Modify: `internal/data/dao/mcp_config.go`

- [ ] Write failing test showing managed Agent ignores selected MCP ids and enables all globally enabled MCP servers.
- [ ] Run `go test ./internal/biz -run TestEnrichHarnessWithManagedAgent -count=1` and confirm the new test fails because only selected MCP allowlist is used.
- [ ] Add loader method for globally enabled MCP allowlist and use it in managed Agent enrichment.
- [ ] Run `go test ./internal/biz -run TestEnrichHarnessWithManagedAgent -count=1` and confirm it passes.

### Task 2: Agent Harness Runtime Defaults

**Files:**
- Modify: `internal/data/components/agent_harness_llm.go`

- [ ] Write or adjust a focused test if an existing component test covers tool options.
- [ ] Make Agent Harness node runtime enable MCP by default and avoid carrying node-level MCP allowlist restrictions.
- [ ] Run the focused Go test for Agent Harness component behavior.

### Task 3: Frontend UI Removal

**Files:**
- Modify: `flowgram/src/management/sections/ManagedAgentsSection.tsx`
- Modify: `flowgram/src/nodes/agent-harness/form-meta.tsx`
- Modify: `flowgram/src/nodes/agent-harness/index.ts`
- Modify: `flowgram/src/utils/rulechain-builder.ts`

- [ ] Remove Managed Agent MCP checkbox UI and MCP table column.
- [ ] Remove Agent Harness MCP allowlist checkbox UI and related imports/helpers.
- [ ] Update descriptions so users know all enabled MCP configs are auto-loaded.
- [ ] Remove node-level MCP config fields instead of preserving legacy DSL compatibility.

### Task 4: DSL Mapping Cleanup

**Files:**
- Modify: `flowgram/src/utils/dsl-mapping/specs.ts`
- Modify: `flowgram/src/utils/dsl-mapping/__tests__/mapping.spec.ts`

- [ ] Update mapping expectations so new exports do not require manual MCP allowlist configuration.
- [ ] Remove `enableMcpTool` and `mcpAllowlist` from the Agent Harness mapping spec.
- [ ] Run `npm run test:unit` in `flowgram`.

### Task 5: Final Verification

**Files:**
- Check recently edited Go and TS/TSX files.

- [ ] Run focused Go tests for backend changes.
- [ ] Run frontend unit tests affected by mapping/UI changes.
- [ ] Run lints/diagnostics for edited files.
