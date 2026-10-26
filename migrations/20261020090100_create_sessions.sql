-- +goose Up
-- Server-side web sessions. The cookie holds only the random id.
CREATE TABLE sessions (
    id         text        PRIMARY KEY,
    user_id    uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    email      text        NOT NULL,
    name       text        NOT NULL,
    roles      jsonb       NOT NULL DEFAULT '[]',
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX sessions_user_id_idx ON sessions (user_id);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

-- +goose Down
DROP TABLE sessions;
