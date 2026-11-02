# Integrated Personal Budget

A self-hosted budgeting tool for one person or a household. It runs as a single
containerized service with its own pod and a Tailscale sidecar, exposes an HTTP
API, and ships a mobile-friendly web UI. AI agents and a WhatsApp bot use the
same API through MCP.

> **Status: foundation phase.** The technical specification is approved and the
> repository is scaffolded. Health endpoints, the SvelteKit shell, and the local
> quality gates are in place; feature milestones are next. See
> [`docs/TECH_SPEC.md`](docs/TECH_SPEC.md) and [`TASKS.md`](TASKS.md).

## Goals

1. **Optimized backend.** Low memory usage, a small image, and a small-footprint
   database.
2. **Agent access.** AI agents and MCP clients can read data and propose
   transactions via the API, with a human confirmation step. WhatsApp provides a
   chat interface.
3. **Mobile-first web UI.** Fast transaction entry, filtering, budgets, and
   charts from a phone.

## Core Features (planned)

### Budgeting

- Add transactions with multiple tags and a simple `+`/`-` expense indicator.
- Separate workspaces for household members, with optional shared workspaces
  later.
- Budgets grouped by multiple transaction tags.
- Wallets, savings, and credit accounts, including transfers between them.
- Filter by account, tag, budget, and amount.
- Budget summaries and spending recommendations.

### AI agents

- Receipt OCR with a summary and suggested tags.
- Add transactions from a chat message.
- MCP tools for querying and drafting transactions.

### Web frontend

- Budgeting dashboard.
- Spending graphs and what-if simulations.
- PWA-friendly, mobile-first layout.

## Architecture

```text
Browser / MCP client / WhatsApp
              │
        Tailscale tailnet
              │
   ┌──────────▼───────────────────┐
   │ Rootless Podman pod (Quadlet)│
   │  Tailscale sidecar           │
   │       ↓                      │
   │  App container               │
   │  API + UI + MCP              │
   │       ↓                      │
   │  Host dir: SQLite + receipts │
   └──────────────────────────────┘
```

- **Backend:** Go, standard-library HTTP, OpenAPI 3.1, SQLite.
- **Frontend:** SvelteKit static build, embedded into the backend binary.
- **Deployment:** one rootless Podman pod managed by systemd Quadlet, with a
  Tailscale sidecar. Units are symlinked into
  `~/.config/containers/systemd/`; no host ports, no public ingress, and no
  Kubernetes.
- **AI/chat:** provider-agnostic AI adapter, MCP over Streamable HTTP, WhatsApp
  Cloud API (optional and deferred until a public webhook is acceptable).

The full rationale, data model, API surface, security rules, and decisions are in
[`docs/TECH_SPEC.md`](docs/TECH_SPEC.md).

## Documentation

| Document | Contents |
| --- | --- |
| [`docs/TECH_SPEC.md`](docs/TECH_SPEC.md) | Architecture, stack, schema, API, AI/MCP, deployment, testing |
| [`AGENTS.md`](AGENTS.md) | Working rules and conventions for agents and contributors |
| [`TASKS.md`](TASKS.md) | Progress tracker and current phase |

## Project status

| Milestone | Status |
| --- | --- |
| M0 — Specification and foundations | Complete: spec approved, scaffolded, CI green, MIT licensed |
| M1 — Backend core | Not started |
| M2 — Web frontend | Not started |
| M3 — AI, MCP, WhatsApp | Not started |
| M4 — Deployment and operations | Not started |

## Development

### Prerequisites

- Go 1.26.4 or newer.
- Node.js 22.12 or newer.
- pnpm 11 or newer.
- GNU Make.

### First run

```bash
make dev
```

`make dev` installs frontend dependencies if needed, then starts the API on
`http://127.0.0.1:8080` and the SvelteKit dev server on
`http://127.0.0.1:5173`. Local data goes under `.local/`, which is ignored by
Git.

To run them separately:

```bash
make dev-api
make dev-web
```

### Quality gates

```bash
make check
```

This runs Go formatting and vetting, frontend format/lint checks, the SvelteKit
type check, both unit-test suites, and the production frontend build. GitHub
Actions runs the same `make check` gate on pushes to `main` and pull requests.

Use `make build` to produce `bin/budget-agent`. The frontend build currently
lives in `web/build`; embedding it into the Go binary lands with the deployment
milestone.

## Contributing

1. Read [`AGENTS.md`](AGENTS.md) and the relevant section of the technical spec.
2. Pick a task from [`TASKS.md`](TASKS.md).
3. Keep one logical change per commit and use
   [Conventional Commits](https://www.conventionalcommits.org/) (for example,
   `feat(api): add transaction filters`).
4. Update the docs and task tracker in the same change.

## License

MIT. See [`LICENSE`](LICENSE).
