---
name: restart-server
description: "Restart the dflh-saf-v2 Go backend server. Use when the user says '서버 재시작', 'restart server', 'restart backend', '서버 다시 시작', '서버 재실행', or needs to kill and restart the Go dev server on port 8080."
---

# Restart Server

Kill any existing process on port 8080 and restart the Go backend dev server.

## Usage

Run the restart script:

```bash
bash <skill-dir>/scripts/restart.sh <project-root> [port]
```

- `<project-root>`: Absolute path to the dflh-saf-v2 monorepo root (contains `backend/` directory)
- `[port]`: Optional, defaults to `8080`

The script will:
1. Find and kill any process on the target port
2. Start `go run ./cmd/server` in the background
3. Verify the server started successfully
4. Print the PID and log file path (`/tmp/dflh-saf-server.log`)

## Checking Logs

If the server fails to start, check the log:

```bash
tail -50 /tmp/dflh-saf-server.log
```

Common failures: DB connection refused (MariaDB not running), port still occupied.
