-- name: CreateUser :one
INSERT INTO users (name, email, password_hash, plan) VALUES ($1, $2, $3, $4)
RETURNING id, name, email, plan, created_at, updated_at;

-- name: ListUsersPaged :many
SELECT id, email, plan, created_at FROM users
WHERE id > $1 ORDER BY id ASC LIMIT $2; -- keyset pagination

-- name: GetUserByID :one
SELECT id, name, email, plan, created_at, updated_at
FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, name, email, plan, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByEmailForLogin :one
SELECT id, name, email, password_hash, plan, created_at, updated_at
FROM users
WHERE email = $1;
