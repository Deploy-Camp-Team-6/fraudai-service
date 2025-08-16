#!/bin/sh
set -euo pipefail

# If app expects JWT_SECRET (not *_FILE), export it from the mounted Docker secret file.
if [ "${JWT_SECRET_FILE:-}" != "" ] && [ -f "$JWT_SECRET_FILE" ]; then
  export JWT_SECRET="$(cat "$JWT_SECRET_FILE")"
fi

# Require PG_DSN unless explicitly skipping migrations
if [ "${MIGRATE:-1}" != "0" ]; then
  if [ "${PG_DSN:-}" = "" ]; then
    echo "ERROR: PG_DSN is not set and MIGRATE!=0" >&2
    exit 1
  fi

  # Run migrations (idempotent). Using file:// source is recommended.
  echo "Running database migrations..."
  migrate -path /root/migrations -database "$PG_DSN" -verbose up
else
  echo "Skipping migrations (MIGRATE=0)"
fi

echo "Starting server..."
exec ./server
