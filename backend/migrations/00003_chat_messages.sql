-- +goose Up
-- Agent conversation memory: user messages, assistant replies (with tool calls)
-- and tool results, in the order they happened.
CREATE TABLE chat_messages (
    id         BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES users (telegram_id) ON DELETE CASCADE,
    role       TEXT        NOT NULL CHECK (role IN ('user', 'assistant', 'tool')),
    content    JSONB       NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- "Last N messages of a user".
CREATE INDEX chat_messages_user_id_idx ON chat_messages (user_id, id DESC);

-- +goose Down
DROP TABLE chat_messages;
