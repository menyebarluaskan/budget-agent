-- +goose Up
-- +goose StatementBegin
CREATE TABLE workspace_invites (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    created_by TEXT NOT NULL REFERENCES users(id),
    expires_at TEXT NOT NULL,
    redeemed_by TEXT REFERENCES users(id),
    redeemed_at TEXT,
    created_at TEXT NOT NULL
);
CREATE INDEX idx_workspace_invites_workspace ON workspace_invites(workspace_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_workspace_invites_workspace;
DROP TABLE IF EXISTS workspace_invites;
-- +goose StatementEnd