#!/bin/sh
set -e
migrate -path /root/migrations -database "$PG_DSN" up
exec ./server
