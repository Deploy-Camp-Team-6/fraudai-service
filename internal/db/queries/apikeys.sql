-- name: GetAPIKeyByHash :one
SELECT id, user_id, key_hash, active, rate_rpm FROM api_keys WHERE key_hash = $1;

-- name: CreateAPIKey :one
INSERT INTO api_keys (user_id, key_hash, label, rate_rpm)
VALUES ($1, $2, $3, $4)
RETURNING id, user_id, label, active, rate_rpm, created_at;

-- name: ListAPIKeysByUser :many
SELECT id, label, active, rate_rpm, created_at FROM api_keys WHERE user_id = $1;

-- name: DeleteAPIKey :exec
UPDATE api_keys SET active = FALSE WHERE user_id = $1 AND id = $2;
