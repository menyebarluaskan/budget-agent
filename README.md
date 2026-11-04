# Integrated Personal Budget

A self-hosted budgeting tool for one person or a household. It runs as a single
containerized service with its own pod and a Tailscale sidecar, exposes an HTTP
API, and ships a mobile-friendly web UI. AI agents and a WhatsApp bot use the
same API through MCP.

> **Status: implementation phase.** The backend core, the budgeting dashboard,
> and the transaction flow are complete. Budget and account management screens
> plus the AI/MCP milestones are next. See [`docs/TECH_SPEC.md`](docs/TECH_SPEC.md)
> and [`TASKS.md`](TASKS.md).

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
- Transaction list with URL-reflected filters and cursor pagination.
- Create, edit, and delete transactions with signed amounts, accounts, tags, and
  dates. Tags can be created, renamed, and recolored inline while editing a
  transaction.
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
| M1 — Backend core | Complete: workspace schema, accounts, tags, transactions, budgets, summaries, auth, OpenAPI |
| M2 — Web frontend | In progress: app shell, dashboard with charts, transaction flow, budget/account screens, what-if simulation |
| M3 — AI, MCP, WhatsApp | Not started |
| M4 — Deployment and operations | Not started |

## Testing

### Automated

```sh
make check          # Go fmt/vet + frontend lint/typecheck + tests + build
make test           # Go + frontend unit tests only
make web-check      # SvelteKit type check
```

### Manual: web UI

1. `make dev` (API on :8080, Vite on :5173).
2. Open http://127.0.0.1:5173. In dev mode (`AUTH_MODE=dev`) there is no login.
3. Add an account, create tags, add transactions, create a budget, view the dashboard and charts.

### Manual: AI parse-transaction

Requires `AI_ENABLED=true` and a configured provider (hosted or local Ollama/LM Studio).

```sh
AI_ENABLED=true AI_BASE_URL=http://127.0.0.1:11434/v1 AI_MODEL=llama3.2 make dev-api
curl -s -X POST http://127.0.0.1:8080/api/v1/ai/parse-transaction \
  -H 'Content-Type: application/json' \
  -d '{"text":"coffee 150 at Starbucks"}'
```

Returns a `draft_id` plus parsed fields. Nothing is persisted until you confirm the draft.

### Manual: MCP endpoint

```sh
curl -s -X POST http://127.0.0.1:8080/mcp \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}'
```

Then call tools (`list_accounts`, `query_transactions`, `create_transaction_draft`, `confirm_transaction_draft`, etc.).

### Manual: receipt upload + OCR

```sh
curl -s -X POST http://127.0.0.1:8080/api/v1/receipts \
  -H 'Content-Type: image/jpeg' --data-binary @receipt.jpg
```

Returns `receipt_id`, `ocr_status`, `ocr_text` (Tesseract, if installed), and `ocr_json` (vision LLM, if AI enabled). Poll `GET /api/v1/receipts/{id}`.

### Manual: backup

```sh
DATABASE_PATH=$PWD/.local/data/budget.db bin/budget-agent backup --dest /tmp/bk
```

### Deployment smoke

```sh
make image
podman run --rm -p 8080:8080 \
  -e DATABASE_PATH=/data/budget.db -e RECEIPTS_DIR=/data/receipts \
  -e BASE_URL=http://localhost:8080 -e AUTH_MODE=dev \
  -e TOKEN_PEPPER=dev -v /tmp/scratchpad/data:/data:Z \
  budget-agent:latest
curl -s http://127.0.0.1:8080/healthz
```

Quadlet generator dry-run: see deploy/quadlet/README.md.

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
