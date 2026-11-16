#!/bin/bash
# Container entrypoint: migrate, then run the worker in the background and the API in front.
#
# Environment:
#   DATABASE_URL          the API's connection (owner role); also runs the migrations
#   WORKER_DATABASE_URL   the worker's connection (gee_worker role)
#   GEE_CREDENTIALS_JSON  the Earth Engine service account key, as JSON text
#   GEE_PROJECT, JWT_SECRET, PORT and the rest as in the .env.example files
set -euo pipefail
cd /app

if [ -n "${GEE_CREDENTIALS_JSON:-}" ]; then
  key_file=$(mktemp)
  chmod 600 "$key_file"
  printf '%s' "$GEE_CREDENTIALS_JSON" > "$key_file"
  export GOOGLE_APPLICATION_CREDENTIALS="$key_file"
  unset GEE_CREDENTIALS_JSON
fi

./bin/migrate

# The worker restarts on its own if it crashes; the API keeps serving meanwhile.
(
  cd worker
  while true; do
    DATABASE_URL="${WORKER_DATABASE_URL:?WORKER_DATABASE_URL is required}" python -m app.worker || true
    echo "worker exited; restarting in 10s" >&2
    sleep 10
  done
) &

exec ./bin/api
