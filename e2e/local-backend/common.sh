#!/usr/bin/env bash
# Shared local-only preflight and explicit Compose invocation.
set -euo pipefail
LOCAL_STACK_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly LOCAL_STACK_DIR

compose() {
    env -u COMPOSE_FILE -u COMPOSE_ENV_FILES -u COMPOSE_PROFILES BUILDX_BUILDER=default \
        docker compose --project-name dflh-local-e2e \
        --project-directory "$LOCAL_STACK_DIR" --env-file "$LOCAL_STACK_DIR/local.env" \
        -f "$LOCAL_STACK_DIR/compose.yaml" "$@"
}

preflight() {
    if [[ $# -ne 0 ]]; then
        echo 'These scripts take no arguments or URL overrides.' >&2
        exit 1
    fi
    python3 "$LOCAL_STACK_DIR/guard.py"
    if ! command -v docker >/dev/null; then
        echo 'SKIP: Docker CLI/daemon unavailable; no stack or tests started.'
        exit 0
    fi
    local endpoint
    endpoint="$(docker context inspect --format '{{.Endpoints.docker.Host}}')"
    if [[ -z "${DOCKER_CONTEXT:-}" ]]; then
        endpoint="${DOCKER_HOST:-$endpoint}"
    fi
    case "$endpoint" in
        unix://*) ;;
        *) echo 'Refusing a non-local Docker endpoint; select a local Unix-socket context.' >&2; exit 1 ;;
    esac
    if ! docker info --format '{{.ServerVersion}}' >/dev/null 2>&1; then
        echo 'SKIP: Docker CLI/daemon unavailable; no stack or tests started.'
        exit 0
    fi
    compose config --format json | python3 "$LOCAL_STACK_DIR/guard.py" --compose
}
