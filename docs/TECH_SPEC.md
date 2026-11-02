# Technical Specification — Integrated Personal Budget

**Status:** Draft for maintainer review. **No implementation may start until this
document is approved.** Approval is tracked as `T-005` in `TASKS.md`.

**Date:** 2026-11-02

**Related:** [README](../README.md) · [AGENTS](../AGENTS.md) · [TASKS](../TASKS.md)

---

## 1. Goals

Build a self-hosted, low-resource personal budgeting system that:

1. Runs as a single Kubernetes pod with a Tailscale sidecar and no public
   ingress.
2. Exposes a stable HTTP API for every operation the UI and agents can perform.
3. Tracks accounts, signed transactions, tags, and tag-based budgets.
4. Provides a mobile-first web UI with a dashboard, filters, charts, and simple
   spending simulations.
5. Lets AI agents read data and propose transactions through MCP, with an
   explicit human confirmation step before writes.
6. Accepts transactions from WhatsApp text messages and receipt images.
7. Stays small: low idle memory, a small image, and a single-file database.

### Non-goals for v1

- Multi-tenant SaaS, team workspaces, or public sign-up.
- Bank or card aggregator integrations.
- Full double-entry accounting, tax reporting, or invoicing.
- Real-time collaboration or multi-writer scaling.
- Native mobile applications.
- Investment portfolio tracking or currency trading.

## 2. Assumptions and constraints

| Constraint | Assumption |
| --- | --- |
| Users | One person or one household. A small number of named users is acceptable. |
| Deployment | Single Kubernetes cluster, one application replica. |
| Network | Reachable only over a Tailscale tailnet. No public Ingress. |
| Data volume | Under ~100k transactions and ~10 GB of receipt images over several years. |
| Currency | One currency per account; a single reporting currency for summaries. |
| Availability | Personal-use availability. Brief restarts during upgrades are acceptable. |
| Backups | The persistent volume is backed up off-cluster on a schedule. |
| Locale | Default timezone and currency are configurable; dates stored in UTC. |

## 3. Architecture

```text
                       Tailnet (WireGuard)
  Browser ───────────────┐
  MCP client ────────────┤
  WhatsApp Cloud API ──┐ │
                       │ │
              ┌────────▼─▼──────────────────────────────┐
              │ Kubernetes Pod                          │
              │                                         │
              │  ┌───────────────────────┐              │
              │  │ Tailscale sidecar     │              │
              │  │ userspace + Serve     │              │
              │  │ :443 → 127.0.0.1:8080 │              │
              │  └──────────┬────────────┘              │
              │             │ localhost                 │
              │  ┌──────────▼────────────────────────┐  │
              │  │ Application container             │  │
              │  │ - REST API /api/v1                │  │
              │  │ - Web UI (embedded static build)  │  │
              │  │ - MCP /mcp                        │  │
              │  │ - WhatsApp webhook                │  │
              │  │ - receipt pipeline                │  │
              │  └──────────┬────────────────────────┘  │
              │             │                           │
              │  ┌──────────▼────────────────────────┐  │
              │  │ PVC /data                         │  │
              │  │ budget.db (SQLite, WAL)           │  │
              │  │ receipts/                         │  │
              │  │ backups/                          │  │
              │  └───────────────────────────────────┘  │
              └─────────────────────────────────────────┘
```

Key decisions:

- **One binary serves everything.** The UI is compiled to static assets and
  embedded into the Go binary with `go:embed`. No separate web server.
- **Loopback binding.** The app listens on `127.0.0.1:8080`; the Tailscale
  sidecar terminates TLS and proxies to it. The app is not exposed on the pod IP
  or via a public Service.
- **SQLite is the system of record.** WAL mode, one writer, foreign keys on,
  busy timeout configured. A PVC holds the database and receipt files.
- **Background work is in-process.** OCR and AI calls run in a small worker with
  a persisted job table so restarts do not lose work. No external queue.
- **The API is the product boundary.** UI, MCP, WhatsApp, and future clients use
  the same domain services. MCP and WhatsApp are adapters, not separate logic.

## 4. Proposed stack

| Layer | Choice | Why |
| --- | --- | --- |
| Backend language | **Go**, latest stable (minimum 1.24) | Small static binary, low RSS, strong concurrency, easy cross-compilation. |
| HTTP router | Standard library `net/http` (`ServeMux` patterns) | No framework needed; fewer dependencies and upgrades. |
| API contract | **OpenAPI 3.1** + `oapi-codegen` | Types and server interface generated from the contract; prevents drift. |
| Database | **SQLite** via `modernc.org/sqlite` (pure Go) | Single file, CGO-free builds, tiny operational footprint. |
| Queries | `database/sql` + **sqlc** | Compile-time checked SQL without an ORM. |
| Migrations | **goose** (library mode), embedded SQL files | Simple, ordered, works with SQLite. |
| Frontend | **SvelteKit 2 / Svelte 5**, TypeScript, static adapter | Small output, excellent mobile ergonomics, no Node runtime in production. |
| Styling/UI | **Tailwind CSS 4** + `shadcn-svelte` (bits-ui) | Consistent accessible primitives without a heavy component framework. |
| Charts | **LayerChart** (Svelte-native) or small hand-rolled SVG | Avoids a large charting runtime; keep the option open after a spike. |
| PWA | `vite-plugin-pwa` | Installable UI, cached shell, read-only offline behavior. |
| MCP | Official **Go SDK** (`modelcontextprotocol/go-sdk`) | Same process and auth model as the API. |
| AI provider | Provider-agnostic interface; OpenAI-compatible HTTP first | Works with hosted APIs and local models (Ollama, LM Studio, vLLM). |
| WhatsApp | **Meta WhatsApp Cloud API** | Official webhook and messaging API; no unofficial bridge. |
| Packaging | Multi-stage Docker build to `gcr.io/distroless/static` | Small, non-root, no shell. |
| Orchestration | Kubernetes manifests managed with **Kustomize** | One small app; Helm is unnecessary overhead. |
| Local dev | Docker Compose for app + optional fake WhatsApp webhook | Reproducible without a cluster. |
| CI | Platform-agnostic steps; GitHub Actions if hosted on GitHub | Keep build/test/lint identical locally and in CI. |

### Versions and dependency policy

- Pin toolchain versions in `go.mod`, `package.json` (with a committed lockfile),
  and the Dockerfile.
- Renovate or Dependabot may update dependencies; updates must pass CI.
- Avoid CGO. The final image should not need a libc.
- Every new direct dependency needs a one-line justification in the pull request.

### Rejected alternatives

| Alternative | Why not now |
| --- | --- |
| TypeScript/Node backend | Higher idle memory and a larger runtime image; shared language with the UI is not worth the footprint. |
| Rust backend | Excellent runtime profile, but slower iteration and a smaller contributor pool for this project. |
| Python/FastAPI | Simple to prototype, but packaging, memory, and image size are worse for a long-lived small service. |
| PostgreSQL | Operationally heavier: another pod, PVC, backups, and credentials. SQLite fits the single-user, single-writer constraint. |
| React/Next.js | Larger bundles and a heavier build/runtime story. SvelteKit's static output is a better fit. |
| htmx + server templates | Very small, but charts, PWA behavior, and interactive simulations would need more custom JavaScript. |
| GraphQL/tRPC | Adds concepts and tooling without a clear benefit for a small, well-defined REST surface. |
| Unofficial WhatsApp bridge | Account-ban risk and no supported webhook contract. The official Cloud API is preferred. |

## 5. Data model

All timestamps are UTC ISO-8601 text in SQLite. Money is stored as a signed
integer in the currency's minor unit (for example, cents).

### 5.1 Tables

**`accounts`**

| Column | Type | Notes |
| --- | --- | --- |
| `id` | TEXT (UUID) | Primary key |
| `name` | TEXT | Unique, non-empty |
| `type` | TEXT | `wallet`, `savings`, or `credit` |
| `currency` | TEXT | ISO 4217, three uppercase letters |
| `opening_balance_minor` | INTEGER | Signed; existing credit debt is negative |
| `archived` | INTEGER | 0/1 |
| `created_at`, `updated_at` | TEXT | UTC |

**`transactions`**

| Column | Type | Notes |
| --- | --- | --- |
| `id` | TEXT (UUID) | Primary key |
| `account_id` | TEXT | FK → accounts |
| `amount_minor` | INTEGER | Signed: positive inflow, negative outflow |
| `occurred_at` | TEXT | UTC timestamp |
| `description` | TEXT | Short merchant or purpose |
| `notes` | TEXT | Optional |
| `source` | TEXT | `manual`, `whatsapp`, `mcp`, `receipt`, `import` |
| `transfer_id` | TEXT | Nullable; the two legs of a transfer share it |
| `external_id` | TEXT | Nullable; idempotency key from an external system |
| `created_at`, `updated_at` | TEXT | UTC |

**`tags`** — `id`, `name` (unique, case-insensitive), `color`, `created_at`.

**`transaction_tags`** — `(transaction_id, tag_id)`, composite primary key.

**`budgets`**

| Column | Type | Notes |
| --- | --- | --- |
| `id` | TEXT | Primary key |
| `name` | TEXT | Non-empty |
| `amount_minor` | INTEGER | Positive limit per period |
| `period` | TEXT | `weekly` or `monthly` |
| `start_date` | TEXT | First period start (`YYYY-MM-DD`) |
| `end_date` | TEXT | Nullable; budget stops after this date |
| `rollover` | INTEGER | 0/1; unused amount carries forward |
| `account_id` | TEXT | Nullable; restrict to one account |
| `created_at`, `updated_at` | TEXT | UTC |

**`budget_tags`** — `(budget_id, tag_id)`. A transaction counts toward a budget if
it has **at least one** of the budget's tags (OR semantics).

**`receipts`** — `id`, `transaction_id` (nullable), `stored_path`, `mime_type`,
`byte_size`, `sha256`, `ocr_status` (`pending`, `done`, `failed`), `ocr_text`,
`ocr_json`, `created_at`.

**`drafts`** — pending AI/MCP/WhatsApp proposals: `id`, `source`, `payload_json`,
`status` (`pending`, `confirmed`, `rejected`, `expired`), `created_at`,
`expires_at`, `confirmed_at`.

**`jobs`** — `id`, `kind` (`ocr`, `ai_parse`), `payload_json`, `status`,
`attempts`, `last_error`, `run_after`, `created_at`, `updated_at`.

**`api_tokens`** — `id`, `name`, `token_hash` (Argon2id), `scopes_json`,
`last_used_at`, `created_at`, `revoked_at`.

**`whatsapp_messages`** — `message_id` (unique), `from_number`, `body`,
`received_at`, `status`, `draft_id`. Used for idempotency and audit.

**`audit_log`** — `id`, `actor`, `action`, `entity_type`, `entity_id`,
`metadata_json`, `created_at`. AI-assisted writes and auth changes are recorded.

**`schema_migrations`** — managed by goose.

### 5.2 Amount and account semantics

- `balance = opening_balance_minor + SUM(amount_minor)` for non-archived
  transactions in that account.
- A purchase is negative. Income is positive.
- For a credit account, a negative balance means money owed. The UI displays the
  debt as an absolute value with a "due" label; the sign remains consistent in
  storage.
- Transfers create two transaction rows with the same `transfer_id`: negative on
  the source account and positive on the destination. Transfer legs are excluded
  from income, expense, budget, and recommendation calculations.
- Refunds are positive rows and offset spending in summaries.
- A credit-card payment is a transfer from a wallet/savings account to the credit
  account. The positive leg reduces the credit balance (debt).

### 5.3 Budget calculation

For a budget and a period `[start, end)`:

```text
spent = max(0, -SUM(amount_minor))
```

over non-transfer transactions that:

- occurred in the period,
- optionally match the budget's `account_id`,
- have at least one tag in `budget_tags`.

`remaining = amount_minor - spent` (plus rollover if enabled). Transfers are
excluded. The calculation is implemented once in the domain layer and reused by
the API, UI, MCP, and WhatsApp flows.

### 5.4 Indexes and integrity

- Foreign keys enabled per connection.
- Indexes on `transactions(occurred_at)`, `transactions(account_id, occurred_at)`,
  `transaction_tags(tag_id)`, and unique indexes on account name, tag name, and
  `whatsapp_messages.message_id`.
- `PRAGMA journal_mode=WAL`, `synchronous=NORMAL`, `busy_timeout=5000`.
- Writes run in short transactions. The API is the only writer.

## 6. HTTP API

Base path: `/api/v1`. JSON only; request and response bodies use `snake_case`.
Errors use RFC 9457 `application/problem+json`.

### 6.1 Resources

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/healthz` | Liveness, no auth |
| `GET` | `/readyz` | Database readiness, no auth |
| `GET` | `/accounts` | List accounts with balances |
| `POST` | `/accounts` | Create account |
| `GET` | `/accounts/{id}` | Account detail |
| `PATCH` | `/accounts/{id}` | Rename, archive, or adjust opening balance |
| `GET` | `/tags` | List tags |
| `POST` | `/tags` | Create tag |
| `PATCH` | `/tags/{id}` | Rename or recolor |
| `DELETE` | `/tags/{id}` | Delete unused tag |
| `GET` | `/transactions` | Filter and cursor-paginate |
| `POST` | `/transactions` | Create transaction |
| `GET` | `/transactions/{id}` | Detail with tags and receipt |
| `PATCH` | `/transactions/{id}` | Update |
| `DELETE` | `/transactions/{id}` | Delete |
| `POST` | `/transfers` | Create a two-leg transfer |
| `GET` | `/budgets` | List budgets |
| `POST` | `/budgets` | Create budget |
| `GET` | `/budgets/{id}` | Budget detail |
| `PATCH` | `/budgets/{id}` | Update |
| `DELETE` | `/budgets/{id}` | Delete |
| `GET` | `/budgets/{id}/summary` | Period progress |
| `GET` | `/summary` | Income, expense, net, by tag/account |
| `GET` | `/recommendations` | Rule-based spending guidance |
| `POST` | `/receipts` | Upload an image and start OCR |
| `GET` | `/receipts/{id}` | OCR status and draft |
| `GET` | `/drafts` | List pending drafts |
| `POST` | `/drafts/{id}/confirm` | Persist a draft |
| `POST` | `/drafts/{id}/reject` | Discard a draft |
| `GET` | `/tokens` | List API token metadata |
| `POST` | `/tokens` | Create a token (plaintext returned once) |
| `DELETE` | `/tokens/{id}` | Revoke a token |

Also: `POST /webhooks/whatsapp`, `GET /webhooks/whatsapp` (verification), and
`/mcp` (Streamable HTTP transport). MCP and webhooks are mounted outside
`/api/v1`.

### 6.2 Query conventions

- `GET /transactions` filters: `from`, `to` (inclusive dates), `account_id`,
  repeated `tag`, `min_amount_minor`, `max_amount_minor`, `q`, `source`,
  `include_transfers`.
- Pagination: `limit` (default 50, max 200) plus an opaque `cursor`; responses
  include `next_cursor`.
- Sorting: `sort=occurred_at|-occurred_at|amount_minor|-amount_minor`.
- Write requests accept `Idempotency-Key`; the response is cached per key for 24
  hours.
- Validation failures return `400` with field-level details. Missing resources
  return `404`. Conflicts return `409`. Rate limits return `429` with
  `Retry-After`.

### 6.3 Auth

- **Web UI:** Tailscale Serve injects identity headers
  (`Tailscale-User-Login`). The app trusts them only because it binds to loopback
  and is reachable solely through the sidecar. Requests without identity are
  rejected unless a fallback auth mode is enabled.
- **API/MCP clients:** `Authorization: Bearer <token>`. Tokens are shown once,
  stored as Argon2id hashes, and carry scopes (`read`, `write`, `admin`).
- **WhatsApp webhook:** Meta signature verification plus a phone-number
  allowlist. The webhook endpoint is the only route exempt from user auth.
- **Local development:** an explicit `AUTH_MODE=dev` disables user auth and must
  never be enabled in the deployed manifest.
- **CSRF:** the UI uses a same-site session cookie only if a fallback login mode
  is added; otherwise identity headers are used per request. State-changing API
  calls require a custom header, which also blocks simple CSRF.

Open decision: whether to add a passphrase fallback for browsers outside the
tailnet. The recommended default is tailnet identity only.

## 7. AI, MCP, and WhatsApp

### 7.1 AI abstraction

```go
type Provider interface {
    ParseTransaction(ctx context.Context, input ParseInput) (TransactionDraft, error)
    ExtractReceipt(ctx context.Context, image []byte, mime string) (ReceiptDraft, error)
}
```

- First implementation: OpenAI-compatible `/v1/chat/completions` endpoint with
  structured JSON output. This also covers local models.
- Optional Anthropic adapter later; no provider is hard-coded into handlers.
- Configuration: base URL, API key, text model, vision model, timeout, and a
  feature flag. Without configuration, AI endpoints return `503` and the rest of
  the app works normally.
- Prompts and schemas are versioned and tested with fixtures. Output is validated
  before it becomes a draft.
- **No auto-write.** All AI and MCP writes produce a `drafts` row. A human
  confirms it through the UI, a WhatsApp reply, or a second MCP call.

### 7.2 MCP tools

| Tool | Behavior |
| --- | --- |
| `list_accounts` | Read accounts and balances |
| `list_tags` | Read tags |
| `query_transactions` | Read with bounded filters |
| `budget_summary` | Budget progress for a period |
| `spending_summary` | Totals by tag/account |
| `create_transaction_draft` | Validate and store a pending draft |
| `confirm_transaction_draft` | Persist the draft after confirmation |
| `reject_transaction_draft` | Discard a draft |
| `extract_receipt` | Store an image, run OCR, return a receipt draft |

- Transport: Streamable HTTP at `/mcp`, bearer auth, same scopes as the API.
- Tool inputs and outputs have explicit JSON schemas. No arbitrary SQL or shell
  access.
- Rate-limit expensive tools (OCR, AI parsing).

### 7.3 WhatsApp flow

1. Meta calls `GET /webhooks/whatsapp` for verification; respond with the
   challenge only when the verify token matches.
2. Meta calls `POST /webhooks/whatsapp`; verify `X-Hub-Signature-256` against the
   raw body **before** parsing.
3. Ignore or politely reject numbers outside `WHATSAPP_ALLOWED_NUMBERS`.
4. Deduplicate by `message_id`; retries from Meta must be harmless.
5. Text message: parse into a transaction draft, reply with a summary and ask for
   confirmation. Image message: download, store, run OCR, reply with a draft.
6. Affirmative replies (`yes`, `confirm`, `ok`) confirm the draft; `no`/`cancel`
   rejects it. Ambiguous replies ask again. Drafts expire after 30 minutes.
7. Reply through the Cloud API. Failures are logged with the message ID and
   retried a bounded number of times.

### 7.4 Receipt OCR

- Accept JPEG, PNG, WebP, and HEIC up to 10 MiB. Verify magic bytes, not just the
  declared content type.
- Store files under `/data/receipts/<sha256>` with restrictive permissions.
- Extract merchant, date, total, tax, line items, and suggested tags. Every field
  is nullable; the user edits before confirming.
- Use the vision model when configured. A local OCR fallback (Tesseract) is
  optional and out of scope for v1.
- Receipt images are never sent to a third party unless an AI provider is
  configured and the user confirms the upload.

## 8. Web frontend

### 8.1 Routes

| Route | Content |
| --- | --- |
| `/` | Dashboard: balance, month spend, budget progress, recent transactions |
| `/transactions` | Filterable, cursor-paginated list |
| `/transactions/new` | Quick-add with amount, sign, account, tags, date |
| `/transactions/[id]` | Details, edit, receipt preview |
| `/budgets` | Budget list and monthly progress |
| `/budgets/[id]` | Detail, tags, trend, simulation |
| `/accounts` | Accounts and balances, transfer action |
| `/settings` | Tokens, AI/WhatsApp status, preferences |
| `/login` | Only if a fallback auth mode is added |

### 8.2 UX requirements

- Mobile-first: bottom navigation on small screens, sidebar on wide screens.
- Quick add is reachable in one tap and supports a leading `+`/`-` amount.
- Filters are reflected in the URL so views are linkable and reload-safe.
- Charts: spending by tag, monthly trend, budget burn-down.
- Simulation: adjust a budget or hypothetical spend locally and show the effect
  without writing data.
- Empty, loading, and error states are designed, not afterthoughts.
- Accessibility: keyboard navigation, visible focus, labeled controls, and
  color-independent status cues.

### 8.3 Frontend constraints

- Static build output embedded into the Go binary. No server-side rendering.
- TypeScript strict mode.
- No client-side secret storage. Auth relies on the tailnet identity or a
  server-issued secure cookie.
- Initial JS budget: under 200 KiB gzip for the dashboard route.
- PWA caches the app shell and last-viewed data read-only. Offline writes are out
  of scope for v1.

## 9. Configuration

| Variable | Required | Purpose |
| --- | --- | --- |
| `APP_ADDR` | No | Listen address, default `127.0.0.1:8080` |
| `DATABASE_PATH` | Yes | SQLite file, for example `/data/budget.db` |
| `RECEIPTS_DIR` | Yes | Receipt storage, for example `/data/receipts` |
| `BASE_URL` | Yes | Public tailnet URL, used in links |
| `AUTH_MODE` | Yes | `tailnet` (default), `dev` |
| `TOKEN_PEPPER` | Yes | Server-side pepper for token hashing |
| `AI_ENABLED` | No | Feature flag, default `false` |
| `AI_BASE_URL` | No | OpenAI-compatible endpoint |
| `AI_API_KEY` | No | Provider key; Kubernetes Secret |
| `AI_MODEL` | No | Text model |
| `AI_VISION_MODEL` | No | Vision model |
| `WHATSAPP_ENABLED` | No | Feature flag, default `false` |
| `WHATSAPP_VERIFY_TOKEN` | No | Webhook verification token |
| `WHATSAPP_APP_SECRET` | No | Signature verification |
| `WHATSAPP_ACCESS_TOKEN` | No | Cloud API token |
| `WHATSAPP_PHONE_NUMBER_ID` | No | Sender identity |
| `WHATSAPP_ALLOWED_NUMBERS` | No | Comma-separated E.164 allowlist |
| `DEFAULT_CURRENCY` | No | ISO 4217, default `IDR` |
| `TZ` | No | Display timezone, default `UTC` |
| `LOG_LEVEL` | No | `info` (default), `debug`, `warn`, `error` |

Secrets come from Kubernetes Secrets. A committed `.env.example` contains only
placeholders. Configuration is validated at startup; the process exits with a
clear error rather than running half-configured.

## 10. Deployment

### 10.1 Image

- Stage 1: build the SvelteKit static output with the pinned Node LTS image.
- Stage 2: `go build` with `-trimpath -ldflags="-s -w"` and `CGO_ENABLED=0`,
  embedding the static output.
- Stage 3: `gcr.io/distroless/static:nonroot`, copy the binary and CA
  certificates. Final image target: under 60 MB.
- Run as a non-root user, read-only root filesystem, writable `/data` only,
  `tmpfs` at `/tmp`, and no shell.

### 10.2 Kubernetes

- One `Deployment` with `replicas: 1` and `strategy: Recreate` (SQLite and the
  PVC cannot be shared safely).
- `PersistentVolumeClaim` mounted at `/data`.
- `Service` of type `ClusterIP` only; no Ingress and no LoadBalancer.
- Resource requests/limits: `32Mi/128Mi` memory and `50m/500m` CPU as a starting
  point, tuned after measuring. Hard memory limit plus Go `GOMEMLIMIT` set below
  it.
- Probes: `GET /healthz` and `GET /readyz` on the loopback address.
- `securityContext`: non-root, no privilege escalation, all capabilities
  dropped, `seccompProfile: RuntimeDefault`.
- `NetworkPolicy`: deny ingress except from the Tailscale sidecar where the
  cluster CNI supports it; all egress allowed only as needed for AI, WhatsApp,
  and DNS.

### 10.3 Tailscale sidecar

- Image: `tailscale/tailscale` pinned to a specific version.
- `TS_USERSPACE=true` avoids `NET_ADMIN` and a TUN device. A Serve config
  terminates TLS and proxies `https://<host>.<tailnet>.ts.net` to
  `http://127.0.0.1:8080`.
- State directory on the PVC (`/data/tailscale`) so the node identity survives
  restarts. `TS_AUTHKEY` comes from a Secret; prefer a tagged, reusable auth key
  with key expiry disabled per Tailscale guidance.
- The app container shares the pod network namespace, so loopback works across
  containers.
- If userspace Serve proves problematic, fall back to kernel networking with
  `NET_ADMIN`; record the change in this section before merging.
- Tailnet ACLs restrict who can reach the service. MagicDNS provides the stable
  hostname.

### 10.4 Backups

- Nightly job: `VACUUM INTO '/data/backups/budget-YYYYMMDD.db'`, then copy
  off-cluster with restic/rclone.
- Keep 7 daily, 4 weekly, and 6 monthly snapshots (configurable).
- Receipts are included in the same backup set.
- Restore procedure is documented and tested before the first real data is
  entered. `T-404` owns this.

### 10.5 Upgrades and migrations

- Migrations run at startup inside a transaction and are idempotent.
- The readiness probe fails until migrations complete.
- Rollback plan: restore the pre-upgrade backup. Destructive migrations are not
  allowed in a single release; use expand/contract steps.

## 11. Observability

- Structured JSON logs via `log/slog`: timestamp, level, request ID, route,
  status, duration, and error code.
- Never log full request bodies, receipt contents, tokens, phone numbers, or
  transaction descriptions at info level.
- Request ID is accepted from a trusted proxy header or generated; it is returned
  in the response.
- `/healthz` checks the process; `/readyz` checks database access and migration
  state.
- Optional `/metrics` in Prometheus text format, bound to loopback and reachable
  only through the tailnet.
- Audit log for token changes, AI/MCP writes, WhatsApp confirmations, and failed
  auth attempts.

## 12. Security and privacy

- Tailnet-only by default; no public endpoints except the WhatsApp webhook path
  if the operator chooses to expose it through a controlled tunnel. Prefer a
  Tailscale Funnel-free design and a dedicated webhook relay if needed.
- Verify webhook signatures on the raw body before JSON parsing.
- Hash tokens with Argon2id; use constant-time comparison.
- Enforce scopes per route and per MCP tool.
- Bound request bodies (1 MiB JSON, 10 MiB uploads), query ranges, page sizes,
  and AI timeouts.
- Parameterized SQL only; no dynamic identifiers from user input.
- Strict Content Security Policy; no inline scripts except the built bundle's
  hashes/nonces as required.
- Receipt files use unguessable names, are served only through authenticated
  endpoints, and never get public URLs.
- Dependency and container scanning in CI.
- No analytics, telemetry, or third-party fonts by default.

## 13. Testing strategy

| Layer | Tooling | Coverage focus |
| --- | --- | --- |
| Domain/unit | Go `testing`, table-driven | Signed amounts, balances, budget periods, rollover, transfers, refunds |
| API integration | `httptest` + temporary SQLite | CRUD, filters, pagination, validation, auth, idempotency |
| Contract | OpenAPI validation in tests | Response schemas and status codes match the spec |
| MCP | Official SDK test client | Tool schemas, scopes, draft/confirm flow, error cases |
| WhatsApp | Signed fixture payloads | Verification, bad signatures, dedupe, allowlist, confirm/reject |
| Frontend unit | Vitest + Testing Library | Filters, forms, money formatting, budget progress |
| End-to-end | Playwright | Add transaction, filter, create budget, confirm draft |
| Deployment smoke | Container run + probe checks | Starts, migrates, serves UI, fails closed without config |

- Critical paths target meaningful coverage, not a vanity percentage.
- Golden fixtures cover budget math and representative receipt OCR output.
- Tests must not call real AI or WhatsApp APIs; use fakes and recorded fixtures.
- CI runs formatting, static analysis, unit tests, the frontend build, and the
  container build.

## 14. Performance budgets

| Metric | Target |
| --- | --- |
| Idle RSS (app container) | < 80 MiB |
| Final image size | < 60 MB |
| Cold start to ready | < 1 s with a small database |
| API p95 (local, 10k transactions) | < 100 ms |
| Dashboard initial JS | < 200 KiB gzip |
| Lighthouse mobile performance | ≥ 90 on the dashboard |
| Database size | < 100 MB for 100k transactions, excluding receipts |

These are budgets, not guarantees; measure before optimizing and record
regressions.

## 15. Delivery plan

Aligns with `TASKS.md`.

| Milestone | Outcome |
| --- | --- |
| M0 Foundations | Spec approved, repo scaffolded, CI green, quality gates defined |
| M1 Backend core | Schema, accounts, tags, transactions, transfers, budgets, summaries |
| M2 Web frontend | Dashboard, transaction flows, budgets, charts, simulation, PWA |
| M3 AI/MCP/WhatsApp | MCP tools, draft/confirm flow, WhatsApp bot, receipt OCR, audit |
| M4 Deployment | Image, manifests, Tailscale sidecar, backups, runbook |

Each milestone ends with a runnable, tested increment and an updated `TASKS.md`.

## 16. Open decisions for review

Please confirm or change these before implementation:

1. **Stack:** Go + SQLite + SvelteKit as proposed in §4.
2. **Users:** single user or a small household? This affects auth and data
   ownership (there is no `user_id` in the schema today).
3. **Currency:** one currency per account with a single reporting currency, or a
   stricter single-currency system for v1?
4. **Auth:** tailnet identity only, plus API tokens for agents? Or also a
   passphrase fallback?
5. **AI provider:** OpenAI-compatible by default, Anthropic adapter later, or no
   default at all?
6. **WhatsApp:** is a Cloud API account available now, or should the bot be
   deferred until after the core app ships?
7. **Deployment:** Kubernetes-only, or also a Docker Compose path for a home
   server? (Compose is useful for local dev regardless.)
8. **Offline behavior:** read-only PWA caching is enough?
9. **Backups:** target and retention (S3-compatible, restic repository, or manual
   copies)?
10. **License:** MIT, Apache-2.0, AGPL-3.0, or private/unlicensed?
11. **Repository host/CI:** GitHub Actions, GitLab CI, or something else?
12. **Recommendations:** confirm that v1 uses deterministic rules rather than an
    LLM for spending advice. An LLM summary can be added later.

## 17. Approval

Approval means the maintainer accepts the architecture, stack, data model, API
shape, security posture, and delivery plan, and resolves or explicitly defers
the open decisions above.

- [ ] Approved by: ____________________
- [ ] Date: ____________________
- [ ] Deferred decisions recorded in `TASKS.md`: ____________________
