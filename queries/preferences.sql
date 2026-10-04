-- name: CreatePreferences :one
INSERT INTO preferences (user_id, apt_groups, digest_enabled, delivery_time)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetPreferencesByUserID :one
SELECT * FROM preferences WHERE user_id = $1;

-- name: UpdateAptGroupsOnly :execresult
UPDATE preferences
SET apt_groups = $2
WHERE user_id = $1;

-- null arguments leave their column untouched
-- name: UpdateDigestSettings :execresult
UPDATE preferences
SET digest_enabled = COALESCE(sqlc.narg(enabled)::boolean, digest_enabled),
    delivery_time = COALESCE(sqlc.narg(delivery_time)::time, delivery_time)
WHERE user_id = sqlc.arg(user_id);

-- name: ListUsersForDelivery :many
-- half-open window: a late tick still lands, but each time stays due once
SELECT u.telegram_id, p.apt_groups
FROM users u
JOIN preferences p ON u.id = p.user_id
WHERE p.digest_enabled = true
  AND p.delivery_time IS NOT NULL
  AND p.delivery_time > date_trunc('minute', NOW() AT TIME ZONE 'UTC')::time - INTERVAL '1 minute'
  AND p.delivery_time <= date_trunc('minute', NOW() AT TIME ZONE 'UTC')::time;
