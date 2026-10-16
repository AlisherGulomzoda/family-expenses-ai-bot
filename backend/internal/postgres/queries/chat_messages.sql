-- name: AppendChatMessage :exec
INSERT INTO chat_messages (user_id, role, content)
VALUES (@user_id, @role, @content);

-- name: LastChatMessages :many
-- The n most recent messages, oldest first.
SELECT content
FROM (
    SELECT id, content
    FROM chat_messages
    WHERE user_id = @user_id
    ORDER BY id DESC
    LIMIT @row_limit
) recent
ORDER BY id;
