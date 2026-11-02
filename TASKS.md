# TASKS

Live progress tracker for **Integrated Personal Budget**.

## Status legend

| Mark | Meaning |
| --- | --- |
| `[ ]` | Not started |
| `[~]` | In progress |
| `[x]` | Done |
| `[!]` | Blocked or awaiting a decision |

## Rules

- One task per logical change; one semantic commit per completed task.
- Update this file in the same commit as the work it describes.
- Mark blocked tasks `[!]` and write the blocker and owner next to them.
- Do not start implementation tasks before `T-005` is approved.
- Add the commit subject in the **Commit** column so progress is traceable from
  `git log` alone.

## Current phase

**Phase 0 — Specification and approval.** No application code has been written.

## Milestone 0 — Project foundations

| ID | Task | Status | Commit |
| --- | --- | --- | --- |
| T-001 | Inspect repository and existing README | `[x]` | `docs: add initial project README` |
| T-002 | Create `AGENTS.md` with working rules and conventions | `[x]` | `docs: add agent guidelines and task tracker` |
| T-003 | Write `docs/TECH_SPEC.md` covering architecture, stack, data model, API, AI/MCP, UI, deployment, security, and tests | `[x]` | `docs: add technical specification for review` |
| T-004 | Update `README.md` with project status, architecture summary, and documentation links | `[x]` | `docs: refresh README with project status and links` |
| T-005 | **Maintainer review and approval of the technical specification** | `[!]` | — |
| T-006 | Resolve the open decisions listed in the spec | `[!]` | — |
| T-007 | Scaffold repository layout, toolchain, and `Makefile` | `[ ]` | — |
| T-008 | Add CI pipeline and baseline quality gates | `[ ]` | — |
| T-009 | Revise deployment from Kubernetes to rootless Podman Quadlet with symlinked units | `[x]` | `docs(deploy): replace Kubernetes with rootless Podman Quadlet` |
| T-010 | Record approved/deferred decisions and expand the user-model options | `[x]` | `docs(spec): record decisions and expand user model options` |

## Milestone 1 — Backend core

| ID | Task | Status | Commit |
| --- | --- | --- | --- |
| T-101 | Database connection, migrations, and schema for accounts, transactions, tags, and budgets | `[ ]` | — |
| T-102 | Account CRUD and balance calculation | `[ ]` | — |
| T-103 | Tag CRUD | `[ ]` | — |
| T-104 | Transaction CRUD with tags, signed amounts, and filtering | `[ ]` | — |
| T-105 | Budget CRUD and tag-based budget summaries | `[ ]` | — |
| T-106 | Spending summary and recommendation endpoints | `[ ]` | — |
| T-107 | OpenAPI contract and request/response validation | `[ ]` | — |
| T-108 | Authentication (bearer tokens and/or tailnet identity) and rate limiting | `[ ]` | — |

## Milestone 2 — Web frontend

| ID | Task | Status | Commit |
| --- | --- | --- | --- |
| T-201 | Frontend scaffold, design tokens, and responsive layout | `[ ]` | — |
| T-202 | Dashboard with balance, spending, and budget widgets | `[ ]` | — |
| T-203 | Transaction list, filters, and create/edit flow | `[ ]` | — |
| T-204 | Budget and account management screens | `[ ]` | — |
| T-205 | Spending charts and what-if simulation | `[ ]` | — |
| T-206 | PWA polish, accessibility pass, and performance budget check | `[ ]` | — |

## Milestone 3 — AI, MCP, and WhatsApp

| ID | Task | Status | Commit |
| --- | --- | --- | --- |
| T-301 | MCP server with transaction, query, budget, and receipt tools | `[ ]` | — |
| T-302 | AI provider abstraction and chat-to-transaction draft flow | `[ ]` | — |
| T-303 | WhatsApp webhook with signature verification, allowlist, and idempotency | `[!]` | Deferred by D5 |
| T-304 | Receipt upload and vision-based OCR into a confirmable draft | `[ ]` | — |
| T-305 | Audit trail for AI-assisted writes | `[ ]` | — |

## Milestone 4 — Deployment and operations

| ID | Task | Status | Commit |
| --- | --- | --- | --- |
| T-401 | Multi-stage container image with embedded frontend | `[ ]` | — |
| T-402 | Quadlet pod and container units with rootless config, data mounts, health checks, and hardening | `[ ]` | — |
| T-403 | Tailscale sidecar via Quadlet and tailnet-only Serve exposure | `[ ]` | — |
| T-404 | Backup and restore procedure for SQLite | `[ ]` | — |
| T-405 | Operational runbook, health checks, journald logging, and upgrade/rollback steps | `[ ]` | — |
| T-406 | Install docs for symlinking units into `~/.config/containers/systemd`, lingering, start, and uninstall | `[ ]` | — |

## Open decisions blocking T-006

These are recorded as D1–D13 in `docs/TECH_SPEC.md` §16.

**Still blocking:**

- [ ] D2 — Choose **A (one shared ledger)** or **B (multi-workspace)**. Recommended: B.
- [ ] D6 — Provide miniPC details: OS/distro, architecture, Podman version, SELinux state, deployment user, hostname.

**Resolved:**

- [x] D1 — Approved: Go + SQLite + SvelteKit + rootless Quadlet.
- [x] D3 — Approved: tailnet identity + hashed API tokens.
- [x] D4 — Approved: pluggable OpenAI-compatible AI adapter.
- [x] D5 — Deferred for v1; keep the narrow Funnel webhook design for later.
- [x] D7 — Deferred; build local backup now, choose off-host destination later.

**Later (D8–D13):**

- [ ] D8 — Currency defaults.
- [ ] D9 — Offline behavior.
- [ ] D10 — License.
- [ ] D11 — CI host.
- [ ] D12 — Recommendation engine.
- [ ] D13 — Receipt privacy.

## Changelog

| Date | Change |
| --- | --- |
| 2026-11-02 | Initial task list; specification phase opened. |
| 2026-11-02 | Deployment revised to rootless Podman Quadlet; decisions prioritized. |
| 2026-11-02 | D1/D3/D4 approved; D5/D7 deferred; D2 options expanded; D6 moved to miniPC. |
