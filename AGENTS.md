# AGENTS.md

Guidance for AI agents and human contributors working in this repository.

## 1. Project summary

**Integrated Personal Budget** is a self-hosted personal budgeting system. The
target deployment is a rootless Podman pod managed by systemd Quadlet units,
containing:

- one application container (HTTP API, web UI, and MCP endpoint), and
- one Tailscale sidecar container for tailnet-only access.

The product combines budgeting, transaction tracking, an AI assistant, receipt
OCR, and a mobile-friendly web UI. See `README.md` for the product overview and
`docs/TECH_SPEC.md` for the proposed architecture.

## 2. Current phase: implementation

The technical specification was approved on 2026-11-02. Implementation follows
`TASKS.md` and the milestone order in the spec.

- Work only on the active task or a clearly scoped follow-up.
- Do not start a later milestone without maintainer approval.
- Do not introduce a stack or architecture change without updating the spec.
- If a task needs a deferred decision, stop and ask.
- Add tests with the code; scaffolding is the only exception.

The current approval gate is no longer T-005; it is the next unstarted task in
`TASKS.md`.

## 3. Source of truth

| Document | Purpose |
| --- | --- |
| `README.md` | Product overview, features, and how to use the project |
| `docs/TECH_SPEC.md` | Architecture, stack, data model, API, deployment (proposed) |
| `TASKS.md` | Live progress tracker and current work queue |
| `AGENTS.md` | Working rules and conventions (this file) |

If documents disagree, the maintainer's latest explicit instruction wins.
Otherwise, prefer `docs/TECH_SPEC.md` for technical details and update the
affected document in the same change.

## 4. Working agreement

Follow this loop for every task:

1. Pick one task from `TASKS.md` and mark it `[~]` (in progress).
2. Read the relevant sections of `README.md` and `docs/TECH_SPEC.md` first.
3. Make the smallest coherent change that completes the task.
4. Run the relevant checks (see §7). If no toolchain exists yet, say so.
5. Update `TASKS.md`: mark the task `[x]`, or `[!]` with a blocker note.
6. Commit the task with a Conventional Commit message (§6).
7. Always uses the provided agentic editor tool for updating files, don't use script!

One logical task per commit. Do not bundle unrelated changes. Do not commit
generated secrets, local databases, `node_modules`, build output, or editor
state.

## 5. Engineering principles

- **Small and boring by default.** Low memory, small image, standard library
  first. Add a dependency only when it clearly earns its maintenance cost.
- **API-first.** The HTTP API is the product boundary. Keep `docs/TECH_SPEC.md`
  and the OpenAPI contract in sync with handlers.
- **Money is never a float.** Store and compute amounts as integer minor units
  (for example, cents). Keep currency explicit at the account boundary.
- **Single-writer persistence.** The planned SQLite database assumes one
  application replica. Do not scale the app horizontally without revisiting the
  storage design.
- **Tailnet-only exposure.** No public ingress. Do not weaken network,
  authentication, or webhook-signature checks for convenience.
- **AI proposes, humans dispose.** AI-generated transactions and receipt data
  must be shown as drafts and confirmed before they are persisted.
- **Mobile-first UI.** Design for a narrow viewport first, then enhance.
- **Accessible and fast.** Keyboard usable, sufficient contrast, no layout shift;
  respect performance budgets in the spec.
- **Privacy.** Receipts and financial history are sensitive. Do not send them to
  third parties, log them verbatim, or add analytics without explicit approval.

## 6. Git conventions

Use [Conventional Commits](https://www.conventionalcommits.org/):

```text
<type>(<scope>): <short imperative summary>

[optional body]
[optional footer]
```

Allowed types: `feat`, `fix`, `docs`, `refactor`, `test`, `build`, `ci`,
`chore`, `perf`, `revert`.

Examples:

- `docs: add technical specification for review`
- `docs(tasks): mark specification task complete`
- `feat(api): add transaction create endpoint`
- `fix(budget): exclude refunds from expense totals`
- `test(mcp): cover receipt tool validation`

Rules:

- Write the subject in the imperative mood, lowercase, no trailing period.
- Keep the subject at or under 72 characters.
- Explain *why* in the body when the change is not obvious.
- Reference a task ID from `TASKS.md` in the body when useful, for example
  `Task: T-012`.
- Never rewrite or force-push shared history. Do not amend commits that the
  maintainer may have based work on.
- Do not commit unless the maintainer asked for the work to be committed or the
  task instructions say to. This repository's current workflow commits every
  completed task.

## 7. Commands

The project uses a `Makefile` as the canonical entry point. Commands run from
the repository root.

| Command | Purpose |
| --- | --- |
| `make help` | List available targets. |
| `make dev` | Run the API and SvelteKit dev server together. |
| `make dev-api` | Run the Go API with local defaults (`127.0.0.1:8080`). |
| `make dev-web` | Run the SvelteKit dev server on `127.0.0.1:5173`. |
| `make web-install` | Install frontend dependencies with `pnpm`. |
| `make build` | Build the static frontend and compile the Go binary to `bin/budget-agent`. |
| `make test` | Run Go and frontend unit tests. |
| `make fmt` | Format Go sources and the frontend. |
| `make lint` | Run `go vet` and frontend linting/format checks. |
| `make web-check` | Type-check the SvelteKit app. |
| `make check` | Run the full local quality gate (lint, type check, tests, build). |
| `make clean` | Remove generated build output. |

Quality gates:

- Backend: `gofmt`, `go vet`, unit tests (`go test ./...`).
- Frontend: type check (`svelte-check`), lint/format (`prettier`, `eslint`), unit tests (`vitest`), production build (`vite build`).
- CI: `.github/workflows/ci.yml` runs `make check` on pushes to `main` and on pull requests.
- Integration/E2E tests land with their milestones.

## 8. Security rules

- Never commit secrets, tokens, private keys, real phone numbers, or personal
  financial data. Use `.env.example` with placeholder values.
- Load configuration from environment variables; secret values come from a
  `0600` environment file outside the repository or from Podman secrets.
- Keep Quadlet units free of secrets. The real environment file lives at
  `~/.config/budget-agent/budget-agent.env`, not in the repository.
- Quadlet units are symlinked into `~/.config/containers/systemd/`. Do not
  duplicate them or edit the generated units; edit the source files under
  `deploy/quadlet/` and rerun `systemctl --user daemon-reload`.
- Use parameterized SQL only. Never build SQL with string concatenation.
- Validate and bound every external input: webhook bodies, uploads, query
  filters, and AI output.
- Hash API tokens at rest and compare them in constant time.
- Verify WhatsApp webhook signatures before processing.
- Enforce an explicit phone-number allowlist for the WhatsApp bot.
- Keep cookie sessions `HttpOnly`, `Secure`, and `SameSite=Lax` (or stricter).
- Do not enable permissive CORS, debug endpoints, or stack traces in production.
- Flag any new dependency that needs network access, telemetry, or elevated
  privileges in the pull request.

## 9. Writing and documentation

- Prefer short sentences, active voice, and concrete examples.
- Update `README.md` for user-visible changes (features, setup, usage).
- Update `docs/TECH_SPEC.md` for architecture, stack, schema, API, or deployment
  changes. Record contested decisions as an ADR under `docs/adr/` once the
  project grows.
- Keep `TASKS.md` current. Stale task lists are treated as bugs.
- Mark proposals clearly as **proposed** until approved. Do not present an
  unapproved decision as final.

## 10. Definition of done

A task is done only when:

- [ ] The change works and has been exercised (test, manual run, or documented
      reason why neither is possible yet).
- [ ] Relevant checks pass, or failures are explained in the task/comments.
- [ ] Tests cover new behavior on critical paths.
- [ ] User-facing and architecture documentation is updated when needed.
- [ ] `TASKS.md` reflects the new status.
- [ ] The change is committed with a semantic message, if committing is in scope.
- [ ] No secrets, debug artifacts, or unrelated edits are included.

## 11. Blocked or ambiguous work

If a task conflicts with this file, the spec, or an earlier maintainer
instruction:

1. Stop before making an irreversible change.
2. State the conflict and the options concisely.
3. Ask the maintainer to choose.
4. Record the decision in `TASKS.md` or the spec once made.
