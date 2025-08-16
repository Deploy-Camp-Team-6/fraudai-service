#!/bin/sh
set -e

# If app expects JWT_SECRET env (not _FILE), export it from the secret:
if [ -n "$JWT_SECRET_FILE" ] && [ -f "$JWT_SECRET_FILE" ]; then
  export JWT_SECRET="$(cat "$JWT_SECRET_FILE")"
fi

# Run migrations, then start
migrate -path /root/migrations -database "$PG_DSN" up
exec ./server
