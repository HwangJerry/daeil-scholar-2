#!/bin/bash
# restart.sh — Kill existing process on target port and restart the Go backend server.
# Usage: restart.sh <project-root> [port]

set -e

PROJECT_ROOT="${1:-.}"
PORT="${2:-8080}"

# Kill existing process on the port
PIDS=$(lsof -ti:"$PORT" 2>/dev/null || true)
if [ -n "$PIDS" ]; then
  echo "Killing process(es) on port $PORT: $PIDS"
  echo "$PIDS" | xargs kill -9 2>/dev/null || true
  sleep 1
fi

# Verify port is free
if lsof -ti:"$PORT" >/dev/null 2>&1; then
  echo "ERROR: Port $PORT still in use after kill attempt"
  exit 1
fi

# Load .env.local if present (export KEY=VALUE lines, skip comments)
cd "$PROJECT_ROOT/backend"
if [ -f .env.local ]; then
  echo "Loading .env.local..."
  set -a
  while IFS= read -r line || [ -n "$line" ]; do
    # Skip blank lines and comment-only lines
    [[ "$line" =~ ^[[:space:]]*# ]] && continue
    [[ -z "${line// }" ]] && continue
    eval "export $line" 2>/dev/null || true
  done < .env.local
  set +a
fi

# Start the Go backend server
echo "Starting server on port $PORT..."
nohup go run ./cmd/server > /tmp/dflh-saf-server.log 2>&1 &
SERVER_PID=$!

# Wait briefly and check if server started
sleep 2
if kill -0 "$SERVER_PID" 2>/dev/null; then
  echo "Server started (PID: $SERVER_PID). Log: /tmp/dflh-saf-server.log"
else
  echo "ERROR: Server failed to start. Last 20 lines of log:"
  tail -20 /tmp/dflh-saf-server.log 2>/dev/null || echo "(no log output)"
  exit 1
fi
