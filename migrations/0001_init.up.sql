-- 0001_init.sql
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE users (
  id           BIGSERIAL PRIMARY KEY,
  email        CITEXT UNIQUE NOT NULL,
  password_hash TEXT, -- optional if you issue JWTs
  plan         TEXT NOT NULL DEFAULT 'free',
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE api_keys (
  id          BIGSERIAL PRIMARY KEY,
  user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  key_hash    BYTEA NOT NULL, -- store hash of API key
  label       TEXT,
  active      BOOLEAN NOT NULL DEFAULT true,
  rate_rpm    INT NOT NULL DEFAULT 100,
  last_used_at TIMESTAMPTZ,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON api_keys (user_id);
CREATE INDEX ON api_keys (key_hash);