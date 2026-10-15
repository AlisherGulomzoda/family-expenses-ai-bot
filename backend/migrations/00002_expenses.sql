-- +goose Up
CREATE TABLE expenses (
    id             BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id        BIGINT      NOT NULL REFERENCES users (telegram_id),
    -- Money is stored in kopecks to avoid floating point errors.
    amount_kopecks BIGINT      NOT NULL CHECK (amount_kopecks > 0),
    -- Stable English codes; Russian labels live in the bot and the UI.
    category       TEXT        NOT NULL CHECK (category IN (
                                   'food', 'transport', 'housing', 'entertainment',
                                   'health', 'shopping', 'other')),
    spent_at       DATE        NOT NULL,
    comment        TEXT        CHECK (char_length(comment) <= 500),
    -- Where the expense was created. No default: the application always sets it.
    source         TEXT        NOT NULL CHECK (source IN ('bot', 'web')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Maintained by the application in UPDATE statements.
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Bot queries: one user's expenses by period, newest first.
CREATE INDEX expenses_user_spent_at_idx ON expenses (user_id, spent_at DESC, id DESC);
-- Web panel queries: all expenses by period.
CREATE INDEX expenses_spent_at_idx ON expenses (spent_at DESC, id DESC);

-- +goose Down
DROP TABLE expenses;
