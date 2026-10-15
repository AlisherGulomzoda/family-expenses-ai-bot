-- name: UpsertUser :one
INSERT INTO users (telegram_id, name, username)
VALUES (@telegram_id, @name, sqlc.narg(username))
ON CONFLICT (telegram_id) DO UPDATE
    SET name = EXCLUDED.name, username = EXCLUDED.username
RETURNING telegram_id, name, username;

-- name: UserExists :one
SELECT EXISTS (SELECT 1 FROM users WHERE telegram_id = @telegram_id);

-- name: ListUsers :many
SELECT telegram_id, name, username
FROM users
ORDER BY name, telegram_id;
