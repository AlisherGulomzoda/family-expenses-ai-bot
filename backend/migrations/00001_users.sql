-- +goose Up
-- Family members. A row is created when a person first writes to the bot.
CREATE TABLE users (
    telegram_id BIGINT      PRIMARY KEY,
    name        TEXT        NOT NULL,
    username    TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE users;
