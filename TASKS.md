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
- Follow the milestone order in `docs/TECH_SPEC.md`; the specification was
  approved on 2026-11-02 (`T-005`).
- Add the commit subject in the **Commit** column so progress is traceable from
  `git log` alone.

## Current phase

**Phase 3 — Web frontend.** Backend core (M1) is complete. The app shell,
dashboard with charts, transaction flow, and budget/account screens are in
place. Next: T-206 PWA polish and accessibility pass.

**Known issue:** `make lint` reports pre-existing formatting and ESLint errors in
older account and budget screens, unrelated to T-205. Tracked as T-209.

## Milestone 0 — Project foundations

| ID | Task | Status | Commit |
| --- | --- | --- | --- |
| T-001 | Inspect repository and existing README | `[x]` | `docs: add initial project README` |
| T-002 | Create `AGENTS.md` with working rules and conventions | `[x]` | `docs: add agent guidelines and task tracker` |
| T-003 | Write `docs/TECH_SPEC.md` covering architecture, stack, data model, API, AI/MCP, UI, deployment, security, and tests | `[x]` | `docs: add technical specification for review` |
| T-004 | Update `README.md` with project status, architecture summary, and documentation links | `[x]` | `docs: refresh README with project status and links` |
| T-005 | **Maintainer review and approval of the technical specification** | `[x]` | `docs: approve technical specification` |
| T-006 | Resolve the open decisions listed in the spec | `[x]` | `docs(spec): finalize D2 and deployment details` |
| T-007 | Scaffold repository layout, toolchain, and `Makefile` | `[x]` | `chore: scaffold repository and toolchain` |
| T-008 | Add CI pipeline and baseline quality gates | `[x]` | `ci: add GitHub Actions quality gate` |
| T-009 | Revise deployment from Kubernetes to rootless Podman Quadlet with symlinked units | `[x]` | `docs(deploy): replace Kubernetes with rootless Podman Quadlet` |
| T-010 | Record approved/deferred decisions and expand the user-model options | `[x]` | `docs(spec): record decisions and expand user model options` |
| T-011 | Update README for multi-workspace budgeting and the miniPC target | `[x]` | `docs: mention multi-workspace budgeting in README` |

## Milestone 1 — Backend core

| ID | Task | Status | Commit |
| --- | --- | --- | --- |
| T-101 | Database connection, migrations, and workspace-scoped schema for users, workspaces, accounts, transactions, tags, and budgets | `[x]` | `feat(db): add SQLite connection, migrations, and workspace-scoped schema` |
| T-102 | Account CRUD and balance calculation | `[x]` | `feat(accounts): complete account CRUD and balance calculation` |
| T-103 | Tag CRUD | `[x]` | `feat(tags): add tag CRUD endpoints` |
| T-104 | Transaction CRUD with tags, signed amounts, and filtering | `[x]` | `feat(transactions): add transaction CRUD with tags and filtering` |
| T-105 | Budget CRUD and tag-based budget summaries | `[x]` | `feat(budgets): add budget CRUD and tag-based summaries` |
| T-106 | Spending summary and recommendation endpoints | `[x]` | `feat(spending): add spending summaries and recommendation endpoints` |
| T-107 | OpenAPI contract and request/response validation | `[x]` | `docs: add OpenAPI 3.0.1 specification` |
| T-108 | Authentication (tailnet identity and bearer tokens), workspace mapping, and rate limiting | `[x]` | `feat(auth): add bearer token auth and rate limiting` |

## Milestone 2 — Web frontend

| ID | Task | Status | Commit |
| --- | --- | --- | --- |
| T-201 | Frontend scaffold, design tokens, and responsive layout | `[x]` | `feat(web): add app shell and responsive navigation` |
| T-202 | Dashboard with balance, spending, and budget widgets | `[x]` | `feat(web): add dashboard widgets and aggregate API` |
| T-203 | Transaction list, filters, and create/edit flow | `[x]` | `feat(web): add transaction list and create/edit flow` |
| T-204 | Budget and account management screens | `[x]` | `feat(web): add budget and account management screens` |
| T-205 | Spending charts and what-if simulation | `[x]` | `feat(web): add spending charts and what-if simulation` |
| T-206 | PWA polish, accessibility pass, and performance budget check | `[x]` | `feat(web): add PWA manifest and service worker` |
| T-206a | Accessibility pass and performance budget check (follow-up) | `[x]` | `feat(web): complete accessibility and performance budget check` |
| T-207 | Workspace switcher and personal/shared workspace UI, hidden until more than one workspace exists | `[!]` | Blocked: no workspace API endpoints yet |
| T-208 | Inline tag picker in the transaction form with create, rename, and predefined color editing | `[x]` | `feat(web): add inline tag management to transactions` |
| T-209 | Fix pre-existing frontend lint and formatting failures | `[x]` | `fix(web): resolve pre-existing lint and formatting failures` |

## Milestone 3 — AI, MCP, and WhatsApp

| ID | Task | Status | Commit |
| --- | --- | --- | --- |
| T-301 | MCP server with transaction, query, budget, and receipt tools | `[x]` | `feat(mcp): add MCP server with transaction, query, budget, and receipt tools` |
| T-302 | AI provider abstraction and chat-to-transaction draft flow | `[x]` | `feat(ai): add provider interface, OpenAI adapter, config wiring, and 503 gating` |
| T-302a | Provider interface, OpenAI-compatible adapter, config wiring, and 503 gating | `[x]` | `feat(ai): add provider interface, OpenAI adapter, config wiring, and 503 gating` |
| T-302b | parse-transaction endpoint → draft creation, validation, audit log | `[x]` | `feat(ai): add parse-transaction endpoint creating a draft` |
| T-302c | Prompt fixtures and golden tests for parsing accuracy | `[x]` | `test(ai): add prompt fixture and golden test for parsing` |
| T-303 | WhatsApp webhook with signature verification, allowlist, and idempotency | `[!]` | Deferred by D5 |
| T-304 | Receipt upload and vision-based OCR into a confirmable draft | `[x]` | `feat(receipts): add receipt upload with tesseract + vision LLM OCR` |
| T-305 | Audit trail for AI-assisted writes | `[x]` | `feat(audit): instrument draft confirm/reject, receipt upload, and failed auth` |
| T-305a | Audit service + Log helper + unit test | `[x]` | `feat(audit): instrument draft confirm/reject, receipt upload, and failed auth` |
| T-305b | Instrument draft confirm/reject, receipt upload, failed auth | `[x]` | `feat(audit): instrument draft confirm/reject, receipt upload, and failed auth` |

## Milestone 4 — Deployment and operations

| ID | Task | Status | Commit |
| --- | --- | --- | --- |
| T-401 | Multi-stage container image with embedded frontend | `[x]` | `build: add multi-stage Containerfile with embedded frontend` |
| T-402 | Quadlet pod and container units with rootless config, data mounts, health checks, and hardening | `[x]` | `feat(deploy): add Quadlet pod and container units` |
| T-403 | Tailscale sidecar via Quadlet and tailnet-only Serve exposure | `[x]` | `feat(deploy): add Quadlet pod and container units` |
| T-404 | Backup and restore procedure for SQLite | `[ ]` | — |
| T-405 | Operational runbook, health checks, journald logging, and upgrade/rollback steps | `[ ]` | — |
| T-406 | Install docs for symlinking units into `~/.config/containers/systemd`, lingering, start, and uninstall | `[ ]` | — |

## Decisions (D1–D13)

These are recorded as D1–D13 in `docs/TECH_SPEC.md` §16.

**Resolved:**

- [x] D1 — Approved: Go + SQLite + SvelteKit + rootless Quadlet.
- [x] D2 — Approved: Option B (multi-workspace). Personal workspace by default.
- [x] D3 — Approved: tailnet identity + hashed API tokens.
- [x] D4 — Approved: pluggable OpenAI-compatible AI adapter.
- [x] D5 — Deferred for v1; keep the narrow Funnel webhook design for later.
- [x] D6 — Approved: Rocky Linux 10.2 (x86_64), Podman 5.8.2, SELinux Enforcing, user `kegiatan` on miniPC, always on, no external drive.
- [x] D7 — Deferred; local backup now, restic to Cloudflare R2/B2 later.

**Later (D8–D13):**

- [ ] D8 — Currency defaults.
- [ ] D9 — Offline behavior.
- [x] D10 — MIT License.
- [x] D11 — GitHub Actions for `menyebarluaskan/budget-agent`; `make check` is the local and CI gate.
- [ ] D12 — Recommendation engine.
- [ ] D13 — Receipt privacy.
- [ ] D14 — WhatsApp number-to-workspace mapping (deferred with D5).

## Changelog

| Date | Change |
| --- | --- |
| 2026-11-02 | Initial task list; specification phase opened. |
| 2026-11-02 | Deployment revised to rootless Podman Quadlet; decisions prioritized. |
| 2026-11-02 | D1/D3/D4 approved; D5/D7 deferred; D2 options expanded; D6 moved to miniPC. |
| 2026-11-02 | D2 approved as multi-workspace; miniPC details confirmed; tailnet hostname proposed. |
| 2026-11-02 | T-005 approved; T-007 scaffolded the Go API, SvelteKit UI, and Makefile quality gates. |
| 2026-11-02 | D11 resolved and T-008 added the GitHub Actions CI quality gate. |
| 2026-11-02 | D10 resolved as MIT; M0 foundation complete. |
| 2026-11-03 | M1 backend core complete; T-201 added the app shell, design tokens, and responsive navigation. |
| 2026-11-03 | T-202 dashboard and aggregate API completed. |
| 2026-11-04 | T-208 added inline tag creation, rename, and color editing to the transaction form. |
| 2026-11-04 | T-205 added tag spending and monthly trend charts, budget burn-down, and a local what-if simulator. |
