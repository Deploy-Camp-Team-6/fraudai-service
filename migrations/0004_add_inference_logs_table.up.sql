-- 0004_add_inference_logs_table.up.sql
CREATE TABLE inference_logs (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  api_key_id BIGINT REFERENCES api_keys(id),
  request_payload JSONB NOT NULL,
  response_payload JSONB,
  error TEXT,
  request_time TIMESTAMPTZ NOT NULL,
  response_time TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON inference_logs (user_id);
CREATE INDEX ON inference_logs (api_key_id);
