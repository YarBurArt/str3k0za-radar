-- name: CreateUser :one
INSERT INTO users (telegram_id, username)
VALUES ($1, $2)
RETURNING *;

-- name: GetUserByTelegramID :one
SELECT * FROM users WHERE telegram_id = $1;
