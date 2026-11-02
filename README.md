# Integrated Personal Budget

A self-hosted budgeting tool for one person or a household. It runs as a single
containerized service with its own pod and a Tailscale sidecar, exposes an HTTP
API, and ships a mobile-friendly web UI. AI agents and a WhatsApp bot use the
same API through MCP.

> **Status: design phase.** The technical specification is awaiting review. No
> application code exists yet. See [`docs/TECH_SPEC.md`](docs/TECH_SPEC.md) and
> [`TASKS.md`](TASKS.md).

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

## Proposed architecture

```text
Browser / MCP client / WhatsApp
              │
        Tailscale tailnet
              │
   ┌──────────▼──────────┐
   │ Kubernetes pod      │
   │  Tailscale sidecar  │
   │       ↓             │
   │  App container      │
   │  API + UI + MCP     │
   │       ↓             │
   │  PVC: SQLite + files│
   └─────────────────────┘
```

- **Backend:** Go, standard-library HTTP, OpenAPI 3.1, SQLite.
- **Frontend:** SvelteKit static build, embedded into the backend binary.
- **Deployment:** one non-root container, a Tailscale sidecar, and a persistent
  volume. No public ingress.
- **AI/chat:** provider-agnostic AI adapter, MCP over Streamable HTTP, WhatsApp
  Cloud API.

The full rationale, data model, API surface, security rules, and open decisions
are in [`docs/TECH_SPEC.md`](docs/TECH_SPEC.md). **The stack is a proposal until
the maintainer approves it.**

## Documentation

| Document | Contents |
| --- | --- |
| [`docs/TECH_SPEC.md`](docs/TECH_SPEC.md) | Architecture, stack, schema, API, AI/MCP, deployment, testing |
| [`AGENTS.md`](AGENTS.md) | Working rules and conventions for agents and contributors |
| [`TASKS.md`](TASKS.md) | Progress tracker and current phase |

## Project status

| Milestone | Status |
| --- | --- |
| M0 — Specification and foundations | In progress: spec under review |
| M1 — Backend core | Not started |
| M2 — Web frontend | Not started |
| M3 — AI, MCP, WhatsApp | Not started |
| M4 — Deployment and operations | Not started |

## Development

There is no runnable code yet. Once the specification is approved, this section
will document prerequisites, local development, tests, and the container build.
Planned entry points will go through a `Makefile`.

## Contributing

1. Read [`AGENTS.md`](AGENTS.md) and the relevant section of the technical spec.
2. Pick a task from [`TASKS.md`](TASKS.md).
3. Keep one logical change per commit and use
   [Conventional Commits](https://www.conventionalcommits.org/) (for example,
   `feat(api): add transaction filters`).
4. Update the docs and task tracker in the same change.

## License

Not decided yet. See the open decisions in the technical spec.
