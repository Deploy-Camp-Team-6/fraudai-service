-- name: CreateInferenceLog :exec
INSERT INTO inference_logs (
  user_id, api_key_id, request_payload, response_payload, error, request_time, response_time
) VALUES (
  $1, $2, $3, $4, $5, $6, $7
);
