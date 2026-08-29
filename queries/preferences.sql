-- name: CreatePreferences :one
INSERT INTO preferences (user_id, apt_groups, digest_enabled, delivery_time)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetPreferencesByUserID :one
SELECT * FROM preferences WHERE user_id = $1;

-- name: UpdatePreferences :one
UPDATE preferences
SET apt_groups = $2, digest_enabled = $3, delivery_time = $4
WHERE user_id = $1
RETURNING *;

-- name: ListUsersForDelivery :many
SELECT u.telegram_id, p.apt_groups
FROM users u
JOIN preferences p ON u.id = p.user_id
WHERE p.digest_enabled = true
  AND p.delivery_time IS NOT NULL
  AND p.delivery_time = CURRENT_TIME(0);
