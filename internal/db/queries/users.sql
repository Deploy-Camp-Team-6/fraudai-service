-- name: CreateUser :one
INSERT INTO users (email, plan) VALUES ($1, $2)
RETURNING id, email, plan, created_at, updated_at;

-- name: ListUsersPaged :many
SELECT id, email, plan, created_at FROM users
WHERE id > $1 ORDER BY id ASC LIMIT $2; -- keyset pagination

-- name: GetUserByID :one
SELECT id, email, plan, created_at, updated_at
FROM users
WHERE id = $1;
