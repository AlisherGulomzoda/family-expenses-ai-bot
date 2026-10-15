-- name: CreateExpense :one
WITH ins AS (
    INSERT INTO expenses (user_id, amount_kopecks, category, spent_at, comment, source)
    VALUES (@user_id, @amount_kopecks, @category, @spent_at, sqlc.narg(comment), @source)
    RETURNING *
)
SELECT ins.id, ins.user_id, u.name AS user_name, ins.amount_kopecks, ins.category,
       ins.spent_at, ins.comment, ins.source, ins.created_at, ins.updated_at
FROM ins
JOIN users u ON u.telegram_id = ins.user_id;

-- name: GetExpense :one
SELECT e.id, e.user_id, u.name AS user_name, e.amount_kopecks, e.category,
       e.spent_at, e.comment, e.source, e.created_at, e.updated_at
FROM expenses e
JOIN users u ON u.telegram_id = e.user_id
WHERE e.id = @id
  AND (sqlc.narg(owner)::bigint IS NULL OR e.user_id = sqlc.narg(owner));

-- name: UpdateExpense :one
-- NULL parameters keep the current value; an empty comment clears it.
-- The join with users returns the (new) owner's name. An unknown new_user_id
-- matches no user, so no row is updated: the repository then tells
-- "unknown user" from "not found" with UserExists.
UPDATE expenses e SET
    user_id        = COALESCE(sqlc.narg(new_user_id)::bigint, e.user_id),
    amount_kopecks = COALESCE(sqlc.narg(amount_kopecks)::bigint, e.amount_kopecks),
    category       = COALESCE(sqlc.narg(category)::text, e.category),
    spent_at       = COALESCE(sqlc.narg(spent_at)::date, e.spent_at),
    comment        = CASE WHEN sqlc.narg(comment)::text IS NULL THEN e.comment
                          ELSE NULLIF(sqlc.narg(comment)::text, '') END,
    updated_at     = now()
FROM users u
WHERE e.id = @id
  AND (sqlc.narg(owner)::bigint IS NULL OR e.user_id = sqlc.narg(owner))
  AND u.telegram_id = COALESCE(sqlc.narg(new_user_id)::bigint, e.user_id)
RETURNING e.id, e.user_id, u.name AS user_name, e.amount_kopecks, e.category,
          e.spent_at, e.comment, e.source, e.created_at, e.updated_at;

-- name: DeleteExpense :one
WITH del AS (
    DELETE FROM expenses
    WHERE id = @id
      AND (sqlc.narg(owner)::bigint IS NULL OR user_id = sqlc.narg(owner))
    RETURNING *
)
SELECT del.id, del.user_id, u.name AS user_name, del.amount_kopecks, del.category,
       del.spent_at, del.comment, del.source, del.created_at, del.updated_at
FROM del
JOIN users u ON u.telegram_id = del.user_id;

-- name: ListExpenses :many
-- sort_field and sort_order come from a whitelist in the domain; the CASE
-- expressions keep the query static (no SQL built from strings).
SELECT e.id, e.user_id, u.name AS user_name, e.amount_kopecks, e.category,
       e.spent_at, e.comment, e.source, e.created_at, e.updated_at
FROM expenses e
JOIN users u ON u.telegram_id = e.user_id
WHERE (sqlc.narg(user_id)::bigint IS NULL OR e.user_id = sqlc.narg(user_id))
  AND (sqlc.narg(from_date)::date IS NULL OR e.spent_at >= sqlc.narg(from_date))
  AND (sqlc.narg(to_date)::date IS NULL OR e.spent_at <= sqlc.narg(to_date))
  AND (sqlc.narg(category)::text IS NULL OR e.category = sqlc.narg(category))
  AND (sqlc.narg(query)::text IS NULL OR e.comment ILIKE '%' || sqlc.narg(query) || '%')
ORDER BY
    CASE WHEN @sort_field::text = 'spent_at'   AND @sort_order::text = 'asc'  THEN e.spent_at END ASC,
    CASE WHEN @sort_field::text = 'spent_at'   AND @sort_order::text = 'desc' THEN e.spent_at END DESC,
    CASE WHEN @sort_field::text = 'amount'     AND @sort_order::text = 'asc'  THEN e.amount_kopecks END ASC,
    CASE WHEN @sort_field::text = 'amount'     AND @sort_order::text = 'desc' THEN e.amount_kopecks END DESC,
    CASE WHEN @sort_field::text = 'created_at' AND @sort_order::text = 'asc'  THEN e.created_at END ASC,
    CASE WHEN @sort_field::text = 'created_at' AND @sort_order::text = 'desc' THEN e.created_at END DESC,
    -- Stable pagination: ties are broken by id in the same direction.
    CASE WHEN @sort_order::text = 'asc'  THEN e.id END ASC,
    CASE WHEN @sort_order::text = 'desc' THEN e.id END DESC
LIMIT @row_limit OFFSET @row_offset;

-- name: CountExpenses :one
SELECT count(*)
FROM expenses e
WHERE (sqlc.narg(user_id)::bigint IS NULL OR e.user_id = sqlc.narg(user_id))
  AND (sqlc.narg(from_date)::date IS NULL OR e.spent_at >= sqlc.narg(from_date))
  AND (sqlc.narg(to_date)::date IS NULL OR e.spent_at <= sqlc.narg(to_date))
  AND (sqlc.narg(category)::text IS NULL OR e.category = sqlc.narg(category))
  AND (sqlc.narg(query)::text IS NULL OR e.comment ILIKE '%' || sqlc.narg(query) || '%');

-- name: SummarizeExpenses :many
-- Totals per category, largest first; equal totals keep the display order.
SELECT e.category,
       sum(e.amount_kopecks)::bigint AS total_kopecks,
       count(*)                      AS expense_count
FROM expenses e
WHERE (sqlc.narg(user_id)::bigint IS NULL OR e.user_id = sqlc.narg(user_id))
  AND (sqlc.narg(from_date)::date IS NULL OR e.spent_at >= sqlc.narg(from_date))
  AND (sqlc.narg(to_date)::date IS NULL OR e.spent_at <= sqlc.narg(to_date))
  AND (sqlc.narg(category)::text IS NULL OR e.category = sqlc.narg(category))
  AND (sqlc.narg(query)::text IS NULL OR e.comment ILIKE '%' || sqlc.narg(query) || '%')
GROUP BY e.category
ORDER BY total_kopecks DESC,
         array_position(
             ARRAY['food', 'transport', 'housing', 'entertainment', 'health', 'shopping', 'other']::text[],
             e.category);
